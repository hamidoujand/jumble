// Package handlers provides endpoints to intract with users domain.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/hamidoujand/jumble/internal/auth"
	"github.com/hamidoujand/jumble/internal/domains/user/bus"
	"github.com/hamidoujand/jumble/internal/errs"
	"github.com/hamidoujand/jumble/internal/page"
	"github.com/hamidoujand/jumble/pkg/mux"
)

type handler struct {
	userBus     *bus.Bus
	a           *auth.Auth
	kid         string
	issuer      string
	tokenMaxAge time.Duration
}

func (h *handler) CreateUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var nu newUser
	if err := json.NewDecoder(r.Body).Decode(&nu); err != nil {
		return errs.Newf(http.StatusBadRequest, "invalid json: %s", err)
	}

	if err := nu.Validate(); err != nil {
		return err
	}

	busUser, err := toBusNewUser(nu)
	if err != nil {
		return errs.New(http.StatusBadRequest, err)
	}

	usr, err := h.userBus.Create(ctx, busUser)

	if errors.Is(err, bus.ErrDuplicatedEmail) {
		return errs.New(http.StatusBadRequest, err)
	}

	if err != nil {
		return errs.Newf(http.StatusInternalServerError, "create: %s", err)
	}

	appUser := toAppUser(usr)
	c := auth.Claims{
		Roles: appUser.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    h.issuer,
			Subject:   appUser.ID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.tokenMaxAge)),
		},
	}

	token, err := h.a.GenerateToken(h.kid, c)
	if err != nil {
		return errs.Newf(http.StatusInternalServerError, "generateToken: %s", err)
	}

	appUser.Token = token

	if err := mux.Respond(ctx, w, http.StatusCreated, appUser); err != nil {
		return errs.Newf(http.StatusInternalServerError, "respond: %s", err)
	}

	return nil
}

func (h *handler) QueryUserByID(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	p := r.PathValue("id")

	userID, err := uuid.Parse(p)
	if err != nil {
		return errs.Newf(http.StatusBadRequest, "invalid id: %s", err)
	}

	usr, err := h.userBus.QueryByID(ctx, userID)
	if errors.Is(err, bus.ErrUserNotFound) {
		return errs.New(http.StatusNotFound, err)
	}

	if err != nil {
		return errs.Newf(http.StatusInternalServerError, "queryByID: %s", err)
	}

	if err := mux.Respond(ctx, w, http.StatusOK, toAppUser(usr)); err != nil {
		return errs.Newf(http.StatusInternalServerError, "respond: %s", err)
	}

	return nil
}

func (h *handler) DeleteUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	p := r.PathValue("id")

	userId, err := uuid.Parse(p)
	if err != nil {
		return errs.Newf(http.StatusBadRequest, "invalid user id: %s", err)
	}

	//fetch the user from ctx doing this action.
	usr, err := auth.GetUser(ctx)
	if err != nil {
		return errs.New(http.StatusUnauthorized, err)
	}

	//either the user itself or admin can delete a user .
	if userId != usr.ID && !isAdmin(usr.Roles) {
		return errs.Newf(http.StatusUnauthorized, "user is unauthorized to take this action")
	}

	targetUser, err := h.userBus.QueryByID(ctx, userId)
	if errors.Is(err, bus.ErrUserNotFound) {
		return errs.New(http.StatusNotFound, err)
	}

	if err != nil {
		return errs.Newf(http.StatusInternalServerError, "queryByID: %s", err)
	}

	//delete the target user
	if err := h.userBus.Delete(ctx, targetUser); err != nil {
		return errs.Newf(http.StatusInternalServerError, "delete: %s", err)
	}

	if err := mux.Respond(ctx, w, http.StatusNoContent, nil); err != nil {
		return errs.Newf(http.StatusInternalServerError, "respond: %s", err)
	}

	return nil
}

func (h *handler) UpdateUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	usr, err := auth.GetUser(ctx)
	if err != nil {
		return errs.New(http.StatusUnauthorized, err)
	}

	var uu updateUser
	if err := json.NewDecoder(r.Body).Decode(&uu); err != nil {
		return errs.Newf(http.StatusBadRequest, "invalid json: %s", err)
	}

	if err := uu.Validate(); err != nil {
		return err
	}

	busUserUpdate, err := toBusUpdateUser(uu)
	if err != nil {
		return errs.New(http.StatusBadRequest, err)
	}

	updated, err := h.userBus.Update(ctx, usr, busUserUpdate)
	if errors.Is(err, bus.ErrDuplicatedEmail) {
		return errs.New(http.StatusBadRequest, err)
	}

	if err != nil {
		return errs.Newf(http.StatusInternalServerError, "update: %s", err)
	}

	if err := mux.Respond(ctx, w, http.StatusOK, toAppUser(updated)); err != nil {
		return errs.Newf(http.StatusInternalServerError, "respond: %s", err)
	}

	return nil
}

