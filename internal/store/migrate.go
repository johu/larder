package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var migrationPattern = regexp.MustCompile(`^([0-9]+)_(.+)\.sql$`)

type migration struct {
	version int
	name    string
	sql     string
}

// Migrate applies all migrations embedded in the binary (the migrations
// directory) that the database has not seen yet.
func Migrate(ctx context.Context, db *sql.DB) error {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("open migrations directory: %w", err)
	}

	return migrate(ctx, db, sub)
}

// migrate applies the migrations found in the root of fsys.
//
// Rules (see migrate_test.go):
//   - a migration is a file named NNNN_description.sql; NNNN is its
//     version, a number. Files without the .sql extension are ignored; a
//     .sql file whose name does not match the pattern is an error
//   - versions must be unique; files are applied in ascending numeric order
//   - applied versions are recorded in the table schema_migrations
//     (version INTEGER PRIMARY KEY, name TEXT, applied_at TEXT)
//   - a migration may contain several statements
//   - every migration runs in its own transaction together with the
//     insert into schema_migrations; if it fails, it is rolled back, the
//     later migrations are not applied, and the error names the file
//   - migrations that are already recorded are skipped, so running it
//     again is a no-op
//   - if the database has a recorded version that fsys does not contain,
//     the database is newer than the binary: return an error and do not
//     apply anything
//   - all checks on the file names happen before anything is applied
func migrate(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	migrations, err := loadMigrations(fsys)
	if err != nil {
		return err
	}

	if err := ensureSchemaTable(ctx, db); err != nil {
		return err
	}

	applied, err := loadApplied(ctx, db)
	if err != nil {
		return err
	}

	if err := checkNotNewer(applied, migrations); err != nil {
		return err
	}

	for _, m := range migrations {
		if _, exists := applied[m.version]; exists {
			continue
		}

		if err := applyMigration(ctx, db, m); err != nil {
			return fmt.Errorf("apply migration %q: %w", m.name, err)
		}
	}

	return nil
}

// loadMigrations reads, validates and sorts the migrations in the root of
// fsys. It does not touch the database; the rules are described at migrate.
func loadMigrations(fsys fs.FS) ([]migration, error) {
	// discover and validate migrations
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	var migrations []migration
	versions := make(map[int]struct{})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		matches := migrationPattern.FindStringSubmatch(name)
		if matches == nil {
			return nil, fmt.Errorf("invalid migration filename: %s", name)
		}

		version, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, fmt.Errorf("invalid migration version in %s: %w", name, err)
		}

		if _, exists := versions[version]; exists {
			return nil, fmt.Errorf("duplicate migration version: %d and %s", version, name)
		}

		versions[version] = struct{}{}

		content, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", name, err)
		}

		migrations = append(migrations, migration{
			version: version,
			name:    name,
			sql:     string(content),
		})
	}

	// sort migrations by numeric version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	return migrations, nil
}

// ensureSchemaTable creates the table that records the applied migrations if
// it does not exist yet.
func ensureSchemaTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	return nil
}

// loadApplied returns the versions recorded in schema_migrations.
func loadApplied(ctx context.Context, db *sql.DB) (map[int]struct{}, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]struct{})
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return applied, fmt.Errorf("scan schema_migrations: %w", err)
		}
		applied[version] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}

	return applied, nil
}

// checkNotNewer returns an error if the database has recorded a version that
// is not in migrations, which means it was migrated by a newer binary.
func checkNotNewer(applied map[int]struct{}, migrations []migration) error {
	known := make(map[int]struct{}, len(migrations))
	for _, m := range migrations {
		known[m.version] = struct{}{}
	}

	for version := range applied {
		if _, ok := known[version]; !ok {
			return fmt.Errorf("database contains migration version %d, which is missing from this binary", version)
		}
	}

	return nil
}

// applyMigration runs m and records it in schema_migrations in one
// transaction, so a failure leaves no partial change.
func applyMigration(ctx context.Context, db *sql.DB, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return fmt.Errorf("execute SQL: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO schema_migrations (version, name, applied_at)
		VALUES (?, ?, ?)
	`, m.version, m.name, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
