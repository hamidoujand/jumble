package otel

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/jumble/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type Config struct {
	// Name of your service in traces
	ServiceName string
	// HTTP routes that shouldn't be traced (like health checks)
	ExcludedRoutes map[string]struct{}
	// OpenTelemetry collector address
	Host string
	// Sampling rate (0.0-1.0) - what percentage of traces to keep
	Probability float64
}

func InitTracing(log logger.Logger, cfg Config) (trace.TracerProvider, func(context.Context), error) {
	//setup the grpc exporter
	exporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),         // Use insecure connection (no TLS)
			otlptracegrpc.WithEndpoint(cfg.Host), // Set collector endpoint
		),
	)

	if err != nil {
		return nil, nil, fmt.Errorf("creating new exporter: %w", err)
	}

	var traceProvider trace.TracerProvider

	// Default teardown function that does nothing
	teardown := func(ctx context.Context) {}

	// Decide which tracer provider to use based on configuration
	switch cfg.Host {
	case "":
		// If no host provided, use NOOP (No Operation) tracer - traces are discarded
		log.Info(context.Background(), "OTEL Setup", "provider", "NOOP")
		traceProvider = noop.NewTracerProvider()
	default:
		// Use real OpenTelemetry tracer that sends data to collector
		log.Info(context.Background(), "OTEL Setup", "provider", cfg.Host)
		tp := sdktrace.NewTracerProvider(
			// Sampler setup - decides which traces to keep
			sdktrace.WithSampler(sdktrace.ParentBased(endpointExcluder{endpoints: cfg.ExcludedRoutes, probability: cfg.Probability})),

			// Batcher setup - batches traces for efficient exporting
			sdktrace.WithBatcher(
				exporter,
				sdktrace.WithMaxExportBatchSize(sdktrace.DefaultMaxExportBatchSize),
				sdktrace.WithBatchTimeout(sdktrace.DefaultScheduleDelay*time.Millisecond),
			),

			// Resource setup - describes the service
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				//  Sets service name in traces
				semconv.ServiceNameKey.String(cfg.ServiceName),
			)),
		)

		// Teardown function that properly shuts down the tracer provider
		teardown = func(ctx context.Context) {
			tp.Shutdown(ctx) // Flushes any pending traces and cleans up
		}

		traceProvider = tp
	}

	// Set this provider as the global one - can be accessed via otel.GetTracerProvider()
	otel.SetTracerProvider(traceProvider)

	// Setup propagation - how trace context is passed between services
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, // W3C Trace Context standard
		propagation.Baggage{},      // Additional key-value data
	))

	return traceProvider, teardown, nil
}

// =============================================================================
func InjectTracing(ctx context.Context, tracer trace.Tracer) context.Context {
	// Store the tracer in the context for later use
	ctx = setTracer(ctx, tracer)

	// Get the current trace ID from the active span
	traceID := trace.SpanFromContext(ctx).SpanContext().SpanID().String()
	// If no active span (default trace ID), generate a new one
	if traceID == defaultTraceID {
		traceID = uuid.NewString()
	}

	// Store the trace ID in the context
	ctx = setTraceID(ctx, traceID)
	return ctx
}

// ==============================================================================
func AddSpan(ctx context.Context, spanName string, keyVals ...attribute.KeyValue) (context.Context, trace.Span) {
	// Retrieve the tracer from context that was set by InjectTracing
	tracer, ok := ctx.Value(tracerKey).(trace.Tracer)

	// If no tracer found, return current context and span (no new span created)
	if !ok || tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	// Start a new child span with the given name
	ctx, span := tracer.Start(ctx, spanName)
	// Add custom attributes (key-value pairs) to the span
	span.SetAttributes(keyVals...)

	return ctx, span
}

// ==============================================================================
func AddTraceToRequest(ctx context.Context, r *http.Request) {
	// Create a carrier that uses HTTP headers to transport trace context
	hc := propagation.HeaderCarrier(r.Header)
	// Inject the current trace context into the HTTP headers
	otel.GetTextMapPropagator().Inject(ctx, hc)
}

//==============================================================================

type endpointExcluder struct {
	endpoints   map[string]struct{} // Routes to exclude from tracing
	probability float64             // Sampling rate for other routes
}

// ShouldSample implements the sampler interface.
func (ee endpointExcluder) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	// Extract the endpoint from span attributes
	ep := endpoint(parameters)
	// Check if this endpoint should be excluded
	if _, ok := ee.endpoints[ep]; ok {
		// Decision: Drop - don't trace this request
		return sdktrace.SamplingResult{Decision: sdktrace.Drop}
	}

	// For non-excluded endpoints, use probability-based sampling
	return sdktrace.TraceIDRatioBased(ee.probability).ShouldSample(parameters)
}

// Description implements the sampler interface.
func (endpointExcluder) Description() string {
	return "customSampler"
}

// Helper to extract endpoint from sampling parameters
func endpoint(parameters sdktrace.SamplingParameters) string {
	var path string
	var query string

	// Look through span attributes to find URL path and query
	for _, attr := range parameters.Attributes {
		switch attr.Key {
		case "url.path":
			path = attr.Value.AsString()
		case "url.query":
			query = attr.Value.AsString()
		}
	}

	// Reconstruct the full endpoint path?query
	switch {
	case path == "":
		return ""
	case query == "":
		return path
	default:
		return fmt.Sprintf("%s?%s", path, query)
	}

}
