package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type ctxKey int

const (
	tracerKey  ctxKey = 1
	traceIDKey ctxKey = 2
)

func SetTracer(ctx context.Context, tracer trace.Tracer) context.Context {
	return context.WithValue(ctx, tracerKey, tracer)
}

func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

func GetTraceID(ctx context.Context) string {
	traceID, ok := ctx.Value(traceIDKey).(string)
	if !ok {
		return "00000000000000000000000000000000"
	}
	return traceID
}
