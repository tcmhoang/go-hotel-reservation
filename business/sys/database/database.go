// Package database provides support for accessing the database
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/tcmhoang/sservices/foundation/web"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

const (
	uniqueViolationCode = "23505"
	undefinedTableCode  = "42P01"
)

var (
	ErrDBNotFound        = sql.ErrNoRows
	ErrDBDuplicatedEntry = errors.New("duplicated entry")
	ErrUndefinedTable    = errors.New("undefined table")
)

type Config struct {
	User         string
	Password     string
	Host         string
	Name         string
	Schema       string
	MaxIdleConns int
	MaxOpenConns int
	DisableTLS   bool
}

func Open(cfg Config) (*sqlx.DB, error) {
	sslMode := "require"
	if cfg.DisableTLS {
		sslMode = "disable"
	}

	query := make(url.Values)
	query.Set("sslmode", sslMode)
	query.Set("timezone", "utc")

	if cfg.Schema != "" {
		query.Set("search_path", cfg.Schema)
	}

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Host,
		Path:     cfg.Name,
		RawQuery: query.Encode(),
	}

	db, err := sqlx.Open("pgx", u.String())
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetMaxOpenConns(cfg.MaxOpenConns)

	return db, nil
}

func StatusCheck(ctx context.Context, db *sqlx.DB) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second)
		defer cancel()
	}

	var pingError error

	for attempts := 1; ; attempts++ {
		pingError = db.Ping()
		if pingError == nil {
			break
		}
		time.Sleep(time.Duration(attempts) * 100 * time.Millisecond)
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	const qraw = `SELECT true`
	var tmp bool
	return db.QueryRowContext(ctx, qraw).Scan(&tmp)
}

func NamedExecContext[A any](ctx context.Context, log *zap.SugaredLogger, db sqlx.ExtContext, query string, data A) error {
	q := queryString(query, data)
	log.Infow("database.NamedExecuteContext", "traceid", web.GetTraceID(ctx), "query", q)

	ctx, span := web.AddSpan(ctx, "business.sys.database.exec", attribute.String("query", q))
	defer span.End()

	if _, err := sqlx.NamedExecContext(ctx, db, query, data); err != nil {
		if qperr, ok := err.(*pgconn.PgError); ok {
			switch qperr.Code {
			case undefinedTableCode:
				return ErrUndefinedTable
			case uniqueViolationCode:
				return ErrDBDuplicatedEntry
			}
		}
		return err
	}
	return nil
}

func NamedQueryAggregation[A, B any](ctx context.Context, log *zap.SugaredLogger, db sqlx.ExtContext, query string, data A, dest *[]B) error {
	q := queryString(query, data)
	log.Infow("database.NamedQueryAggregation", "traceid", web.GetTraceID(ctx), "query", q)

	ctx, span := web.AddSpan(ctx, "business.sys.database.queryaggregation", attribute.String("query", q))
	defer span.End()

	rows, err := sqlx.NamedQueryContext(ctx, db, query, data)
	if err != nil {
		if pqerr, ok := err.(*pgconn.PgError); ok && pqerr.Code == undefinedTableCode {
			return ErrUndefinedTable
		}
		return err
	}
	defer rows.Close()

	var bs []B
	for rows.Next() {
		v := new(B)
		if err := rows.StructScan(v); err != nil {
			return err
		}
		bs = append(bs, *v)
	}
	*dest = bs

	return nil

}

func NamedQueryScalar[A, B any](ctx context.Context, log *zap.SugaredLogger, db sqlx.ExtContext, query string, data A, dest B) error {

	q := queryString(query, data)
	log.Infow("database.NamedQueryScalar", "traceid", web.GetTraceID(ctx), "query", q)

	ctx, span := web.AddSpan(ctx, "business.sys.database.queryscalar", attribute.String("query", q))
	defer span.End()

	rows, err := sqlx.NamedQueryContext(ctx, db, query, data)

	if err != nil {
		if pqerr, ok := err.(*pgconn.PgError); ok && pqerr.Code == undefinedTableCode {
			return ErrUndefinedTable
		}
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrDBNotFound
	}

	if err := rows.StructScan(dest); err != nil {
		return err
	}

	return nil

}

func queryString(query string, args any) string {
	query, params, err := sqlx.Named(query, args)
	if err != nil {
		return err.Error()
	}

	for _, params := range params {
		var value string

		switch v := params.(type) {
		case string:
			value = fmt.Sprintf("'%s'", v)
		case []byte:
			value = fmt.Sprintf("'%s'", string(v))
		default:
			value = fmt.Sprintf("'%v'", v)
		}
		query = strings.Replace(query, "?", value, 1)
	}

	query = strings.ReplaceAll(query, "\t", "")
	query = strings.ReplaceAll(query, "\n", " ")

	return strings.Trim(query, " ")

}
