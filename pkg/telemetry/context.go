package telemetry

import (
	"context"
)

const defaultTraceID = "00000000000000000000000000000000"

type ctxKey int

const (
	traceIDKey ctxKey = 1
)

func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// GetTraceID returns the trace id from the context.
func GetTraceID(ctx context.Context) string {
	v, ok := ctx.Value(traceIDKey).(string)
	if !ok {
		return defaultTraceID
	}

	return v
}
