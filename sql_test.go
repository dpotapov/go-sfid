package sfid

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"sync"
	"testing"
)

type sqlHarnessState struct {
	mu       sync.Mutex
	execArgs []driver.NamedValue
	queryVal any
}

type sqlHarnessDriver struct{}

type sqlHarnessConn struct{}

type sqlHarnessRows struct {
	value any
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

	if _, err := db.ExecContext(t.Context(), "insert", id); err != nil {
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
	if err := db.QueryRowContext(t.Context(), "select").Scan(&scanned); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if scanned != id {
		t.Fatalf("scanned ID = %v, want %v", scanned, id)
	}
}

func TestTypedAliasDatabaseSQLExecAndScan(t *testing.T) {
	db, state := openSQLHarness(t)
	uid := UserID(MustParse("d000000000000"))

	if _, err := db.ExecContext(t.Context(), "insert", uid); err != nil {
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
	if err := db.QueryRowContext(t.Context(), "select").Scan(&scanned); err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if scanned != uid {
		t.Fatalf("scanned UserID = %v, want %v", scanned, uid)
	}
}

func TestIDValue(t *testing.T) {
	t.Helper()

	id := MustParse("d000000000000")

	got, err := id.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}
	if gotInt, ok := got.(int64); !ok || gotInt != int64(id) {
		t.Fatalf("Value() = %v (%T), want int64(%d)", got, got, int64(id))
	}

	var zero ID
	got, err = zero.Value()
	if err != nil {
		t.Fatalf("Value() on zero error = %v", err)
	}
	if got != nil {
		t.Fatalf("Value() on zero = %v, want nil", got)
	}
}

func TestTypedValue(t *testing.T) {
	t.Helper()

	uid := UserID(MustParse("d000000000000"))

	got, err := uid.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}
	if gotInt, ok := got.(int64); !ok || gotInt != int64(uid) {
		t.Fatalf("Value() = %v (%T), want int64(%d)", got, got, int64(uid))
	}

	var zero UserID
	got, err = zero.Value()
	if err != nil {
		t.Fatalf("Value() on zero error = %v", err)
	}
	if got != nil {
		t.Fatalf("Value() on zero = %v, want nil", got)
	}
}

func TestIDScan(t *testing.T) {
	want := MustParse("d000000000000")
	be := make([]byte, 8)
	binary.BigEndian.PutUint64(be, uint64(want))
	small := ID(42)

	tests := []struct {
		name    string
		src     any
		want    ID
		wantErr error
	}{
		{name: "nil", src: nil, want: 0},
		{name: "int64", src: int64(want), want: want},
		{name: "int", src: int(want), want: want},
		{name: "int32", src: int32(small), want: small},
		{name: "uint64", src: uint64(want), want: want},
		{name: "string", src: "d000000000000", want: want},
		{name: "empty string", src: "", want: 0},
		{name: "bytes big-endian", src: be, want: want},
		{name: "bytes string", src: []byte("d000000000000"), want: want},
		{name: "bytes overflow", src: func() []byte {
			b := make([]byte, 8)
			binary.BigEndian.PutUint64(b, uint64(math.MaxInt64)+1)
			return b
		}(), wantErr: ErrInvalidScanValue},
		{name: "negative int64", src: int64(-1), wantErr: ErrInvalidScanValue},
		{name: "uint64 overflow", src: uint64(math.MaxInt64) + 1, wantErr: ErrInvalidScanValue},
		{name: "float64", src: float64(1), wantErr: ErrInvalidScanSource},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got ID
			err := got.Scan(tc.src)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Scan() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Scan() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("Scan() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTypedScan(t *testing.T) {
	want := UserID(MustParse("d000000000000"))
	be := make([]byte, 8)
	binary.BigEndian.PutUint64(be, uint64(want))
	small := UserID(42)

	tests := []struct {
		name    string
		src     any
		want    UserID
		wantErr error
	}{
		{name: "nil", src: nil, want: 0},
		{name: "int64", src: int64(want), want: want},
		{name: "int", src: int(want), want: want},
		{name: "int32", src: int32(small), want: small},
		{name: "uint64", src: uint64(want), want: want},
		{name: "prefixed string", src: "u_d000000000000", want: want},
		{name: "empty string", src: "", want: 0},
		{name: "bytes big-endian", src: be, want: want},
		{name: "bytes prefixed string", src: []byte("u_d000000000000"), want: want},
		{name: "wrong prefix", src: "org_d000000000000", wantErr: ErrTypedPrefixMismatch},
		{name: "negative int64", src: int64(-1), wantErr: ErrInvalidScanValue},
		{name: "uint64 overflow", src: uint64(math.MaxInt64) + 1, wantErr: ErrInvalidScanValue},
		{name: "float64", src: float64(1), wantErr: ErrInvalidScanSource},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got UserID
			err := got.Scan(tc.src)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Scan() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Scan() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("Scan() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIDDatabaseSQLScanSources(t *testing.T) {
	want := MustParse("d000000000000")
	be := make([]byte, 8)
	binary.BigEndian.PutUint64(be, uint64(want))

	tests := []struct {
		name  string
		query any
	}{
		{name: "int64", query: int64(want)},
		{name: "string", query: "d000000000000"},
		{name: "bytes", query: be},
		{name: "nil", query: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, state := openSQLHarness(t)

			state.mu.Lock()
			state.queryVal = tc.query
			state.mu.Unlock()

			var scanned ID
			if err := db.QueryRowContext(t.Context(), "select").Scan(&scanned); err != nil {
				t.Fatalf("Scan() error = %v", err)
			}

			if tc.query == nil {
				if scanned != 0 {
					t.Fatalf("scanned ID = %v, want zero", scanned)
				}
				return
			}

			if scanned != want {
				t.Fatalf("scanned ID = %v, want %v", scanned, want)
			}
		})
	}
}

func TestTypedDatabaseSQLScanSources(t *testing.T) {
	want := UserID(MustParse("d000000000000"))

	tests := []struct {
		name  string
		query any
	}{
		{name: "int64", query: int64(want)},
		{name: "prefixed string", query: "u_d000000000000"},
		{name: "nil", query: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, state := openSQLHarness(t)

			state.mu.Lock()
			state.queryVal = tc.query
			state.mu.Unlock()

			var scanned UserID
			if err := db.QueryRowContext(t.Context(), "select").Scan(&scanned); err != nil {
				t.Fatalf("Scan() error = %v", err)
			}

			if tc.query == nil {
				if scanned != 0 {
					t.Fatalf("scanned UserID = %v, want zero", scanned)
				}
				return
			}

			if scanned != want {
				t.Fatalf("scanned UserID = %v, want %v", scanned, want)
			}
		})
	}
}
