// Package user provides an example of a core business API.
package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"

	"github.com/jmoiron/sqlx"
	"github.com/tcmhoang/sservices/business/data/store/user"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var ErrAuthenticationFailure = errors.New("authentication failed")

type Core struct {
	log   *zap.SugaredLogger
	Store user.Store
}

func NewCore(log *zap.SugaredLogger, db *sqlx.DB) *Core {
	return &Core{
		log,
		*user.NewStore(log, db),
	}
}

func (c *Core) Authenticate(ctx context.Context, email mail.Address, password string) (user.User, error) {
	usr, err := c.Store.QueryByEmail(ctx, email)
	if err != nil {
		return user.User{}, fmt.Errorf("query: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword(usr.PasswordHash, []byte(password)); err != nil {
		return user.User{}, ErrAuthenticationFailure
	}

	return usr, nil
}
