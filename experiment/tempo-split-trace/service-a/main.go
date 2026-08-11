package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// service-a receives requests from the (uninstrumented) client, which makes
// the span created here the root of the trace. It then calls service-b three
// times in sequence, once per fixed-status endpoint (200/400/500),
// propagating the trace context via the traceparent header on each call.
// service-a's own span is exported to a different Tempo backend (tempo1)
// than service-b's.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	tempo1Endpoint := getenv("TEMPO1_ENDPOINT", "localhost:4317")
	serviceBBaseURL := getenv("SERVICE_B_BASE_URL", "http://localhost:8081")
	listenAddr := getenv("LISTEN_ADDR", ":8080")

	ctx := context.Background()

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(tempo1Endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("failed to create exporter: %v", err)
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName("service-a"),
	))
	if err != nil {
		log.Fatalf("failed to create resource: %v", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(time.Second)),
		sdktrace.WithResource(res),
	)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			log.Printf("tracer shutdown error: %v", err)
		}
	}()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	endpoints := []string{"/handle/200", "/handle/400", "/handle/500"}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := trace.SpanFromContext(r.Context()).SpanContext().TraceID().String()

		var results []string
		for _, ep := range endpoints {
			req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, serviceBBaseURL+ep, nil)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			results = append(results, fmt.Sprintf("%s -> status=%d body=%s", ep, resp.StatusCode, strings.TrimSpace(string(body))))
		}

		fmt.Fprintf(w, "service-a handled request, called service-b 3 times:\n%s\nTRACE_ID=%s\n", strings.Join(results, "\n"), traceID)
	})

	mux := http.NewServeMux()
	mux.Handle("/handle", otelhttp.NewHandler(handler, "service-a.handle"))

	log.Println("service-a listening on", listenAddr, "exporting spans to tempo1 via", tempo1Endpoint, "calling service-b at", serviceBBaseURL)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
