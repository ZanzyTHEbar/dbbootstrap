// Package libsql provides an adapter for github.com/tursodatabase/libsql-client-go.
package libsql

import (
	"context"
	"database/sql"

	"github.com/ZanzyTHEbar/dbbootstrap"
	"github.com/ZanzyTHEbar/faults-go"
	client "github.com/tursodatabase/libsql-client-go/libsql"
)

type adapter struct {
	options []client.Option
}

// New returns a libsql-client-go adapter.
func New(options ...client.Option) db.Adapter {
	return adapter{options: append([]client.Option(nil), options...)}
}

func (a adapter) Open(_ context.Context, cfg db.Config) (*sql.DB, error) {
	connector, err := client.NewConnector(cfg.DSN, a.options...)
	if err != nil {
		return nil, faults.Wrap(faults.CodeConfigInvalid, "configure libsql client", err)
	}
	return sql.OpenDB(connector), nil
}

func (adapter) Ping(ctx context.Context, database *sql.DB) error {
	if err := database.PingContext(ctx); err != nil {
		return faults.Wrap(faults.CodeDatabaseRead, "ping libsql database", err)
	}
	return nil
}

func (adapter) Dialect() string {
	return db.DialectSQLite
}
