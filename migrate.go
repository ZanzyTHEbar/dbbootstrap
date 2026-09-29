package db

import (
	"database/sql"
	"io/fs"

	"github.com/ZanzyTHEbar/faults-go"
	"github.com/pressly/goose/v3"
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

func newGooseProvider(db *sql.DB, migrations fs.FS, dialect goose.Dialect) (*goose.Provider, error) {
	return goose.NewProvider(
		dialect,
		db,
		migrations,
		goose.WithDisableGlobalRegistry(true),
	)
}
