package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// openTest opens a fresh database file in a temporary directory. A file is
// used on purpose: every connection of an in-memory database would get its
// own empty database.
func openTest(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	if db == nil {
		t.Fatal("Open() returned a nil database")
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func pragmaInt(t *testing.T, conn *sql.Conn, name string) int {
	t.Helper()
	var v int
	if err := conn.QueryRowContext(context.Background(), "PRAGMA "+name).Scan(&v); err != nil {
		t.Fatalf("PRAGMA %s: %v", name, err)
	}
	return v
}

func TestOpenRejectsInMemory(t *testing.T) {
	t.Chdir(t.TempDir()) // keep a stray file out of the repo

	for _, path := range []string{":memory:", ""} {
		t.Run(path, func(t *testing.T) {
			db, err := Open(path)
			if err == nil {
				db.Close()
				t.Fatalf("Open(%q) returned no error", path)
			}
			if _, err := os.Stat(":memory:"); err == nil {
				t.Error("a file named :memory: was created")
			}
		})
	}
}

func TestOpenCreatesTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "larder.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(path); err != nil {
		t.Errorf("database file was not created: %v", err)
	}
}

func TestOpenFailsForMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "larder.db")
	db, err := Open(path)
	if err == nil {
		db.Close()
		t.Fatal("Open() returned no error for a path in a missing directory")
	}
}

func TestOpenUsesWAL(t *testing.T) {
	db := openTest(t)

	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q; want %q", mode, "wal")
	}
}

// TestOpenSettingsApplyToEveryConnection holds several connections at the
// same time, so the pool has to open new ones, and checks all of them.
func TestOpenSettingsApplyToEveryConnection(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()

	const n = 3
	conns := make([]*sql.Conn, 0, n)
	for i := 0; i < n; i++ {
		conn, err := db.Conn(ctx)
		if err != nil {
			t.Fatalf("Conn() %d: %v", i, err)
		}
		defer conn.Close()
		conns = append(conns, conn)
	}

	for i, conn := range conns {
		if got := pragmaInt(t, conn, "foreign_keys"); got != 1 {
			t.Errorf("connection %d: foreign_keys = %d; want 1", i, got)
		}
		if got := pragmaInt(t, conn, "busy_timeout"); got < 1000 {
			t.Errorf("connection %d: busy_timeout = %d ms; want at least 1000", i, got)
		}
	}
}

func TestOpenAllowsReopeningTheSameFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "larder.db")

	first, err := Open(path)
	if err != nil {
		t.Fatalf("first Open() error: %v", err)
	}
	if _, err := first.Exec("CREATE TABLE t (x INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if _, err := first.Exec("INSERT INTO t VALUES (42)"); err != nil {
		t.Fatal(err)
	}
	first.Close()

	second, err := Open(path)
	if err != nil {
		t.Fatalf("second Open() error: %v", err)
	}
	defer second.Close()

	var x int
	if err := second.QueryRow("SELECT x FROM t").Scan(&x); err != nil || x != 42 {
		t.Errorf("SELECT x = %d, %v; want 42 after reopening", x, err)
	}
}

// TestOpenPathWithSpecialCharacters makes sure that a path is not
// misinterpreted as a URI.
func TestOpenPathWithSpecialCharacters(t *testing.T) {
	for _, name := range []string{"with space.db", "question?mark.db", "hash#tag.db", "percent%20.db", "amp&ersand.db"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			db, err := Open(path)
			if err != nil {
				t.Fatalf("Open(%q) error: %v", path, err)
			}
			defer db.Close()

			if _, err := os.Stat(path); err != nil {
				t.Errorf("the file %q was not created: %v", path, err)
			}
		})
	}
}
