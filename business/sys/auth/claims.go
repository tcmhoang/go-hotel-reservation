package auth

import (
	"context"
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

type Roles int

const (
	Admin Roles = iota
	User
)

type Claims struct {
	jwt.RegisteredClaims
	Roles []Roles `json:"roles"`
}

func (c Claims) Authorized(roles ...Roles) bool {
	for _, has := range c.Roles {
		for _, want := range roles {
			if has == want {
				return true
			}
		}
	}
	return false
}

type ctxKey int

const key ctxKey = 1

func setClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, key, claims)
}

func getClaims(ctx context.Context) (Claims, error) {
	claims, ok := ctx.Value(key).(Claims)
	if !ok {
		return Claims{}, errors.New("claims value missing from context")
	}
	return claims, nil
}
