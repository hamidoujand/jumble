package handler

import (
	"fmt"
	"net/mail"
	"time"

	"github.com/hamidoujand/jumble/internal/domains/user/bus"
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
	Email           string `json:"email" binding:"required,email"`
	Password        string `json:"password" binding:"required,min=8,max=128"`
	PasswordConfirm string `json:"passwordConfirm" binding:"required,eqfield=Password"`
}

// ==============================================================================
type Token struct {
	Token string `json:"token"`
}

//==============================================================================

type newUser struct {
	Name            string   `json:"name" binding:"required,min=4"`
	Email           string   `json:"email" binding:"required,email"`
	Roles           []string `json:"roles" binding:"gt=0,dive,required,oneof=admin user"`
	Department      string   `json:"department" binding:"required,oneof=sales shipping marketing"`
	Password        string   `json:"password" binding:"required,min=8,max=128"`
	PasswordConfirm string   `json:"passwordConfirm" binding:"required,eqfield=Password"`
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
	Name            *string `json:"name" binding:"omitempty,min=4"`
	Email           *string `json:"email" binding:"omitempty,email"`
	Deaprtment      *string `json:"department" binding:"omitempty,oneof=sales shipping marketing"`
	Password        *string `json:"password" binding:"omitempty,min=8,max=128"`
	PasswordConfirm *string `json:"passwordConfirm" binding:"omitempty,eqfield=Password"`
	Enabled         *bool   `json:"enabled"`
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
	Roles []string `json:"roles" binding:"required,oneof=user admin"`
}

func toBusUpdateUserRoles(ur updateUserRoles) (bus.UpdateUser, error) {
	roles, err := bus.ParseManyRoles(ur.Roles)
	if err != nil {
		return bus.UpdateUser{}, fmt.Errorf("parseManyRoles: %w", err)
	}

	return bus.UpdateUser{Roles: roles}, nil
}
