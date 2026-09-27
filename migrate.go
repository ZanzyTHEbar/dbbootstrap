package db

import (
	"context"
	"database/sql"
	"io/fs"

	"github.com/ZanzyTHEbar/faults-go"
	"github.com/pressly/goose/v3"
	goosedb "github.com/pressly/goose/v3/database"
)

func gooseDialect(name string) (goose.Dialect, error) {
	switch name {
	case DialectPostgres:
		return goose.DialectPostgres, nil
	case DialectSQLite:
		return goose.DialectSQLite3, nil
	default:
		return "", faults.New(faults.CodeConfigInvalid, "unsupported migration dialect", "dialect", name)
	}
}

func ensureGooseVersionTable(ctx context.Context, db *sql.DB, dialect goose.Dialect) error {
	store, err := goosedb.NewStore(dialect, goose.DefaultTablename)
	if err != nil {
		return err
	}
	if _, err := store.ListMigrations(ctx, db); err == nil {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := store.CreateVersionTable(ctx, tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := store.Insert(ctx, tx, goosedb.InsertRequest{Version: 0}); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func newGooseProvider(db *sql.DB, migrations fs.FS, dialect goose.Dialect) (*goose.Provider, error) {
	return goose.NewProvider(
		dialect,
		db,
		migrations,
		goose.WithDisableGlobalRegistry(true),
	)
}
