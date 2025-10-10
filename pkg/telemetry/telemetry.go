package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/trace/noop"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// Config defines the information needed to init tracing.
type Config struct {
	ServiceName    string
	Host           string
	ExcludedRoutes map[string]struct{}
	Probability    float64
	Build          string
}

func SetupOTelSDK(cfg Config) (func(context.Context) error, error) {
	shutdown := func(ctx context.Context) error { return nil }
	var provider trace.TracerProvider

	switch cfg.Host {
	case "":
		//for tests
		provider = noop.NewTracerProvider()
	default:
		//for dev & prod
		exporter, err := otlptrace.New(
			context.Background(),
			otlptracegrpc.NewClient(
				otlptracegrpc.WithInsecure(), // This should be configurable
				otlptracegrpc.WithEndpoint(cfg.Host),
			),
		)

		if err != nil {
			return nil, fmt.Errorf("creating new exporter: %w", err)
		}

		tp := sdktrace.NewTracerProvider(
			//sampler
			sdktrace.WithSampler(sdktrace.ParentBased(newEndpointExcluder(cfg.ExcludedRoutes, cfg.Probability))),
			//batcher
			sdktrace.WithBatcher(exporter,
				sdktrace.WithMaxExportBatchSize(sdktrace.DefaultMaxExportBatchSize),
				sdktrace.WithBatchTimeout(sdktrace.DefaultScheduleDelay*time.Millisecond),
			),
			//resource
			sdktrace.WithResource(
				resource.NewWithAttributes(
					semconv.SchemaURL,
					semconv.ServiceNameKey.String(cfg.ServiceName),
					semconv.ServiceVersionKey.String(cfg.Build),
				),
			),
		)

		shutdown = func(ctx context.Context) error {
			tp.Shutdown(ctx)
			return nil
		}

		provider = tp
	}

	//setup propagator
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	//set this provider as global provider
	otel.SetTracerProvider(provider)

	return shutdown, nil
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

//=============================================================================

// for dev using this provider
func newTraceProvider(serviceName string) (*sdktrace.TracerProvider, error) {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint())

	if err != nil {
		return nil, fmt.Errorf("new stdouttrace: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(
			exporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			sdktrace.WithBatchTimeout(time.Second),
		),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(serviceName),
				semconv.ServiceVersionKey.String("v0.0.1"),
			),
		),
	)

	return provider, nil
}
