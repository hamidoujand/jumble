package mid

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hamidoujand/jumble/internal/auth"
	"github.com/hamidoujand/jumble/internal/domains/user/bus"
	"github.com/hamidoujand/jumble/internal/errs"
	"github.com/hamidoujand/jumble/pkg/mux"
)

func Authenticate(a *auth.Auth, usrBus *bus.Bus) mux.Middleware {
	return func(next mux.HandlerFunc) mux.HandlerFunc {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			// using a 5 seconds ctx to hit the db
			ctx, cancel := context.WithTimeout(ctx, time.Second*5)
			defer cancel()

			token := r.Header.Get("authorization")

			c, err := a.VerifyToken(ctx, token)
			if err != nil {
				return errs.New(http.StatusUnauthorized, err)
			}

			if c.Subject == "" {
				return errs.Newf(http.StatusUnauthorized, "you are not authorized to take this action")
			}

			userID, err := uuid.Parse(c.Subject)
			if err != nil {
				return errs.Newf(http.StatusUnauthorized, "invalid user id: %s", err)
			}

			//fetch the user from db
			usr, err := usrBus.QueryByID(ctx, userID)
			if errors.Is(err, bus.ErrUserNotFound) {
				return errs.New(http.StatusUnauthorized, err)
			}

			if err != nil {
				return errs.Newf(http.StatusInternalServerError, "queryById: %s", err)
			}

			if !usr.Enabled {
				return errs.Newf(http.StatusUnauthorized, "user is disabled")
			}

			//add claims to request context
			ctx = auth.SetClaims(ctx, c)
			//add user to request context
			ctx = auth.SetUser(ctx, usr)

			return next(ctx, w, r)
		}
	}
}
