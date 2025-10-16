// Package handlers provides endpoints to intract with users domain.
package handler

import (
	"errors"
	"net/http"
	"net/mail"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/hamidoujand/jumble/internal/auth"
	"github.com/hamidoujand/jumble/internal/domains/user/bus"
	"github.com/hamidoujand/jumble/internal/errs"
	"github.com/hamidoujand/jumble/internal/page"
	"github.com/hamidoujand/jumble/pkg/logger"
	"go.opentelemetry.io/otel/trace"
)

type handler struct {
	userBus     *bus.Bus
	a           *auth.Auth
	kid         string
	issuer      string
	tokenMaxAge time.Duration
	tracer      trace.Tracer
	logger      logger.Logger
}

func (h *handler) CreateUser(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "user.handler.createUser")
	defer span.End()

	var nu newUser
	if err := c.ShouldBindJSON(&nu); err != nil {
		c.Error(err)
		return
	}

	busUser, err := toBusNewUser(nu)
	if err != nil {
		c.Error(errs.New(http.StatusBadRequest, "toBusNewUser: %s", err))
		return
	}

	usr, err := h.userBus.Create(ctx, busUser)
	if errors.Is(err, bus.ErrDuplicatedEmail) {
		c.Error(errs.New(http.StatusBadRequest, "create: %s", err))
		return
	}

	if err != nil {
		c.Error(errs.New(http.StatusInternalServerError, "create: %s", err))
		return
	}

	appUser := toAppUser(usr)
	claims := auth.Claims{
		Roles: appUser.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    h.issuer,
			Subject:   appUser.ID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.tokenMaxAge)),
		},
	}

	token, err := h.a.GenerateToken(h.kid, claims)
	if err != nil {
		c.Error(errs.New(http.StatusInternalServerError, "generateToken: %s", err))
		return
	}

	appUser.Token = token
	c.JSON(http.StatusCreated, appUser)
}

func (h *handler) QueryUserByID(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "user.handler.queryByID")
	defer span.End()

	p := c.Param("id")

	userID, err := uuid.Parse(p)
	if err != nil {
		c.Error(errs.New(http.StatusBadRequest, "invalid id: %s", p))
		return
	}

	usr, err := h.userBus.QueryByID(ctx, userID)
	if errors.Is(err, bus.ErrUserNotFound) {
		c.Error(errs.New(http.StatusNotFound, "queryByID: %s", err))
		return
	}

	if err != nil {
		c.Error(errs.New(http.StatusInternalServerError, "queryByID: %s", err))
		return
	}

	c.JSON(http.StatusOK, toAppUser(usr))
}

