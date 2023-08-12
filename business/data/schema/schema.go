// Package dbschema contains the database schema, migrations and seeding data.
package schema

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"github.com/ardanlabs/darwin/v3"
	"github.com/ardanlabs/darwin/v3/dialects/postgres"
	"github.com/ardanlabs/darwin/v3/drivers/generic"
	"github.com/jmoiron/sqlx"
	"github.com/tcmhoang/sservices/business/sys/database"
)

var (
	//go:embed sql/schema.sql
	schemaDoc string

	//go:embed sql/seed.sql
	seedDoc string

	//go:embed sql/delete.sql
	deleteDoc string
)

func Migrate(ctx context.Context, db *sqlx.DB) error {
	if err := database.StatusCheck(ctx, db); err != nil {
		return fmt.Errorf("status check databse: %w", err)
	}

	driver, err := generic.New(db.DB, postgres.Dialect{})
	if err != nil {
		return fmt.Errorf("construct darwin driver: %w", err)
	}

	d := darwin.New(driver, darwin.ParseMigrations(schemaDoc))

	return d.Migrate()

}

func DeleteAll(ctx context.Context, db *sqlx.DB) error {
	return auxExec(ctx, db, deleteDoc)
}

func Seed(ctx context.Context, db *sqlx.DB) error {
	return auxExec(ctx, db, seedDoc)
}

func auxExec(ctx context.Context, db *sqlx.DB, doc string) (rerr error) {
	if err := database.StatusCheck(ctx, db); err != nil {
		return fmt.Errorf("status check databse: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if errp := tx.Rollback(); errp != nil {
			if errors.Is(errp, sql.ErrTxDone) {
				return
			}
			rerr = fmt.Errorf("rollback: %w", err)
			return
		}
	}()

	if _, err := tx.Exec(doc); err != nil {
		return fmt.Errorf("execute: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return rerr
}
