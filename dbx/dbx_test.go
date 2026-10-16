package dbx

import (
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

// openDB returns an in-memory database with a table kv(k, v) holding a=1 and
// b=2. One connection, since every connection to :memory: is its own
// database.
func openDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	if _, err := db.ExecContext(t.Context(), `CREATE TABLE kv (k TEXT PRIMARY KEY, v INTEGER NOT NULL); INSERT INTO kv VALUES ('a', 1), ('b', 2)`); err != nil {
		t.Fatal(err)
	}

	return db
}

func scanInt(sc Scanner) (int, error) {
	var v int

	err := sc.Scan(&v)

	return v, err
}

func TestGet(t *testing.T) {
	db := openDB(t)

	v, err := Get(t.Context(), db, scanInt, `SELECT v FROM kv WHERE k = ?`, "b")
	if err != nil || v != 2 {
		t.Fatalf("Get = %d, %v; want 2", v, err)
	}

	if _, err := Get(t.Context(), db, scanInt, `SELECT v FROM kv WHERE k = ?`, "x"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Get of a missing row: got %v, want sql.ErrNoRows", err)
	}
}

func TestSelect(t *testing.T) {
	db := openDB(t)

	got, err := Select(t.Context(), db, scanInt, `SELECT v FROM kv ORDER BY k`)
	if err != nil || len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("Select = %v, %v; want [1 2]", got, err)
	}

	got, err = Select(t.Context(), db, scanInt, `SELECT v FROM kv WHERE k = 'x'`)
	if err != nil || len(got) != 0 {
		t.Fatalf("Select of no rows = %v, %v", got, err)
	}

	if _, err := Select(t.Context(), db, scanInt, `SELECT nope FROM kv`); err == nil {
		t.Fatal("Select of a bad query must fail")
	}
}

func TestEachStops(t *testing.T) {
	db := openDB(t)
	stop := errors.New("stop")
	calls := 0

	err := Each(t.Context(), db, func(Scanner) error {
		calls++

		return stop
	}, `SELECT v FROM kv`)
	if !errors.Is(err, stop) || calls != 1 {
		t.Fatalf("Each = %v after %d calls, want stop after 1", err, calls)
	}
}

func TestExecOne(t *testing.T) {
	db := openDB(t)
	notFound := errors.New("not found")

	if err := ExecOne(t.Context(), db, notFound, `DELETE FROM kv WHERE k = ?`, "a"); err != nil {
		t.Fatalf("ExecOne of an existing row: %v", err)
	}

	if err := ExecOne(t.Context(), db, notFound, `DELETE FROM kv WHERE k = ?`, "a"); !errors.Is(err, notFound) {
		t.Fatalf("ExecOne of a missing row: got %v, want notFound", err)
	}
}

func TestExecCount(t *testing.T) {
	db := openDB(t)

	n, err := ExecCount(t.Context(), db, `UPDATE kv SET v = v + 1`)
	if err != nil || n != 2 {
		t.Fatalf("ExecCount = %d, %v; want 2", n, err)
	}
}

func TestInTx(t *testing.T) {
	db := openDB(t)
	fail := errors.New("fail")

	err := InTx(t.Context(), db, func(q Querier) error {
		if _, err := q.ExecContext(t.Context(), `DELETE FROM kv`); err != nil {
			return err
		}

		return fail
	})
	if !errors.Is(err, fail) {
		t.Fatalf("InTx = %v, want fail", err)
	}

	if n, _ := Get(t.Context(), db, scanInt, `SELECT COUNT(*) FROM kv`); n != 2 {
		t.Fatalf("after a failed transaction %d rows remain, want 2 (rolled back)", n)
	}

	err = InTx(t.Context(), db, func(q Querier) error {
		_, err := q.ExecContext(t.Context(), `DELETE FROM kv WHERE k = 'a'`)

		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	if n, _ := Get(t.Context(), db, scanInt, `SELECT COUNT(*) FROM kv`); n != 1 {
		t.Fatalf("after a committed transaction %d rows remain, want 1", n)
	}
}

func TestInTxJoins(t *testing.T) {
	db := openDB(t)

	tx, err := db.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	var inner Querier

	if err := InTx(t.Context(), tx, func(q Querier) error {
		inner = q

		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if inner != tx {
		t.Fatal("InTx on a transaction must run fn in it")
	}

	// The joined transaction is still open: committing is the caller's.
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit after InTx joined: %v", err)
	}
}
