package db

import (
	"context"
	"database/sql"

	"github.com/ZanzyTHEbar/faults-go"
	_ "modernc.org/sqlite"
)

type sqliteAdapter struct{}

// SQLite returns a pure-Go SQLite database/sql adapter.
func SQLite() Adapter {
	return sqliteAdapter{}
}

func (sqliteAdapter) Open(_ context.Context, cfg Config) (*sql.DB, error) {
	database, err := sql.Open("sqlite", cfg.DSN)
	if err != nil {
		return nil, faults.Wrap(faults.CodeConfigInvalid, "open sqlite database", err)
	}
	return database, nil
}

func (sqliteAdapter) Ping(ctx context.Context, db *sql.DB) error {
	if err := db.PingContext(ctx); err != nil {
		return faults.Wrap(faults.CodeDatabaseRead, "ping sqlite database", err)
	}
	return nil
}

func (sqliteAdapter) Dialect() string {
	return DialectSQLite
}
