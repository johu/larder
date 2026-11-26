// Package store holds the SQLite storage: opening the database, applying
// schema migrations and, later, the repositories.
package store

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// Open opens the SQLite database file at path, creating it if it does not
// exist, and verifies that a connection can be made.
//
// Every connection of the returned pool must have these settings, because
// SQLite applies them per connection, not per database file:
//   - foreign_keys = ON
//   - busy_timeout of at least one second, so a writer waits for a lock
//     instead of failing at once
//
// The database must use journal_mode = WAL (this setting is stored in the
// file). Use the driver's "_pragma" DSN parameters so that the settings are
// applied to every new connection, not only to the first one, and remember
// that a path may contain characters that are special in a URI.
//
// It returns an error if the file cannot be opened or created, for example
// because its directory does not exist. The caller must Close the result.
func Open(path string) (*sql.DB, error) {
	if path == "" || path == ":memory:" {
		return nil, fmt.Errorf("store: a database file path is required, got %q", path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("store: resolve database path: %w", err)
	}

	dsn := (&url.URL{
		Scheme: "file",
		Path:   absPath,
	}).String()

	params := url.Values{}
	params.Add("_pragma", "foreign_keys(ON)")
	params.Add("_pragma", "busy_timeout(1000)")
	params.Add("_pragma", "journal_mode(WAL)")

	dsn += "?" + params.Encode()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: open database: %w", err)
	}

	db.SetMaxOpenConns(4)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: ping database: %w", err)
	}

	return db, nil
}
