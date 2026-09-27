package db

import (
	"context"
	"database/sql"
	"io/fs"
	"time"

	"github.com/ZanzyTHEbar/faults-go"
)

const (
	DialectPostgres = "postgres"
	DialectSQLite   = "sqlite3"
)

// Config contains the connection string and database/sql pool settings.
type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// Adapter opens a database/sql pool and provides its health check.
type Adapter interface {
	Open(context.Context, Config) (*sql.DB, error)
	Ping(context.Context, *sql.DB) error
}

// Adapters may implement Dialect() string to select the migration dialect.
type dialectAdapter interface {
	Dialect() string
}

// DB owns a database/sql connection pool.
type DB struct {
	sql     *sql.DB
	adapter Adapter
	dialect string
}

// Open creates a database pool, applies its settings, and verifies connectivity.
func Open(ctx context.Context, cfg Config, adapter Adapter) (*DB, error) {
	if adapter == nil {
		return nil, faults.New(faults.CodeConfigInvalid, "database adapter is nil")
	}

	sqlDB, err := adapter.Open(ctx, cfg)
	if err != nil {
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
		return nil, wrapFault(faults.CodeDatabaseWrite, "open database", err)
	}
	if sqlDB == nil {
		return nil, faults.New(faults.CodeInvariantFailed, "database adapter returned a nil database")
	}

	applyPoolConfig(sqlDB, cfg)
	if err := adapter.Ping(ctx, sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, wrapFault(faults.CodeDatabaseRead, "ping database", err)
	}

	dialect := DialectPostgres
	if dialectAdapter, ok := adapter.(dialectAdapter); ok {
		dialect = dialectAdapter.Dialect()
	}
	return &DB{sql: sqlDB, adapter: adapter, dialect: dialect}, nil
}

func applyPoolConfig(db *sql.DB, cfg Config) {
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
}

// SQL returns the underlying pool. The DB remains responsible for closing it.
func (db *DB) SQL() *sql.DB {
	return db.sql
}

// Ping checks the database connection using the configured adapter.
func (db *DB) Ping(ctx context.Context) error {
	if err := db.adapter.Ping(ctx, db.sql); err != nil {
		return wrapFault(faults.CodeDatabaseRead, "ping database", err)
	}
	return nil
}

// Migrate applies pending SQL migrations from migrations.
func (db *DB) Migrate(ctx context.Context, migrations fs.FS) error {
	if migrations == nil {
		return faults.New(faults.CodeConfigInvalid, "migration filesystem is nil")
	}

	dialect, err := gooseDialect(db.dialect)
	if err != nil {
		return err
	}

	provider, err := newGooseProvider(db.sql, migrations, dialect)
	if err != nil {
		return wrapFault(faults.CodeConfigInvalid, "create migration provider", err)
	}
	if err := ensureGooseVersionTable(ctx, db.sql, dialect); err != nil {
		return wrapFault(faults.CodeDatabaseWrite, "initialize migration tracking", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return wrapFault(faults.CodeDatabaseWrite, "migrate database", err)
	}
	return nil
}

// Tx runs fn in a transaction, committing on success and rolling back on error.
func (db *DB) Tx(ctx context.Context, fn func(*sql.Tx) error) error {
	if fn == nil {
		return faults.New(faults.CodeConfigInvalid, "transaction callback is nil")
	}

	tx, err := db.sql.BeginTx(ctx, nil)
	if err != nil {
		return wrapFault(faults.CodeDatabaseWrite, "begin transaction", err)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback()
			panic(recovered)
		}
	}()

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return wrapFault(
				faults.CodeDatabaseWrite,
				"transaction callback failed and rollback failed",
				err,
				"rollback_error", rollbackErr,
			)
		}
		return wrapFault(faults.CodeDatabaseWrite, "transaction callback failed", err)
	}
	if err := tx.Commit(); err != nil {
		return wrapFault(faults.CodeDatabaseWrite, "commit transaction", err)
	}
	return nil
}

// Close closes the underlying connection pool.
func (db *DB) Close() error {
	if err := db.sql.Close(); err != nil {
		return wrapFault(faults.CodeDatabaseWrite, "close database", err)
	}
	return nil
}

func wrapFault(code faults.Code, message string, cause error, fields ...any) error {
	if existing := faults.CodeOf(cause); existing != faults.CodeUnknown {
		code = existing
	}
	return faults.Wrap(code, message, cause, fields...)
}
