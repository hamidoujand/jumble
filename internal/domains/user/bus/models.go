package bus

import (
	"net/mail"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Name         string
	Email        mail.Address
	Roles        []Role
	PasswordHash []byte
	Department   string
	Enabled      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type NewUser struct {
	Name       string
	Email      mail.Address
	Roles      []Role
	Department string
	Password   string
}

type UpdateUser struct {
	Name       *string
	Email      *mail.Address
	Roles      []Role
	Department *string
	Password   *string
	Enabled    *bool
}