func (h *handler) UpdateRole(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	admin, err := auth.GetUser(ctx)
	if err != nil {
		return errs.New(http.StatusUnauthorized, err)
	}

	//must be an admin
	if !isAdmin(admin.Roles) {
		return errs.Newf(http.StatusUnauthorized, "unauthorized to take this action")
	}

	//target userID
	p := r.PathValue("id")
	userId, err := uuid.Parse(p)
	if err != nil {
		return errs.Newf(http.StatusBadRequest, "invalid user id: %s", err)
	}

	var ur updateUserRoles
	if err := json.NewDecoder(r.Body).Decode(&ur); err != nil {
		return errs.Newf(http.StatusBadRequest, "invalid json: %s", err)
	}

	if err := ur.Validate(); err != nil {
		return err
	}

	busUpdateRoles, err := toBusUpdateUserRoles(ur)
	if err != nil {
		return errs.New(http.StatusBadRequest, err)
	}

	//fetch the usr from db
	usr, err := h.userBus.QueryByID(ctx, userId)
	if errors.Is(err, bus.ErrUserNotFound) {
		return errs.New(http.StatusNotFound, err)
	}

	if err != nil {
		return errs.Newf(http.StatusInternalServerError, "queryByID: %s", err)
	}

	if !usr.Enabled {
		return errs.Newf(http.StatusBadRequest, "user is disabled")
	}

	updated, err := h.userBus.Update(ctx, usr, busUpdateRoles)
	if err != nil {
		return errs.Newf(http.StatusInternalServerError, "update: %s", err)
	}

	if err := mux.Respond(ctx, w, http.StatusOK, toAppUser(updated)); err != nil {
		return errs.Newf(http.StatusInternalServerError, "respond: %s", err)
	}

	return nil
}

func (h *handler) DisableUser(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	p := r.PathValue("id")

	userId, err := uuid.Parse(p)
	if err != nil {
		return errs.Newf(http.StatusBadRequest, "invalid user id: %s", err)
	}

	//fetch the user from ctx doing this action.
	usr, err := auth.GetUser(ctx)
	if err != nil {
		return errs.New(http.StatusUnauthorized, err)
	}

	//either the user itself or admin can delete a user .
	if userId != usr.ID && !isAdmin(usr.Roles) {
		return errs.Newf(http.StatusUnauthorized, "user is unauthorized to take this action")
	}

	targetUser, err := h.userBus.QueryByID(ctx, userId)
	if errors.Is(err, bus.ErrUserNotFound) {
		return errs.New(http.StatusNotFound, err)
	}

	if err != nil {
		return errs.Newf(http.StatusInternalServerError, "queryByID: %s", err)
	}

	enabled := false
	uu := updateUser{
		Enabled: &enabled,
	}

	//since we are setting the enabled to false no need for error checking
	busUpdateUser, _ := toBusUpdateUser(uu)

	updated, err := h.userBus.Update(ctx, targetUser, busUpdateUser)
	if err != nil {
		return errs.Newf(http.StatusInternalServerError, "update: %s", err)
	}

	if err := mux.Respond(ctx, w, http.StatusOK, toAppUser(updated)); err != nil {
		return errs.Newf(http.StatusInternalServerError, "respond: %s", err)
	}

	return nil
}

func (h *handler) Query(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	//pagination
	p := r.URL.Query().Get("page")
	rows := r.URL.Query().Get("rows")

	page, err := page.Parse(p, rows)
	if err != nil {
		return errs.Newf(http.StatusBadRequest, "pagination: %s", err)
	}

	//filters
	filters, err := parseFilters(r)
	if err != nil {
		return err
	}

	//order by
	orderBy, err := bus.ParseOrderBy(r.URL.Query().Get("order_by"))
	if err != nil {
		return errs.Newf(http.StatusBadRequest, "parse order_by: %s", err)
	}

	busUsers, err := h.userBus.Query(ctx, filters, orderBy, page)
	if err != nil {
		return errs.Newf(http.StatusInternalServerError, "query: %s", err)
	}

	total, err := h.userBus.Count(ctx, filters)
	if err != nil {
		return errs.Newf(http.StatusInternalServerError, "count: %s", err)
	}

	users := make([]user, len(busUsers))
	for i, usr := range busUsers {
		users[i] = toAppUser(usr)
	}

	qr := newQueryResult(users, total, page.Number, page.Rows)

	if err := mux.Respond(ctx, w, http.StatusOK, qr); err != nil {
		return errs.Newf(http.StatusInternalServerError, "respond: %s", err)
	}

	return nil
}

func (h *handler) Authenticate(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var authData authenticate
	if err := json.NewDecoder(r.Body).Decode(&authData); err != nil {
		return errs.Newf(http.StatusBadRequest, "invalid json: %s", err)
	}

	if err := authData.Validate(); err != nil {
		return err
	}

	email, err := mail.ParseAddress(authData.Email)
	if err != nil {
		return errs.Newf(http.StatusBadRequest, "invalid email or password")
	}

	usr, err := h.userBus.Authenticate(ctx, *email, authData.Password)
	if err != nil {
		return errs.Newf(http.StatusBadRequest, "invalid email or password")
	}

	c := auth.Claims{
		Roles: bus.RolesToString(usr.Roles),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    h.issuer,
			Subject:   usr.ID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.tokenMaxAge)),
		},
	}

	token, err := h.a.GenerateToken(h.kid, c)
	if err != nil {
		return errs.Newf(http.StatusInternalServerError, "generateToken: %s", err)
	}

	t := Token{Token: token}

	if err := mux.Respond(ctx, w, http.StatusCreated, t); err != nil {
		return errs.Newf(http.StatusInternalServerError, "respond: %s", err)
	}

	return nil
}

// ==============================================================================
func isAdmin(roles []bus.Role) bool {
	return slices.Contains(roles, bus.RoleAdmin)
}