func (h *handler) DeleteUser(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "user.handler.deleteUser")
	defer span.End()

	p := c.Param("id")

	userId, err := uuid.Parse(p)
	if err != nil {
		c.Error(errs.New(http.StatusBadRequest, "invalid user id: %s", p))
		return
	}

	val, ok := c.Get("user")
	if !ok {
		c.Error(errs.New(http.StatusUnauthorized, "%s", http.StatusText(http.StatusUnauthorized)))
		return
	}

	usr, ok := val.(bus.User)
	if !ok {
		c.Error(errs.New(http.StatusUnauthorized, "%s", http.StatusText(http.StatusUnauthorized)))
		return
	}

	//either the user itself or admin can delete a user .
	if userId != usr.ID && !isAdmin(usr.Roles) {
		c.Error(errs.New(http.StatusUnauthorized, "unauthorized to take this action"))
		return
	}

	targetUser, err := h.userBus.QueryByID(ctx, userId)
	if errors.Is(err, bus.ErrUserNotFound) {
		c.Error(errs.New(http.StatusNotFound, "%s", err))
		return
	}

	if err != nil {
		c.Error(errs.New(http.StatusInternalServerError, "queryByID: %s", err))
		return
	}

	//delete the target user
	if err := h.userBus.Delete(ctx, targetUser); err != nil {
		c.Error(errs.New(http.StatusInternalServerError, "delete: %s", err))
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *handler) UpdateUser(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "user.handler.updateUser")
	defer span.End()

	p := c.Param("id")
	targetID, err := uuid.Parse(p)
	if err != nil {
		c.Error(errs.New(http.StatusBadRequest, "invalid user id: %s", p))
		return
	}

	val, ok := c.Get("user")
	if !ok {
		c.Error(errs.New(http.StatusUnauthorized, "%s", http.StatusText(http.StatusUnauthorized)))
		return
	}

	usr, ok := val.(bus.User)
	if !ok {
		c.Error(errs.New(http.StatusUnauthorized, "%s", http.StatusText(http.StatusUnauthorized)))
		return
	}

	if usr.ID != targetID {
		c.Error(errs.New(http.StatusUnauthorized, "%s", http.StatusText(http.StatusUnauthorized)))
		return
	}

	var uu updateUser
	if err := c.ShouldBindJSON(&uu); err != nil {
		c.Error(err)
		return
	}

	busUserUpdate, err := toBusUpdateUser(uu)
	if err != nil {
		c.Error(errs.New(http.StatusBadRequest, "toUpdateBusUser: %s", err))
		return
	}

	updated, err := h.userBus.Update(ctx, usr, busUserUpdate)
	if errors.Is(err, bus.ErrDuplicatedEmail) {
		c.Error(errs.New(http.StatusBadRequest, "%s", err))
		return
	}

	if err != nil {
		c.Error(errs.New(http.StatusInternalServerError, "delete: %s", err))
		return
	}

	c.JSON(http.StatusOK, toAppUser(updated))
}

func (h *handler) UpdateRole(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "user.handler.updateRole")
	defer span.End()

	val, ok := c.Get("user")
	if !ok {
		c.Error(errs.New(http.StatusUnauthorized, "%s", http.StatusText(http.StatusUnauthorized)))
		return
	}

	admin, ok := val.(bus.User)
	if !ok {
		c.Error(errs.New(http.StatusUnauthorized, "%s", http.StatusText(http.StatusUnauthorized)))
		return
	}

	//must be an admin
	if !isAdmin(admin.Roles) {
		c.Error(errs.New(http.StatusUnauthorized, "%s", http.StatusText(http.StatusUnauthorized)))
		return
	}

	//target userID
	p := c.Param("id")
	userId, err := uuid.Parse(p)
	if err != nil {
		c.Error(errs.New(http.StatusBadRequest, "invalid user id: %s", p))
		return
	}

	var ur updateUserRoles
	if err := c.ShouldBindJSON(&ur); err != nil {
		c.Error(err)
		return
	}

	busUpdateRoles, err := toBusUpdateUserRoles(ur)
	if err != nil {
		c.Error(errs.New(http.StatusBadRequest, "toBusUpdateUserRoles: %s", err))
		return
	}

	//fetch the usr from db
	usr, err := h.userBus.QueryByID(ctx, userId)
	if errors.Is(err, bus.ErrUserNotFound) {
		c.Error(errs.New(http.StatusNotFound, "%s", err))
		return
	}

	if err != nil {
		c.Error(errs.New(http.StatusInternalServerError, "queryByID: %s", err))
		return
	}

	if !usr.Enabled {
		c.Error(errs.New(http.StatusBadRequest, "user is disabled"))
		return
	}

	updated, err := h.userBus.Update(ctx, usr, busUpdateRoles)
	if err != nil {
		c.Error(errs.New(http.StatusInternalServerError, "queryByID: %s", err))
		return
	}

	c.JSON(http.StatusOK, toAppUser(updated))
}

