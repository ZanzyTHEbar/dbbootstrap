package db

import (
	"context"
	"database/sql"

	"github.com/ZanzyTHEbar/faults-go"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type postgresAdapter struct{}

// Postgres returns the PostgreSQL database/sql adapter backed by pgx.
func Postgres() Adapter {
	return postgresAdapter{}
}

func (postgresAdapter) Open(_ context.Context, cfg Config) (*sql.DB, error) {
	database, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, faults.Wrap(faults.CodeConfigInvalid, "open postgres database", err)
	}
	return database, nil
}

func (postgresAdapter) Ping(ctx context.Context, db *sql.DB) error {
	if err := db.PingContext(ctx); err != nil {
		return faults.Wrap(faults.CodeDatabaseRead, "ping postgres database", err)
	}
	return nil
}

func (postgresAdapter) Dialect() string {
	return DialectPostgres
}
