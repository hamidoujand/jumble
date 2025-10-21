package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hamidoujand/jumble/internal/auth"
	"github.com/hamidoujand/jumble/internal/domains/user/bus"
	"github.com/hamidoujand/jumble/internal/mid"
	"github.com/hamidoujand/jumble/pkg/logger"
	"go.opentelemetry.io/otel/trace"
)

type Conf struct {
	Router      *gin.Engine
	UserBus     *bus.Bus
	Auth        *auth.Auth
	Kid         string
	Issuer      string
	TokenMaxAge time.Duration
	Tracer      trace.Tracer
	Logger      *logger.Logger
}

// RegisterRoutes takes the mux and register endpoints on it.
func RegisterRoutes(cfg Conf) {
	usr := handler{
		userBus:     cfg.UserBus,
		a:           cfg.Auth,
		kid:         cfg.Kid,
		issuer:      cfg.Issuer,
		tokenMaxAge: cfg.TokenMaxAge,
		tracer:      cfg.Tracer,
	}

	users := cfg.Router.Group("/v1/users")

	admin := mid.Authorized(usr.a, map[string]struct{}{bus.RoleAdmin.String(): {}})
	user := mid.Authorized(usr.a, map[string]struct{}{bus.RoleUser.String(): {}})
	adminOrUser := mid.Authorized(usr.a, map[string]struct{}{bus.RoleUser.String(): {}, bus.RoleUser.String(): {}})

	authenticated := mid.Authenticate(cfg.Logger, cfg.Auth, cfg.UserBus)

	users.POST("/", usr.CreateUser)
	users.GET("/:id", usr.QueryUserByID, authenticated)
	users.DELETE("/:id", usr.DeleteUser, authenticated, adminOrUser)
	users.PUT("/:id", usr.UpdateUser, authenticated, user)
	users.PUT("/roles/:id", usr.UpdateRole, authenticated, admin)
	users.PUT("/disable/:id", usr.DisableUser, authenticated, adminOrUser)
	users.GET("/", usr.Query)
	users.POST("/login", usr.Authenticate)
}
