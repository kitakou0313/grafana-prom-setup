package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// service-a starts the trace and exports its own span to tempo1, then calls
// service-b over HTTP, propagating the trace context via the traceparent
// header. service-b exports its span to a different Tempo backend (tempo2).
const tempo1Endpoint = "localhost:4317"

func main() {
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
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	tracer := otel.Tracer("service-a")
	spanCtx, span := tracer.Start(ctx, "service-a.request")

	client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	req, err := http.NewRequestWithContext(spanCtx, http.MethodGet, "http://localhost:8081/handle", nil)
	if err != nil {
		log.Fatalf("failed to build request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("request to service-b failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	log.Printf("response from service-b: %s", body)

	traceID := span.SpanContext().TraceID().String()
	span.End()

	flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := tp.ForceFlush(flushCtx); err != nil {
		log.Printf("flush error: %v", err)
	}
	if err := tp.Shutdown(flushCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	log.Printf("TRACE_ID=%s", traceID)
}
