package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite connection with migration support.
type DB struct {
	*sql.DB
}

// Open opens (or creates) the SQLite database at path, runs pragmas and migrations.
func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sql open: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("sql ping: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &DB{db}, nil
}

// Close shuts down the database connection cleanly.
func (db *DB) Close() error {
	return db.DB.Close()
}

// Checkpoint triggers a WAL checkpoint to control WAL file growth.
func (db *DB) Checkpoint() error {
	_, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")

	return err
}
