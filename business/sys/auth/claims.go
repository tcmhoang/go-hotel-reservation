package auth

import (
	"context"
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

type Role int

const (
	Admin Role = iota
	User
)

type Claims struct {
	jwt.RegisteredClaims
	Roles []Role `json:"roles"`
}

func (c Claims) Authorized(roles ...Role) bool {
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

func SetClaims(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, key, claims)
}

func GetClaims(ctx context.Context) (Claims, error) {
	claims, ok := ctx.Value(key).(Claims)
	if !ok {
		return Claims{}, errors.New("claims value missing from context")
	}
	return claims, nil
}
