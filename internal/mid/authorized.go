package mid

import (
	"context"
	"net/http"

	"github.com/hamidoujand/jumble/internal/auth"
	"github.com/hamidoujand/jumble/internal/errs"
	"github.com/hamidoujand/jumble/pkg/mux"
)

func Authorized(a *auth.Auth, roles map[string]struct{}) mux.Middleware {
	return func(next mux.HandlerFunc) mux.HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			c, err := auth.GetClaims(ctx)
			if err != nil {
				return errs.New(http.StatusUnauthorized, err)
			}

			err = a.Authorized(c, roles)
			if err != nil {
				return errs.Newf(http.StatusUnauthorized, "unauthorized to take this action: %s", err)
			}

			return next(ctx, w, r)
		}
	}
}
