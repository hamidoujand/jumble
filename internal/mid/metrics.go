package mid

import (
	"context"
	"net/http"

	"github.com/hamidoujand/jumble/internal/metrics"
	"github.com/hamidoujand/jumble/pkg/mux"
)

func Metrics() mux.Middleware {
	return func(next mux.HandlerFunc) mux.HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			//set the metrics into ctx
			ctx = metrics.Set(ctx)

			err := next(ctx, w, r)

			numReq := metrics.AddRequest(ctx)
			if numReq%1000 == 0 {
				metrics.AddGoroutine(ctx)
			}

			if err != nil {
				metrics.AddError(ctx)
			}

			return err
		}
	}
}
