package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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
// the span created here the root of the trace. It then calls service-b,
// propagating the trace context via the traceparent header. service-a's own
// span is exported to a different Tempo backend (tempo1) than service-b's.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	tempo1Endpoint := getenv("TEMPO1_ENDPOINT", "localhost:4317")
	serviceBURL := getenv("SERVICE_B_URL", "http://localhost:8081/handle")
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

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := trace.SpanFromContext(r.Context()).SpanContext().TraceID().String()

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, serviceBURL, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		fmt.Fprintf(w, "service-a handled request, called service-b: %s\nTRACE_ID=%s\n", body, traceID)
	})

	mux := http.NewServeMux()
	mux.Handle("/handle", otelhttp.NewHandler(handler, "service-a.handle"))

	log.Println("service-a listening on", listenAddr, "exporting spans to tempo1 via", tempo1Endpoint, "calling service-b at", serviceBURL)
	if err := http.ListenAndServe(listenAddr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
