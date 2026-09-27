//go:build cgo

// Package golibsql provides an adapter for github.com/tursodatabase/go-libsql.
package golibsql

import (
	"context"
	"database/sql"

	"github.com/ZanzyTHEbar/dbbootstrap"
	"github.com/ZanzyTHEbar/faults-go"
	_ "github.com/tursodatabase/go-libsql"
)

type adapter struct{}

// New returns a go-libsql adapter. go-libsql requires CGO_ENABLED=1.
func New() db.Adapter {
	return adapter{}
}

func (adapter) Open(_ context.Context, cfg db.Config) (*sql.DB, error) {
	database, err := sql.Open("libsql", cfg.DSN)
	if err != nil {
		return nil, faults.Wrap(faults.CodeConfigInvalid, "open go-libsql database", err)
	}
	return database, nil
}

func (adapter) Ping(ctx context.Context, database *sql.DB) error {
	if err := database.PingContext(ctx); err != nil {
		return faults.Wrap(faults.CodeDatabaseRead, "ping go-libsql database", err)
	}
	return nil
}

func (adapter) Dialect() string {
	return db.DialectSQLite
}
