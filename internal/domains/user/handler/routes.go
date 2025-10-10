package handler

import (
	"net/http"
	"time"

	"github.com/hamidoujand/jumble/internal/auth"
	"github.com/hamidoujand/jumble/internal/domains/user/bus"
	"github.com/hamidoujand/jumble/internal/mid"
	"github.com/hamidoujand/jumble/pkg/mux"
	"go.opentelemetry.io/otel/trace"
)

type Conf struct {
	Mux         *mux.Mux
	UserBus     *bus.Bus
	Auth        *auth.Auth
	Kid         string
	Issuer      string
	TokenMaxAge time.Duration
	Tracer      trace.Tracer
}

// RegisterRoutes takes the mux and register endpoints on it.
func RegisterRoutes(cfg Conf) {
	const version = "v1"

	usr := handler{
		userBus:     cfg.UserBus,
		a:           cfg.Auth,
		kid:         cfg.Kid,
		issuer:      cfg.Issuer,
		tokenMaxAge: cfg.TokenMaxAge,
		tracer:      cfg.Tracer,
	}

	authenticated := mid.Authenticate(cfg.Auth, cfg.UserBus)

	onlyAdmin := mid.Authorized(cfg.Auth, map[string]struct{}{
		bus.RoleAdmin.String(): {},
	})

	onlyOwner := mid.Authorized(cfg.Auth, map[string]struct{}{
		bus.RoleUser.String(): {},
	})

	adminOrOwner := mid.Authorized(cfg.Auth, map[string]struct{}{
		bus.RoleAdmin.String(): {},
		bus.RoleUser.String():  {},
	})

	cfg.Mux.HandleFunc(http.MethodPost, version, "/users", usr.CreateUser)
	cfg.Mux.HandleFunc(http.MethodGet, version, "/users", usr.Query)
	cfg.Mux.HandleFunc(http.MethodGet, version, "/users/{id}", usr.QueryUserByID, authenticated)
	cfg.Mux.HandleFunc(http.MethodPut, version, "/users/{id}", usr.UpdateUser, authenticated, onlyOwner)
	cfg.Mux.HandleFunc(http.MethodDelete, version, "/users/{id}", usr.DeleteUser, authenticated, adminOrOwner)
	cfg.Mux.HandleFunc(http.MethodPut, version, "/users/roles/{id}", usr.UpdateRole, authenticated, onlyAdmin)
	cfg.Mux.HandleFunc(http.MethodPut, version, "/users/disable/{id}", usr.DisableUser, authenticated, adminOrOwner)

	cfg.Mux.HandleFunc(http.MethodPost, version, "/users/login", usr.Authenticate)

}
