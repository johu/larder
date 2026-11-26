package store

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func appliedVersions(t *testing.T, db *sql.DB) []int {
	t.Helper()
	rows, err := db.Query("SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		t.Fatalf("read schema_migrations: %v", err)
	}
	defer rows.Close()

	var versions []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return versions
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n int
	err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?", name).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n == 1
}

func mig(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for name, sqlText := range files {
		fsys[name] = &fstest.MapFile{Data: []byte(sqlText)}
	}
	return fsys
}

func TestMigrateAppliesInOrder(t *testing.T) {
	db := openTest(t)
	fsys := mig(map[string]string{
		"0001_create.sql": "CREATE TABLE t (n INTEGER PRIMARY KEY, v TEXT);",
		"0002_first.sql":  "INSERT INTO t (v) VALUES ('first');",
		"0003_second.sql": "INSERT INTO t (v) VALUES ('second');",
	})

	if err := migrate(context.Background(), db, fsys); err != nil {
		t.Fatalf("migrate() error: %v", err)
	}

	if got, want := appliedVersions(t, db), []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("applied versions = %v; want %v", got, want)
	}
	rows, err := db.Query("SELECT v FROM t ORDER BY n")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		got = append(got, v)
	}
	if want := []string{"first", "second"}; !slices.Equal(got, want) {
		t.Errorf("rows = %v; want %v", got, want)
	}
}

func TestMigrateSortsNumerically(t *testing.T) {
	db := openTest(t)
	// A plain string sort would put 10 before 2.
	fsys := mig(map[string]string{
		"0010_c.sql": "INSERT INTO t (v) VALUES ('c');",
		"0002_b.sql": "INSERT INTO t (v) VALUES ('b');",
		"0001_a.sql": "CREATE TABLE t (n INTEGER PRIMARY KEY, v TEXT); INSERT INTO t (v) VALUES ('a');",
	})

	if err := migrate(context.Background(), db, fsys); err != nil {
		t.Fatalf("migrate() error: %v", err)
	}
	if got, want := appliedVersions(t, db), []int{1, 2, 10}; !slices.Equal(got, want) {
		t.Errorf("applied versions = %v; want %v", got, want)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	db := openTest(t)
	fsys := mig(map[string]string{
		"0001_create.sql": "CREATE TABLE t (v TEXT PRIMARY KEY);",
		"0002_insert.sql": "INSERT INTO t VALUES ('once');",
	})

	for i := 0; i < 2; i++ {
		if err := migrate(context.Background(), db, fsys); err != nil {
			t.Fatalf("run %d: migrate() error: %v", i+1, err)
		}
	}

	if got, want := appliedVersions(t, db), []int{1, 2}; !slices.Equal(got, want) {
		t.Errorf("applied versions = %v; want %v", got, want)
	}
}

func TestMigrateAppliesOnlyNewMigrations(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	first := mig(map[string]string{
		"0001_create.sql": "CREATE TABLE t (v TEXT PRIMARY KEY); INSERT INTO t VALUES ('one');",
	})
	both := mig(map[string]string{
		// Running 0001 again would fail on the primary key.
		"0001_create.sql": "CREATE TABLE t (v TEXT PRIMARY KEY); INSERT INTO t VALUES ('one');",
		"0002_more.sql":   "INSERT INTO t VALUES ('two');",
	})

	if err := migrate(ctx, db, first); err != nil {
		t.Fatalf("first migrate() error: %v", err)
	}
	if err := migrate(ctx, db, both); err != nil {
		t.Fatalf("second migrate() error: %v", err)
	}
	if got, want := appliedVersions(t, db), []int{1, 2}; !slices.Equal(got, want) {
		t.Errorf("applied versions = %v; want %v", got, want)
	}
}

func TestMigrateRunsMultipleStatements(t *testing.T) {
	db := openTest(t)
	fsys := mig(map[string]string{
		"0001_two_tables.sql": "CREATE TABLE a (x INTEGER);\nCREATE TABLE b (y INTEGER);\nINSERT INTO a VALUES (1);\n",
	})

	if err := migrate(context.Background(), db, fsys); err != nil {
		t.Fatalf("migrate() error: %v", err)
	}
	for _, name := range []string{"a", "b"} {
		if !tableExists(t, db, name) {
			t.Errorf("table %q does not exist", name)
		}
	}
}

func TestMigrateRollsBackAFailedMigration(t *testing.T) {
	db := openTest(t)
	fsys := mig(map[string]string{
		"0001_ok.sql": "CREATE TABLE a (x INTEGER);",
		// The first statement succeeds, the second fails: nothing of
		// this file may stay.
		"0002_broken.sql": "CREATE TABLE b (y INTEGER); INSERT INTO does_not_exist VALUES (1);",
	})

	err := migrate(context.Background(), db, fsys)
	if err == nil {
		t.Fatal("migrate() returned no error for a broken migration")
	}
	if !strings.Contains(err.Error(), "0002_broken.sql") {
		t.Errorf("error %q does not name the failing file", err)
	}
	if tableExists(t, db, "b") {
		t.Error("table b exists, but the failed migration must be rolled back")
	}
	if got, want := appliedVersions(t, db), []int{1}; !slices.Equal(got, want) {
		t.Errorf("applied versions = %v; want %v", got, want)
	}
}

func TestMigrateStopsAtTheFirstFailure(t *testing.T) {
	db := openTest(t)
	fsys := mig(map[string]string{
		"0001_ok.sql":     "CREATE TABLE a (x INTEGER);",
		"0002_broken.sql": "THIS IS NOT SQL;",
		"0003_later.sql":  "CREATE TABLE c (z INTEGER);",
	})

	if err := migrate(context.Background(), db, fsys); err == nil {
		t.Fatal("migrate() returned no error")
	}
	if tableExists(t, db, "c") {
		t.Error("migration 0003 was applied after 0002 failed")
	}
}

func TestMigrateRetriesAfterAFix(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	broken := mig(map[string]string{"0001_t.sql": "CREATE TABLE t (x INTEGER); BROKEN;"})
	fixed := mig(map[string]string{"0001_t.sql": "CREATE TABLE t (x INTEGER);"})

	if err := migrate(ctx, db, broken); err == nil {
		t.Fatal("migrate() returned no error for the broken migration")
	}
	if err := migrate(ctx, db, fixed); err != nil {
		t.Fatalf("migrate() after the fix: %v", err)
	}
	if got, want := appliedVersions(t, db), []int{1}; !slices.Equal(got, want) {
		t.Errorf("applied versions = %v; want %v", got, want)
	}
}

func TestMigrateRejectsBadFileNames(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
	}{
		{"no version", map[string]string{"init.sql": "SELECT 1;"}},
		{"version is not a number", map[string]string{"abcd_init.sql": "SELECT 1;"}},
		{"no description", map[string]string{"0001.sql": "SELECT 1;"}},
		{"duplicate version", map[string]string{
			"0001_a.sql": "CREATE TABLE a (x INTEGER);",
			"0001_b.sql": "CREATE TABLE b (x INTEGER);",
		}},
		{"duplicate version with different padding", map[string]string{
			"0001_a.sql": "CREATE TABLE a (x INTEGER);",
			"1_b.sql":    "CREATE TABLE b (x INTEGER);",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openTest(t)
			// A valid file in front of the bad one must not be applied
			// either: names are checked before anything runs.
			files := map[string]string{"0000_first.sql": "CREATE TABLE first (x INTEGER);"}
			for k, v := range tt.files {
				files[k] = v
			}
			if err := migrate(context.Background(), db, mig(files)); err == nil {
				t.Fatal("migrate() returned no error")
			}
			if tableExists(t, db, "first") {
				t.Error("a migration was applied although the file names are invalid")
			}
		})
	}
}

