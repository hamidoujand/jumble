package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// Config defines the information needed to init tracing.
type Config struct {
	ServiceName string
	Host        string
	Probability float64
	Build       string
}

func SetupOTelSDK(cfg Config) (func(context.Context) error, error) {
	shutdown := func(ctx context.Context) error { return nil }
	var provider trace.TracerProvider

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	switch cfg.Host {
	case "":
		//test
		provider = noop.NewTracerProvider()

	case "dev":
		//dev
		exporter, err := stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
		if err != nil {
			return nil, fmt.Errorf("stdout exporter: %w", err)
		}

		p := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter,
				// Default is 5s. Set to 1s for demonstrative purposes.
				sdktrace.WithBatchTimeout(time.Second)),
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
			p.Shutdown(ctx)
			return nil
		}
		provider = p

	default:
		//prod http
		exporter, err := newHTTPExporter(ctx, cfg.Host)
		if err != nil {
			return nil, err
		}

		tp := sdktrace.NewTracerProvider(
			//sampler
			sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.Probability))),
			//batcher
			sdktrace.WithBatcher(exporter,
				sdktrace.WithMaxExportBatchSize(sdktrace.DefaultMaxExportBatchSize),
				sdktrace.WithBatchTimeout(time.Second*1),
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

// ==============================================================================
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newGRPCExporter(ctx context.Context, host string) (*otlptrace.Exporter, error) {
	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(host),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("creating new exporter: %w", err)
	}

	return exporter, nil
}

func newHTTPExporter(ctx context.Context, host string) (*otlptrace.Exporter, error) {
	exporter, err := otlptrace.New(
		ctx,
		otlptracehttp.NewClient(
			otlptracehttp.WithInsecure(),
			otlptracehttp.WithEndpoint(host),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("new exporter: %w", err)
	}

	return exporter, nil
}
