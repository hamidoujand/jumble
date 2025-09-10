package mid

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hamidoujand/jumble/internal/errs"
	"github.com/hamidoujand/jumble/pkg/logger"
	"github.com/hamidoujand/jumble/pkg/mux"
)

func Errors(log logger.Logger) mux.Middleware {
	return func(next mux.HandlerFunc) mux.HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			//no err
			err := next(ctx, w, r)
			if err == nil {
				return nil
			}

			//err
			var appErr *errs.Error
			if !errors.As(err, &appErr) {
				//unknown error, INTERNAL
				internal := errs.Newf(http.StatusInternalServerError, "Internal server error")
				appErr, _ = internal.(*errs.Error)
			}

			//log the error
			log.Error(ctx, "handled error during request", "err", err, "fileName", appErr.FileName, "funcName", appErr.FuncName)

			if err := mux.Respond(ctx, w, appErr.Code, appErr); err != nil {
				return fmt.Errorf("failed to send the error to client: %w", err)
			}
			return nil
		}
	}
}
