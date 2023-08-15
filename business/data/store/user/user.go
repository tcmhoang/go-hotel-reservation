// Package user supports CRUD operations
package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/tcmhoang/sservices/business/sys/auth"
	"github.com/tcmhoang/sservices/business/sys/database"
	"github.com/tcmhoang/sservices/business/sys/validation"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotFound              = errors.New("user not found")
	ErrInvalidEmail          = errors.New("email is not valid")
	ErrUniqueEmail           = errors.New("email is not unique")
	ErrForbidden             = errors.New("forbidden operation")
	ErrAuthenticationFailure = errors.New("authentication failed")
)

type Core struct {
	log *zap.SugaredLogger
	db  *sqlx.DB
}

func NewCore(log *zap.SugaredLogger, db *sqlx.DB) *Core {
	return &Core{
		log: log,
		db:  db,
	}
}

func (c *Core) Create(ctx context.Context, nu NewUser) (User, error) {

	if err := validation.Check(nu); err != nil {
		return User{}, fmt.Errorf("validating data: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(nu.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("generating password hash: %w", err)
	}

	now := time.Now()

	usr := User{
		ID:           uuid.New(),
		Name:         nu.Name,
		Email:        nu.Email,
		PasswordHash: hash,
		Roles:        nu.Roles,
		Department:   nu.Department,
		Enabled:      true,
		DateCreated:  now,
		DateUpdated:  now,
	}

	const q = `
		INSERT INTO users
			(user_id, name, email, password_hash, roles, enabled, date_created, date_updated)
		VALUES
			(:user_id, :name, :email, :password_hash, :roles, :enabled, :date_created, :date_updated)
		`

	if err := database.NamedExecContext(ctx, c.log, c.db, q, usr); err != nil {
		if errors.Is(err, database.ErrDBDuplicatedEntry) {
			return User{}, fmt.Errorf("create: %w", ErrUniqueEmail)
		}
		return User{}, fmt.Errorf("inserting user: %w", err)
	}

	return usr, nil
}

func (c *Core) Update(ctx context.Context, usr User, uu UpdateUser) (User, error) {

	if err := validation.Check(uu); err != nil {
		return User{}, fmt.Errorf("validating data: %w", err)
	}

	if uu.Name != nil {
		usr.Name = *uu.Name
	}
	if uu.Email != nil {
		usr.Email = *uu.Email
	}
	if uu.Roles != nil {
		usr.Roles = uu.Roles
	}
	if uu.Password != nil {
		pw, err := bcrypt.GenerateFromPassword([]byte(*uu.Password), bcrypt.DefaultCost)
		if err != nil {
			return User{}, fmt.Errorf("generating password hash: %w", err)
		}
		usr.PasswordHash = pw
	}
	if uu.Department != nil {
		usr.Department = *uu.Department
	}
	if uu.Enabled != nil {
		usr.Enabled = *uu.Enabled
	}
	usr.DateUpdated = time.Now()

	const q = `
		UPDATE
			users
		SET 
			"name" = :name,
			"email" = :email,
			"roles" = :roles,
			"password_hash" = :password_hash,
			"date_updated" = :date_updated
		WHERE
			user_id = :user_id
		`

	if err := database.NamedExecContext(ctx, c.log, c.db, q, usr); err != nil {
		if errors.Is(err, database.ErrDBDuplicatedEntry) {
			return User{}, ErrUniqueEmail
		}
		return User{}, fmt.Errorf("updating userID[%s]: %w", usr.ID, err)
	}

	return usr, nil
}

func (c *Core) Delete(ctx context.Context, claims auth.Claims, usr User) error {

	if !claims.Authorized(auth.Admin) && claims.Subject != usr.ID.String() {
		return ErrNotFound
	}

	data := struct {
		UserID string `db:"user_id"`
	}{
		UserID: usr.ID.String(),
	}

	const q = `
	DELETE FROM
		users
	WHERE
		user_id = :user_id
		`

	if err := database.NamedExecContext(ctx, c.log, c.db, q, data); err != nil {
		return fmt.Errorf("deleting userID[%s]: %w", usr.ID, err)
	}

	return nil
}

func (c *Core) Query(ctx context.Context, pageNumber int, rowsPerPage int) ([]User, error) {

	data := struct {
		Offset      int `db:"offset"`
		RowsPerPage int `db:"rows_per_page"`
	}{
		Offset:      (pageNumber - 1) * rowsPerPage,
		RowsPerPage: rowsPerPage,
	}

	const q = `
	SELECT
		*
	FROM 
		users
	ORDER BY
		user_id
	OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY
	`

	var usrs []User
	if err := database.NamedQueryAggregation(ctx, c.log, c.db, q, data, &usrs); err != nil {
		return nil, fmt.Errorf("selecting users: %w", err)
	}

	return usrs, nil
}

func (c *Core) QueryByID(ctx context.Context, userID uuid.UUID) (User, error) {
	data := struct {
		UserID string `db:"user_id"`
	}{
		UserID: userID.String(),
	}

	const q = `
		SELECT
			*
		FROM
			users
		WHERE 
			user_id = :user_id
		`
	var usr User
	if err := database.NamedQueryScalar(ctx, c.log, c.db, q, data, &usr); err != nil {
		if errors.Is(err, database.ErrDBNotFound) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("selecting userID[%q]: %w", userID, err)
	}

	return usr, nil
}

func (c *Core) QueryByEmail(ctx context.Context, email mail.Address) (User, error) {
	data := struct {
		Email string `db:"email"`
	}{
		Email: email.Address,
	}

	const q = `
		SELECT
			*
		FROM
			users
		WHERE
			email = :email
		`
	var usr User
	if err := database.NamedQueryScalar(ctx, c.log, c.db, q, data, &usr); err != nil {
		if errors.Is(err, database.ErrDBNotFound) {
			return User{}, ErrNotFound
		}
		return User{}, fmt.Errorf("selecting email[%q]: %w", email, err)
	}

	return usr, nil

}

func (c *Core) Authenticate(ctx context.Context, email mail.Address, password string) (User, error) {
	usr, err := c.QueryByEmail(ctx, email)
	if err != nil {
		return User{}, fmt.Errorf("query: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword(usr.PasswordHash, []byte(password)); err != nil {
		return User{}, ErrAuthenticationFailure
	}

	return usr, nil
}
