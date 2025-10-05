package mid

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hamidoujand/jumble/pkg/mux"
	"github.com/hamidoujand/jumble/pkg/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func AddTracer(tracer trace.Tracer) mux.Middleware {
	return func(next mux.HandlerFunc) mux.HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			ctx = telemetry.SetTracer(ctx, tracer)
			return next(ctx, w, r)
		}
	}
}

func HTTPSpanMiddleware(tracer trace.Tracer) mux.Middleware {
	return func(next mux.HandlerFunc) mux.HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			// Extract trace context from headers FIRST
			ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))

			// Start span for this HTTP request
			ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path))
			defer span.End()

			// Set HTTP attributes
			span.SetAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.route", r.URL.Path),
				attribute.String("http.target", r.URL.String()),
			)

			//set the traceID into ctx
			traceID := span.SpanContext().TraceID().String()
			ctx = telemetry.SetTraceID(ctx, traceID)

			// Process the request
			err := next(ctx, w, r)

			// Set status code if available
			if statusCode := mux.GetStatusCode(ctx); statusCode != 0 {
				span.SetAttributes(attribute.Int("http.status_code", statusCode))
				if statusCode >= 400 {
					span.SetStatus(codes.Error, http.StatusText(statusCode))
				}
			}

			// Inject trace context into response
			otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(w.Header()))

			return err
		}
	}
}
