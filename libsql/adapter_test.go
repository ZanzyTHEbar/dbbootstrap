package libsql

import (
	"context"
	"testing"

	db "github.com/ZanzyTHEbar/dbbootstrap"
	client "github.com/tursodatabase/libsql-client-go/libsql"
)

func TestLocalDatabase(t *testing.T) {
	database, err := db.Open(context.Background(), db.Config{
		DSN:          "file::memory:?cache=shared",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}, New())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer database.Close()

	if _, err := database.SQL().ExecContext(context.Background(), "CREATE TABLE users (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create table error = %v", err)
	}
}

func TestNewCopiesOptions(t *testing.T) {
	options := []client.Option{client.WithTls(true)}
	adapter := New(options...)
	options[0] = nil

	database, err := adapter.Open(context.Background(), db.Config{DSN: "file::memory:?cache=shared"})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
