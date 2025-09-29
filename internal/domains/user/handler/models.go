package handler

import (
	"fmt"
	"net/http"
	"net/mail"
	"time"

	"github.com/hamidoujand/jumble/internal/domains/user/bus"
	"github.com/hamidoujand/jumble/internal/errs"
)

type user struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Email      string   `json:"email"`
	Roles      []string `json:"roles"`
	Department string   `json:"department"`
	Enabled    bool     `json:"enabled"`
	CreatedAt  string   `json:"createdAt"`
	UpdatedAt  string   `json:"updatedAt"`
	Token      string   `json:"token,omitempty"`
}

func toAppUser(usr bus.User) user {
	return user{
		ID:         usr.ID.String(),
		Name:       usr.Name,
		Email:      usr.Email.Address,
		Roles:      bus.RolesToString(usr.Roles),
		Department: usr.Department,
		Enabled:    usr.Enabled,
		CreatedAt:  usr.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  usr.UpdatedAt.Format(time.RFC3339),
	}
}

// ==============================================================================
type QueryResult struct {
	Users       []user `json:"users"`
	Total       int    `json:"total"`
	Page        int    `json:"page"`
	RowsPerPage int    `json:"rowsPerPage"`
}

func newQueryResult(users []user, total int, page int, rows int) QueryResult {
	return QueryResult{
		Users:       users,
		Total:       total,
		Page:        page,
		RowsPerPage: rows,
	}
}

// ==============================================================================
type authenticate struct {
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=8,max=128"`
	PasswordConfirm string `json:"passwordConfirm" validate:"required,eqfield=Password"`
}

func (au authenticate) Validate() error {
	fields := errs.Check(au)
	if len(fields) == 0 {
		return nil
	}

	return errs.NewValidationErr(http.StatusBadRequest, fields)
}

// ==============================================================================
type Token struct {
	Token string `json:"token"`
}

//==============================================================================

type newUser struct {
	Name            string   `json:"name" validate:"required,min=4"`
	Email           string   `json:"email" validate:"required,email"`
	Roles           []string `json:"roles" validate:"required"`
	Department      string   `json:"department" validate:"required,oneof=sales shipping marketing"`
	Password        string   `json:"password" validate:"required,min=8,max=128"`
	PasswordConfirm string   `json:"passwordConfirm" validate:"required,eqfield=Password"`
}

func (nu newUser) Validate() error {
	fields := errs.Check(nu)
	if len(fields) == 0 {
		return nil
	}

	//errs
	return errs.NewValidationErr(http.StatusBadRequest, fields)
}

func toBusNewUser(nu newUser) (bus.NewUser, error) {
	roles, err := bus.ParseManyRoles(nu.Roles)
	if err != nil {
		return bus.NewUser{}, fmt.Errorf("parseManyRoles: %w", err)
	}

	email, err := mail.ParseAddress(nu.Email)
	if err != nil {
		return bus.NewUser{}, fmt.Errorf("parseAddress: %w", err)
	}

	return bus.NewUser{
		Name:       nu.Name,
		Email:      *email,
		Roles:      roles,
		Department: nu.Department,
		Password:   nu.Password,
	}, nil
}

// ==============================================================================
type updateUser struct {
	Name            *string `json:"name" validate:"omitempty,min=4"`
	Email           *string `json:"email" validate:"omitempty,email"`
	Deaprtment      *string `json:"department" validate:"omitempty,oneof=sales shipping marketing"`
	Password        *string `json:"password" validate:"omitempty,min=8,max=128"`
	PasswordConfirm *string `json:"passwordConfirm" validate:"omitempty,eqfield=Password"`
	Enabled         *bool   `json:"enabled"`
}

func (uu updateUser) Validate() error {
	fields := errs.Check(uu)
	if len(fields) == 0 {
		return nil
	}

	return errs.NewValidationErr(http.StatusBadRequest, fields)
}

func toBusUpdateUser(uu updateUser) (bus.UpdateUser, error) {
	email, err := mail.ParseAddress(*uu.Email)
	if err != nil {
		return bus.UpdateUser{}, fmt.Errorf("parseAddress: %w", err)
	}

	return bus.UpdateUser{
		Name:       uu.Name,
		Email:      email,
		Department: uu.Deaprtment,
		Password:   uu.Password,
		Enabled:    uu.Enabled,
	}, nil
}

//==============================================================================

type updateUserRoles struct {
	Roles []string `json:"roles" validate:"required,oneof=user admin"`
}

func (ur updateUserRoles) Validate() error {
	filed := errs.Check(ur)
	if len(filed) == 0 {
		return nil
	}

	return errs.NewValidationErr(http.StatusBadRequest, filed)
}

func toBusUpdateUserRoles(ur updateUserRoles) (bus.UpdateUser, error) {
	roles, err := bus.ParseManyRoles(ur.Roles)
	if err != nil {
		return bus.UpdateUser{}, fmt.Errorf("parseManyRoles: %w", err)
	}

	return bus.UpdateUser{Roles: roles}, nil
}
