package user

import (
	"net/mail"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID    `json:"id"`
	Name         string       `json:"name"`
	Email        mail.Address `json:"email"`
	Roles        []string     `json:"roles"`
	PasswordHash []byte       `json:"-"`
	Department   string       `json:"department"`
	Enabled      bool         `json:"enabled"`
	DateCreated  time.Time    `json:"dateCreated"`
	DateUpdated  time.Time    `json:"dateUpdated"`
}

type NewUser struct {
	Name            string       `json:"name" validate:"required"`
	Email           mail.Address `json:"email" validate:"required,email"`
	Roles           []string     `json:"roles" validate:"required"`
	Department      string       `json:"department"`
	Password        string       `json:"password" validate:"required"`
	PasswordConfirm string       `json:"passwordConfirm" validate:"eqfield=Password"`
}

type UpdateUser struct {
	Name            *string       `json:"name"`
	Email           *mail.Address `json:"email" validate:"omitempty,email"`
	Roles           []string      `json:"roles"`
	Department      *string       `json:"department"`
	Password        *string       `json:"password"`
	PasswordConfirm *string       `json:"passwordConfirm" validate:"omitempty,eqfield=Password"`
	Enabled         *bool         `json:"enabled"`
}
