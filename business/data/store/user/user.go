// Package user supports CRUD operations
package user

import (
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type Core struct {
	*zap.SugaredLogger
	*sqlx.DB
}

func NewCore(log *zap.SugaredLogger, db *sqlx.DB) *Core {
	return &Core{
		log,
		db,
	}
}
