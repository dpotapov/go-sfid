package sfid

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"testing"
)

type sqlHarnessState struct {
	mu       sync.Mutex
	execArgs []driver.NamedValue
	queryVal int64
}

type sqlHarnessDriver struct{}

type sqlHarnessConn struct{}

type sqlHarnessRows struct {
	value int64
	done  bool
}

var (
	sqlHarnessRegister sync.Once
	sqlHarnessStateMu  sync.Mutex
	sqlHarnessCurrent  *sqlHarnessState
)

func openSQLHarness(t *testing.T) (*sql.DB, *sqlHarnessState) {
	t.Helper()

	sqlHarnessRegister.Do(func() {
		sql.Register("sfid_test_driver", sqlHarnessDriver{})
	})

	state := &sqlHarnessState{}
	sqlHarnessStateMu.Lock()
	sqlHarnessCurrent = state
	sqlHarnessStateMu.Unlock()

	db, err := sql.Open("sfid_test_driver", "")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
		sqlHarnessStateMu.Lock()
		if sqlHarnessCurrent == state {
			sqlHarnessCurrent = nil
		}
		sqlHarnessStateMu.Unlock()
	})

	return db, state
}

func currentSQLHarnessState() (*sqlHarnessState, error) {
	sqlHarnessStateMu.Lock()
	defer sqlHarnessStateMu.Unlock()

	if sqlHarnessCurrent == nil {
		return nil, errors.New("sfid: sql harness state not initialized")
	}

	return sqlHarnessCurrent, nil
}

func (sqlHarnessDriver) Open(string) (driver.Conn, error) {
	return sqlHarnessConn{}, nil
}

func (sqlHarnessConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("sfid: Prepare not implemented in sql harness")
}

func (sqlHarnessConn) Close() error {
	return nil
}

func (sqlHarnessConn) Begin() (driver.Tx, error) {
	return nil, errors.New("sfid: Begin not implemented in sql harness")
}

func (sqlHarnessConn) ExecContext(_ context.Context, _ string, args []driver.NamedValue) (driver.Result, error) {
	state, err := currentSQLHarnessState()
	if err != nil {
		return nil, err
	}

	state.mu.Lock()
	state.execArgs = append(state.execArgs[:0], args...)
	state.mu.Unlock()

	return driver.RowsAffected(1), nil
}

func (sqlHarnessConn) QueryContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	state, err := currentSQLHarnessState()
	if err != nil {
		return nil, err
	}

	state.mu.Lock()
	value := state.queryVal
	state.mu.Unlock()

	return &sqlHarnessRows{value: value}, nil
}

func (r *sqlHarnessRows) Columns() []string {
	return []string{"id"}
}

func (r *sqlHarnessRows) Close() error {
	return nil
}

func (r *sqlHarnessRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}

	dest[0] = r.value
	r.done = true
	return nil
}

func TestIDDatabaseSQLExecAndScan(t *testing.T) {
	db, state := openSQLHarness(t)
	id := MustParse("d000000000000")

	if _, err := db.ExecContext(context.Background(), "insert", id); err != nil {
		t.Fatalf("ExecContext() error = %v", err)
	}

	state.mu.Lock()
	if len(state.execArgs) != 1 {
		state.mu.Unlock()
		t.Fatalf("exec args = %d, want 1", len(state.execArgs))
	}
	gotExec := state.execArgs[0].Value
	state.mu.Unlock()

	gotInt, ok := gotExec.(int64)
	if !ok {
		t.Fatalf("exec arg type = %T, want int64", gotExec)
	}
	if gotInt != int64(id) {
		t.Fatalf("exec arg = %d, want %d", gotInt, int64(id))
	}

	state.mu.Lock()
	state.queryVal = int64(id)
	state.mu.Unlock()

	var scanned ID
	if err := db.QueryRowContext(context.Background(), "select").Scan(&scanned); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if scanned != id {
		t.Fatalf("scanned ID = %v, want %v", scanned, id)
	}
}

func TestTypedAliasDatabaseSQLExecAndScan(t *testing.T) {
	db, state := openSQLHarness(t)
	uid := UserID(MustParse("d000000000000"))

	if _, err := db.ExecContext(context.Background(), "insert", uid); err != nil {
		t.Fatalf("ExecContext() error = %v", err)
	}

	state.mu.Lock()
	if len(state.execArgs) != 1 {
		state.mu.Unlock()
		t.Fatalf("exec args = %d, want 1", len(state.execArgs))
	}
	gotExec := state.execArgs[0].Value
	state.mu.Unlock()

	gotInt, ok := gotExec.(int64)
	if !ok {
		t.Fatalf("exec arg type = %T, want int64", gotExec)
	}
	if gotInt != int64(uid) {
		t.Fatalf("exec arg = %d, want %d", gotInt, int64(uid))
	}

	state.mu.Lock()
	state.queryVal = int64(uid)
	state.mu.Unlock()

	var scanned UserID
	if err := db.QueryRowContext(context.Background(), "select").Scan(&scanned); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if scanned != uid {
		t.Fatalf("scanned UserID = %v, want %v", scanned, uid)
	}
}
