package mid

import (
	"context"
	"net/http"
	"runtime/debug"

	"github.com/hamidoujand/jumble/internal/errs"
	"github.com/hamidoujand/jumble/pkg/mux"
)

func Panic() mux.Middleware {
	return func(next mux.HandlerFunc) mux.HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()
					err = errs.Newf(http.StatusInternalServerError, "PANIC[%v] TRACE[%s]", rec, string(stack))
				}
			}()

			return next(ctx, w, r)
		}
	}
}
