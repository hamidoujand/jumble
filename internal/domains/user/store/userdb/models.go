package userdb

import (
	"database/sql"
	"net/mail"
	"time"

	"github.com/google/uuid"
	usrBus "github.com/hamidoujand/jumble/internal/domains/user/bus"
)

type user struct {
	ID           uuid.UUID        `db:"id"`
	Name         string           `db:"name"`
	Email        string           `db:"email"`
	Roles        usrBus.RoleSlice `db:"roles"`
	PasswordHash []byte           `db:"password_hash"`
	Department   sql.NullString   `db:"department"`
	Enabled      bool             `db:"enabled"`
	CreatedAt    time.Time        `db:"created_at"`
	UpdatedAt    time.Time        `db:"updated_at"`
}

func fromBusUser(usr usrBus.User) user {

	return user{
		ID:           usr.ID,
		Name:         usr.Name,
		Email:        usr.Email.Address,
		Roles:        usrBus.RoleSlice(usr.Roles),
		PasswordHash: usr.PasswordHash,
		Department: sql.NullString{
			String: usr.Department,
			//When usr.Department is not empty: Valid becomes true, telling the database this field should be stored with the given string value
			Valid: usr.Department != "",
		},
		Enabled:   usr.Enabled,
		CreatedAt: usr.CreatedAt,
		UpdatedAt: usr.UpdatedAt,
	}
}

func toUserBus(usr user) usrBus.User {
	email := mail.Address{
		Name:    usr.Name,
		Address: usr.Email,
	}

	return usrBus.User{
		ID:           usr.ID,
		Name:         usr.Name,
		Email:        email,
		Roles:        []usrBus.Role(usr.Roles),
		PasswordHash: usr.PasswordHash,
		Department:   usr.Department.String,
		Enabled:      usr.Enabled,
		CreatedAt:    usr.CreatedAt,
		UpdatedAt:    usr.UpdatedAt,
	}
}