func (h *handler) DisableUser(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "user.handler.disableUser")
	defer span.End()

	p := c.Param("id")

	userId, err := uuid.Parse(p)
	if err != nil {
		c.Error(errs.New(http.StatusBadRequest, "invalid user id: %s", p))
		return
	}

	//fetch the user from ctx doing this action.
	val, ok := c.Get("user")
	if !ok {
		c.Error(errs.New(http.StatusUnauthorized, "%s", http.StatusText(http.StatusUnauthorized)))
		return
	}
	usr, ok := val.(bus.User)
	if !ok {
		c.Error(errs.New(http.StatusUnauthorized, "%s", http.StatusText(http.StatusUnauthorized)))
		return
	}

	//either the user itself or admin can delete a user .
	if userId != usr.ID && !isAdmin(usr.Roles) {
		c.Error(errs.New(http.StatusUnauthorized, "%s", http.StatusText(http.StatusUnauthorized)))
		return
	}

	targetUser, err := h.userBus.QueryByID(ctx, userId)
	if errors.Is(err, bus.ErrUserNotFound) {
		c.Error(errs.New(http.StatusNotFound, "%s", err))
		return
	}

	if err != nil {
		c.Error(errs.New(http.StatusInternalServerError, "queryByID: %s", err))
		return
	}

	enabled := false
	uu := updateUser{
		Enabled: &enabled,
	}

	//since we are setting the enabled to false no need for error checking
	busUpdateUser, _ := toBusUpdateUser(uu)

	updated, err := h.userBus.Update(ctx, targetUser, busUpdateUser)
	if err != nil {
		c.Error(errs.New(http.StatusInternalServerError, "update: %s", err))
		return
	}

	c.JSON(http.StatusOK, toAppUser(updated))
}

func (h *handler) Query(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "user.handler.query")
	defer span.End()

	//pagination
	p := c.Query("page")
	rows := c.Query("rows")

	page, err := page.Parse(p, rows)
	if err != nil {
		c.Error(errs.New(http.StatusBadRequest, "parse pagination: %s", err))
		return
	}

	//parse filters
	var filters Filters
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.Error(err)
		return
	}

	busFilter, err := filters.ToBusQueryFilter()
	if err != nil {
		c.Error(errs.New(http.StatusBadRequest, "toBusQueryFilter: %s", err.Error()))
		return
	}

	//order by
	orderBy, err := bus.ParseOrderBy(c.Query("order_by"))
	if err != nil {
		c.Error(errs.New(http.StatusBadRequest, "parse order_by query: %s", err))
		return
	}

	busUsers, err := h.userBus.Query(ctx, busFilter, orderBy, page)
	if err != nil {
		c.Error(errs.New(http.StatusInternalServerError, "query: %s", err))
		return
	}

	total, err := h.userBus.Count(ctx, busFilter)
	if err != nil {
		c.Error(errs.New(http.StatusInternalServerError, "count: %s", err))
		return
	}

	users := make([]user, len(busUsers))
	for i, usr := range busUsers {
		users[i] = toAppUser(usr)
	}

	qr := newQueryResult(users, total, page.Number, page.Rows)
	c.JSON(http.StatusOK, qr)
}

func (h *handler) Authenticate(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "user.handler.authenticate")
	defer span.End()

	var authData authenticate
	if err := c.ShouldBindJSON(&authData); err != nil {
		c.Error(err)
		return
	}

	email, err := mail.ParseAddress(authData.Email)
	if err != nil {
		c.Error(errs.New(http.StatusBadRequest, "parseAddress: %s", err))
		return
	}

	usr, err := h.userBus.Authenticate(ctx, *email, authData.Password)
	if err != nil {
		c.Error(errs.New(http.StatusBadRequest, "invalid email or password: %s", err))
		return
	}

	claims := auth.Claims{
		Roles: bus.RolesToString(usr.Roles),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    h.issuer,
			Subject:   usr.ID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.tokenMaxAge)),
		},
	}

	token, err := h.a.GenerateToken(h.kid, claims)
	if err != nil {
		c.Error(errs.New(http.StatusInternalServerError, "generateToken: %s", err))
		return
	}

	t := Token{Token: token}
	c.JSON(http.StatusOK, t)
}

// ==============================================================================
func isAdmin(roles []bus.Role) bool {
	return slices.Contains(roles, bus.RoleAdmin)
}
