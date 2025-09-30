package mid

import (
	"context"
	"net/http"

	"github.com/hamidoujand/jumble/pkg/mux"
	"github.com/hamidoujand/jumble/pkg/otel"
	"go.opentelemetry.io/otel/trace"
)

func Otel(tracer trace.Tracer) mux.Middleware {
	return func(next mux.HandlerFunc) mux.HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			//inject tracer into each request ctx
			ctx = otel.InjectTracing(ctx, tracer)
			return next(ctx, w, r)
		}
	}
}