func TestMigrateIgnoresOtherFiles(t *testing.T) {
	db := openTest(t)
	fsys := mig(map[string]string{
		"0001_create.sql": "CREATE TABLE t (x INTEGER);",
		"README.md":       "not a migration",
		"notes.txt":       "neither",
	})

	if err := migrate(context.Background(), db, fsys); err != nil {
		t.Fatalf("migrate() error: %v", err)
	}
	if got, want := appliedVersions(t, db), []int{1}; !slices.Equal(got, want) {
		t.Errorf("applied versions = %v; want %v", got, want)
	}
}

func TestMigrateWithoutMigrations(t *testing.T) {
	db := openTest(t)
	if err := migrate(context.Background(), db, fstest.MapFS{}); err != nil {
		t.Fatalf("migrate() error: %v", err)
	}
	if !tableExists(t, db, "schema_migrations") {
		t.Error("schema_migrations was not created")
	}
}

func TestMigrateRejectsADatabaseNewerThanTheBinary(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()
	newer := mig(map[string]string{
		"0001_a.sql": "CREATE TABLE a (x INTEGER);",
		"0002_b.sql": "CREATE TABLE b (x INTEGER);",
	})
	older := mig(map[string]string{
		"0001_a.sql": "CREATE TABLE a (x INTEGER);",
		"0003_c.sql": "CREATE TABLE c (x INTEGER);",
	})

	if err := migrate(ctx, db, newer); err != nil {
		t.Fatalf("migrate() error: %v", err)
	}
	// Version 2 is recorded, but this binary does not know it.
	if err := migrate(ctx, db, older); err == nil {
		t.Fatal("migrate() returned no error for a database with an unknown version")
	}
	if tableExists(t, db, "c") {
		t.Error("migration 0003 was applied although the database has an unknown version")
	}
}

func TestMigrateRecordsNameAndTime(t *testing.T) {
	db := openTest(t)
	fsys := mig(map[string]string{"0001_create_things.sql": "CREATE TABLE t (x INTEGER);"})

	if err := migrate(context.Background(), db, fsys); err != nil {
		t.Fatalf("migrate() error: %v", err)
	}

	var name string
	var appliedAt sql.NullString
	if err := db.QueryRow("SELECT name, applied_at FROM schema_migrations WHERE version = 1").Scan(&name, &appliedAt); err != nil {
		t.Fatal(err)
	}
	if name != "0001_create_things.sql" {
		t.Errorf("name = %q; want the file name", name)
	}
	if !appliedAt.Valid || appliedAt.String == "" {
		t.Error("applied_at is empty")
	}
}

func TestMigrateStopsWhenTheContextIsCancelled(t *testing.T) {
	db := openTest(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fsys := mig(map[string]string{"0001_a.sql": "CREATE TABLE a (x INTEGER);"})
	if err := migrate(ctx, db, fsys); err == nil {
		t.Fatal("migrate() returned no error for a cancelled context")
	}
	if tableExists(t, db, "a") {
		t.Error("a migration was applied with a cancelled context")
	}
}

func TestMigrateEmbedded(t *testing.T) {
	db := openTest(t)
	ctx := context.Background()

	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("second Migrate() error: %v", err)
	}

	got := appliedVersions(t, db)
	if len(got) < 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("applied versions = %v; want them to start with 1, 2", got)
	}
}
