// Package usergrp maintains the group of handlers for user access.
package usergrp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	usercore "github.com/tcmhoang/sservices/business/core/user"
	"github.com/tcmhoang/sservices/business/data/store/user"
	"github.com/tcmhoang/sservices/business/sys/auth"
	"github.com/tcmhoang/sservices/business/sys/validation"
	"github.com/tcmhoang/sservices/foundation/web"
)

type Handlers struct {
	user *usercore.Core
	auth *auth.Auth
}

func New(user *usercore.Core, auth *auth.Auth) *Handlers {
	return &Handlers{
		user: user,
		auth: auth,
	}
}

func (h *Handlers) Query(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	page := web.Param(r, "page")
	pageNumber, err := strconv.Atoi(page)
	if err != nil {
		return validation.NewRequestError(fmt.Errorf("invalid page format [%s]", page), http.StatusBadRequest)
	}

	rows := web.Param(r, "rows")
	rowsPerPage, err := strconv.Atoi(rows)
	if err != nil {
		return validation.NewRequestError(fmt.Errorf("invalid rows format [%s]", rows), http.StatusBadRequest)
	}

	users, err := h.user.Store.Query(ctx, pageNumber, rowsPerPage)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}

	return web.Respond(ctx, w, users, http.StatusOK)
}

func (h Handlers) QueryByID(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	userID := auth.GetUserID(ctx)

	usr, err := h.user.Store.QueryByID(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrNotFound):
			return validation.NewRequestError(err, http.StatusNotFound)
		default:
			return fmt.Errorf("ID[%s]: %w", userID, err)
		}
	}

	return web.Respond(ctx, w, usr, http.StatusOK)
}

func (h *Handlers) Create(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var nu user.NewUser
	if err := web.Decode(r, &nu); err != nil {
		return validation.NewRequestError(err, http.StatusBadRequest)
	}

	usr, err := h.user.Store.Create(ctx, nu)
	if err != nil {
		if errors.Is(err, user.ErrUniqueEmail) {
			return validation.NewRequestError(err, http.StatusConflict)
		}
		return fmt.Errorf("create: usr[%+v]: %w", usr, err)
	}

	return web.Respond(ctx, w, usr, http.StatusCreated)
}

func (h *Handlers) Update(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

	var uu user.UpdateUser
	if err := web.Decode(r, &uu); err != nil {
		return validation.NewRequestError(err, http.StatusBadRequest)
	}

	userID := auth.GetUserID(ctx)

	usr, err := h.user.Store.QueryByID(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrNotFound):
			return validation.NewRequestError(err, http.StatusNotFound)
		default:
			return fmt.Errorf("querybyid: userID[%s]: %w", userID, err)
		}
	}

	usr, err = h.user.Store.Update(ctx, usr, uu)
	if err != nil {
		return fmt.Errorf("update: userID[%s] uu[%+v]: %w", userID, uu, err)
	}

	return web.Respond(ctx, w, usr, http.StatusOK)
}

func (h *Handlers) Delete(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	claims, err := auth.GetClaims(ctx)
	if err != nil {
		return errors.New("claims missing from ctx")
	}

	userID := auth.GetUserID(ctx)

	usr, err := h.user.Store.QueryByID(ctx, userID)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrNotFound):
			return web.Respond[interface{}](ctx, w, nil, http.StatusNoContent)
		default:
			return fmt.Errorf("querybyid: userID[%s]: %w", userID, err)
		}
	}

	if err := h.user.Store.Delete(ctx, claims, usr); err != nil {
		return fmt.Errorf("delete: userID[%s]: %w", userID, err)
	}

	return web.Respond[interface{}](ctx, w, nil, http.StatusNoContent)
}

func (h *Handlers) Token(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	email, pass, ok := r.BasicAuth()
	if !ok {
		return validation.NewRequestError(errors.New("must provide email and password in Basic auth"), http.StatusBadRequest)
	}

	addr, err := mail.ParseAddress(email)
	if err != nil {
		return validation.NewRequestError(errors.New("invalid email format"), http.StatusBadRequest)
	}

	usr, err := h.user.Authenticate(ctx, *addr, pass)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrNotFound):
			return validation.NewRequestError(err, http.StatusNotFound)
		case errors.Is(err, usercore.ErrAuthenticationFailure):
			return validation.NewRequestError(err, http.StatusMethodNotAllowed)
		default:
			return fmt.Errorf("authenticate: %w", err)
		}
	}

	var roles []auth.Role

	for _, r := range usr.Roles {
		switch strings.ToLower(r) {
		case "admin":
			roles = append(roles, auth.Admin)
		case "user":
			roles = append(roles, auth.User)

		default:

		}
	}

	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   usr.ID.String(),
			Issuer:    "service project",
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
		Roles: roles,
	}

	var tkn struct {
		Token string `json:"token"`
	}

	tkn.Token, err = h.auth.GeneratingToken(claims)
	if err != nil {
		return fmt.Errorf("generatetoken: %w", err)
	}

	return web.Respond(ctx, w, tkn, http.StatusOK)
}
