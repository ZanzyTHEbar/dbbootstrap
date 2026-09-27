# dbbootstrap

[![CI](https://github.com/ZanzyTHEbar/dbbootstrap/actions/workflows/ci.yaml/badge.svg)](https://github.com/ZanzyTHEbar/dbbootstrap/actions/workflows/ci.yaml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ZanzyTHEbar/dbbootstrap.svg)](https://pkg.go.dev/github.com/ZanzyTHEbar/dbbootstrap)

Opinionated database lifecycle helpers for Go:

```text
Config -> Open -> Migrate -> SQL/Tx -> Close
```

The package keeps `database/sql` visible and leaves repositories, SQL, and
generated sqlc code to the consuming service.

## Install

```bash
go get github.com/ZanzyTHEbar/dbbootstrap@v0.1.0
```

## PostgreSQL

```go
database, err := db.Open(ctx, db.Config{
	DSN:             cfg.DatabaseURL,
	MaxOpenConns:    20,
	MaxIdleConns:    5,
	ConnMaxLifetime: time.Hour,
}, db.Postgres())
if err != nil {
	return err
}
defer database.Close()

if err := database.Migrate(ctx, migrations); err != nil {
	return err
}

_, err = database.SQL().ExecContext(ctx, "SELECT 1")
```

`Tx` exposes the native `*sql.Tx`, so generated sqlc queries can be created or
used directly by the consuming service.

## SQLite

`db.SQLite()` uses the pure-Go `modernc.org/sqlite` driver. For an in-memory
database, keep at least one idle connection so the database remains alive:

```go
database, err := db.Open(ctx, db.Config{
	DSN:          ":memory:",
	MaxOpenConns: 1,
	MaxIdleConns: 1,
}, db.SQLite())
```

## libSQL

The two upstream libSQL drivers are isolated in subpackages:

```go
import libsql "github.com/ZanzyTHEbar/dbbootstrap/libsql"

database, err := db.Open(ctx, db.Config{DSN: cfg.DatabaseURL}, libsql.New())
```

For the cgo-backed driver:

```go
import golibsql "github.com/ZanzyTHEbar/dbbootstrap/golibsql"

database, err := db.Open(ctx, db.Config{DSN: cfg.DatabaseURL}, golibsql.New())
```

`golibsql` requires `CGO_ENABLED=1` and the platform support provided by
`github.com/tursodatabase/go-libsql`.

Do not import both libSQL subpackages into one executable. Both upstream
drivers register the `libsql` `database/sql` driver name.

## Errors

Package errors use [faults-go](https://github.com/ZanzyTHEbar/faults-go).
Inspect failures with `faults.CodeOf` or `faults.IsCode`.
