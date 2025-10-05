package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/hamidoujand/jumble/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"

	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
)

// Config defines the information needed to init tracing.
type Config struct {
	ServiceName    string
	Host           string
	ExcludedRoutes map[string]struct{}
	Probability    float64
}

func NewTraceProvider(log logger.Logger, cfg Config) (trace.TracerProvider, func(ctx context.Context), error) {
	//otel exporter
	exporter, err := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(), // This should be configurable
			otlptracegrpc.WithEndpoint(cfg.Host),
		),
	)

	if err != nil {
		return nil, nil, fmt.Errorf("creating new exporter: %w", err)
	}

	//resource
	resOpt := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(cfg.ServiceName),
	)
	resource := sdktrace.WithResource(resOpt)

	//batcher
	batcherLimitOpt := sdktrace.WithMaxExportBatchSize(sdktrace.DefaultMaxExportBatchSize)
	batcherTimoutOpt := sdktrace.WithBatchTimeout(sdktrace.DefaultScheduleDelay * time.Millisecond)

	batcherOpt := sdktrace.WithBatcher(exporter, batcherLimitOpt, batcherTimoutOpt)

	//sampler
	samplerOpt := sdktrace.WithSampler(newEndpointExcluder(cfg.ExcludedRoutes, cfg.Probability))

	//init provider
	provider := sdktrace.NewTracerProvider(batcherOpt, samplerOpt, resource)
	teardown := func(ctx context.Context) {
		provider.Shutdown(ctx)
	}

	//set this provider as global trace provider
	otel.SetTracerProvider(provider)

	//use a customized propagator
	//For distributed systems, combining TraceContext and Baggage is common because:
	// TraceContext ensures trace information flows between services.
	// Baggage allows custom key-value metadata to propagate for additional context.
	compositeMapPropgator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(compositeMapPropgator)
	return provider, teardown, nil
}

// ==============================================================================
func AddSpan(ctx context.Context, spanName string, keyvalues ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer, ok := ctx.Value(tracerKey).(trace.Tracer)
	if !ok || tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}

	ctx, span := tracer.Start(ctx, spanName)
	span.SetAttributes(keyvalues...)

	return ctx, span
}

//==============================================================================
//Custom Sampler

type endpointExcluder struct {
	endpoints   map[string]struct{}
	probability float64
}

func newEndpointExcluder(endpoints map[string]struct{}, probability float64) endpointExcluder {
	return endpointExcluder{
		endpoints:   endpoints,
		probability: probability,
	}
}

func endpoint(parameters sdktrace.SamplingParameters) string {
	var path, query string

	for _, attr := range parameters.Attributes {
		switch attr.Key {
		case "url.path":
			path = attr.Value.AsString()
		case "url.query":
			query = attr.Value.AsString()
		}
	}

	switch {
	case path == "":
		return ""

	case query == "":
		return path

	default:
		return fmt.Sprintf("%s?%s", path, query)
	}
}

// ShouldSample implements the sampler interface. It prevents the specified
// endpoints from being added to the trace.
func (ee endpointExcluder) ShouldSample(parameters sdktrace.SamplingParameters) sdktrace.SamplingResult {
	if ep := endpoint(parameters); ep != "" {
		if _, exists := ee.endpoints[ep]; exists {
			return sdktrace.SamplingResult{Decision: sdktrace.Drop}
		}
	}

	return sdktrace.TraceIDRatioBased(ee.probability).ShouldSample(parameters)
}

// Description implements the sampler interface.
func (endpointExcluder) Description() string {
	return "customSampler"
}
