package libsql

import (
	"context"
	"testing"

	db "github.com/ZanzyTHEbar/dbbootstrap"
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
