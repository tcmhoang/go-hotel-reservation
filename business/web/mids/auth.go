package mids

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/tcmhoang/sservices/business/sys/auth"
	"github.com/tcmhoang/sservices/business/sys/validation"
	"github.com/tcmhoang/sservices/foundation/web"
)

func Authenticate(a *auth.Auth) web.Middleware {
	return func(handler web.Handler) web.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			authstr := r.Header.Get("authorization")

			authstrs := strings.Split(authstr, " ")
			if len(authstrs) != 2 || strings.ToLower(authstrs[0]) != "bearer" {
				return validation.NewRequestError(
					errors.New("expected authorization header format: bearer <token>"),
					http.StatusUnauthorized,
				)
			}

			claims, err := a.ValidateToken(authstrs[1])
			if err != nil {
				return validation.NewRequestError(err, http.StatusUnauthorized)
			}

			ctx = auth.SetClaims(ctx, claims)

			return handler(ctx, w, r)

		}
	}
}

func Authorize(roles ...auth.Role) web.Middleware {
	return func(handler web.Handler) web.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			claims, err := auth.GetClaims(ctx)
			if err != nil {
				return validation.NewRequestError(
					fmt.Errorf("not authorized for that action, no claims"),
					http.StatusForbidden,
				)
			}
			if !claims.Authorized(roles...) {
				return validation.NewRequestError(
					fmt.Errorf("not authorized for that action, got %v roles %v", claims.Roles, roles),
					http.StatusForbidden,
				)
			}
			return handler(ctx, w, r)
		}
	}
}
