package mux

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type ctxKey int

const requestMetadataKey ctxKey = 1

// requestMeta represents some metadara about the request.
type requestMeta struct {
	startedAt  time.Time
	requestID  uuid.UUID
	statusCode int
}

func setReqMetadata(ctx context.Context, rm *requestMeta) context.Context {
	return context.WithValue(ctx, requestMetadataKey, rm)
}

func GetReqStartedAt(ctx context.Context) time.Time {
	rm, ok := ctx.Value(requestMetadataKey).(*requestMeta)
	if !ok {
		return time.Time{}
	}

	return rm.startedAt
}

func GetTraceID(ctx context.Context) uuid.UUID {
	rm, ok := ctx.Value(requestMetadataKey).(*requestMeta)
	if !ok {
		return uuid.Nil
	}

	return rm.requestID
}

func setStatusCode(ctx context.Context, statusCode int) error {
	rm, ok := ctx.Value(requestMetadataKey).(*requestMeta)
	if !ok {
		return errors.New("request metatdata not found in ctx")
	}

	rm.statusCode = statusCode
	return nil
}

func GetStatusCode(ctx context.Context) int {
	rm, ok := ctx.Value(requestMetadataKey).(*requestMeta)
	if !ok {
		return 0
	}
	return rm.statusCode
}
