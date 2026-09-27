package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/ZanzyTHEbar/faults-go"
)

type testDriverState struct {
	mu          sync.Mutex
	pingErr     error
	commitErr   error
	rollbackErr error
	pings       int
	closes      int
	commits     int
	rollbacks   int
}

type testAdapter struct {
	state *testDriverState
	pool  *sql.DB
}

func (a *testAdapter) Open(context.Context, Config) (*sql.DB, error) {
	a.pool = sql.OpenDB(testConnector{state: a.state})
	return a.pool, nil
}

func (*testAdapter) Ping(ctx context.Context, db *sql.DB) error {
	return db.PingContext(ctx)
}

type testConnector struct {
	state *testDriverState
}

func (c testConnector) Connect(context.Context) (driver.Conn, error) {
	return &testConn{state: c.state}, nil
}

func (c testConnector) Driver() driver.Driver {
	return testDriver{state: c.state}
}

type testDriver struct {
	state *testDriverState
}

func (d testDriver) Open(string) (driver.Conn, error) {
	return &testConn{state: d.state}, nil
}

type testConn struct {
	state *testDriverState
}

func (c *testConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}

func (c *testConn) Close() error {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	c.state.closes++
	return nil
}

func (c *testConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *testConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return &testTx{state: c.state}, nil
}

func (c *testConn) Ping(context.Context) error {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	c.state.pings++
	return c.state.pingErr
}

type testTx struct {
	state *testDriverState
}

func (tx *testTx) Commit() error {
	tx.state.mu.Lock()
	defer tx.state.mu.Unlock()
	tx.state.commits++
	return tx.state.commitErr
}

func (tx *testTx) Rollback() error {
	tx.state.mu.Lock()
	defer tx.state.mu.Unlock()
	tx.state.rollbacks++
	return tx.state.rollbackErr
}

func (s *testDriverState) counts() (pings, closes, commits, rollbacks int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pings, s.closes, s.commits, s.rollbacks
}

func openTestDB(t *testing.T, state *testDriverState) *DB {
	t.Helper()
	db, err := Open(context.Background(), Config{MaxOpenConns: 1, MaxIdleConns: 1}, &testAdapter{state: state})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestOpenClosesPoolWhenPingFails(t *testing.T) {
	wantErr := errors.New("ping failed")
	state := &testDriverState{pingErr: wantErr}
	adapter := &testAdapter{state: state}

	db, err := Open(context.Background(), Config{}, adapter)
	if db != nil {
		t.Fatal("Open() returned a database after ping failed")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Open() error = %v, want it to wrap %v", err, wantErr)
	}
	if !faults.IsCode(err, faults.CodeDatabaseRead) {
		t.Fatalf("Open() fault code = %q, want %q", faults.CodeOf(err), faults.CodeDatabaseRead)
	}
	if adapter.pool == nil {
		t.Fatal("adapter did not create a pool")
	}
	_, closes, _, _ := state.counts()
	if closes == 0 {
		t.Fatal("Open() did not close the pool after ping failure")
	}
}

func TestOpenAppliesPoolSettings(t *testing.T) {
	db, err := Open(context.Background(), Config{MaxOpenConns: 3}, &testAdapter{state: &testDriverState{}})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	if got := db.SQL().Stats().MaxOpenConnections; got != 3 {
		t.Fatalf("MaxOpenConnections = %d, want 3", got)
	}
}

func TestSQLiteOpenAndMigrate(t *testing.T) {
	db, err := Open(context.Background(), Config{
		DSN:          ":memory:",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	}, SQLite())
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer db.Close()

	migrations := fstest.MapFS{
		"001_create_users.sql": &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE users (id INTEGER PRIMARY KEY);

-- +goose Down
DROP TABLE users;
`)},
	}
	if err := db.Migrate(context.Background(), migrations); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if _, err := db.SQL().Exec("INSERT INTO users (id) VALUES (1)"); err != nil {
		t.Fatalf("insert after migration error = %v", err)
	}
}

func TestTxCommitsOnSuccess(t *testing.T) {
	state := &testDriverState{}
	db := openTestDB(t, state)

	if err := db.Tx(context.Background(), func(tx *sql.Tx) error {
		if tx == nil {
			t.Fatal("transaction callback received a nil transaction")
		}
		return nil
	}); err != nil {
		t.Fatalf("Tx() error = %v", err)
	}
	_, _, commits, rollbacks := state.counts()
	if commits != 1 || rollbacks != 0 {
		t.Fatalf("commit/rollback counts = %d/%d, want 1/0", commits, rollbacks)
	}
}

func TestTxRollsBackCallbackError(t *testing.T) {
	wantErr := errors.New("callback failed")
	state := &testDriverState{}
	db := openTestDB(t, state)

	err := db.Tx(context.Background(), func(*sql.Tx) error { return wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("Tx() error = %v, want it to wrap %v", err, wantErr)
	}
	_, _, commits, rollbacks := state.counts()
	if commits != 0 || rollbacks != 1 {
		t.Fatalf("commit/rollback counts = %d/%d, want 0/1", commits, rollbacks)
	}
}

func TestTxJoinsCallbackAndRollbackErrors(t *testing.T) {
	callbackErr := errors.New("callback failed")
	rollbackErr := errors.New("rollback failed")
	state := &testDriverState{rollbackErr: rollbackErr}
	db := openTestDB(t, state)

	err := db.Tx(context.Background(), func(*sql.Tx) error { return callbackErr })
	if !errors.Is(err, callbackErr) || !faults.IsCode(err, faults.CodeDatabaseWrite) {
		t.Fatalf("Tx() error = %v, want a database fault wrapping the callback error", err)
	}
}

func TestTxReturnsCommitError(t *testing.T) {
	wantErr := errors.New("commit failed")
	state := &testDriverState{commitErr: wantErr}
	db := openTestDB(t, state)

	err := db.Tx(context.Background(), func(*sql.Tx) error { return nil })
	if !errors.Is(err, wantErr) {
		t.Fatalf("Tx() error = %v, want it to wrap %v", err, wantErr)
	}
	_, _, commits, rollbacks := state.counts()
	if commits != 1 || rollbacks != 0 {
		t.Fatalf("commit/rollback counts = %d/%d, want 1/0", commits, rollbacks)
	}
}

func TestTxRollsBackOnPanic(t *testing.T) {
	state := &testDriverState{}
	db := openTestDB(t, state)
	wantPanic := "callback panic"

	func() {
		defer func() {
			if got := recover(); got != wantPanic {
				t.Fatalf("panic = %v, want %v", got, wantPanic)
			}
		}()
		_ = db.Tx(context.Background(), func(*sql.Tx) error { panic(wantPanic) })
	}()
	_, _, commits, rollbacks := state.counts()
	if commits != 0 || rollbacks != 1 {
		t.Fatalf("commit/rollback counts = %d/%d, want 0/1", commits, rollbacks)
	}
}

var _ driver.Connector = testConnector{}
var _ driver.Pinger = (*testConn)(nil)
var _ driver.ConnBeginTx = (*testConn)(nil)
