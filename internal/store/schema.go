package store

import (
	"database/sql"
	"fmt"
)

const schemaVersion = 1

var migrations = map[int]string{
	1: `
	CREATE TABLE IF NOT EXISTS buffers (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		label        TEXT    NOT NULL DEFAULT '',
		content      TEXT    NOT NULL DEFAULT '',
		line_count   INTEGER NOT NULL DEFAULT 0,
		byte_count   INTEGER NOT NULL DEFAULT 0,
		tags         TEXT    NOT NULL DEFAULT '',
		created_at   TEXT    NOT NULL,
		updated_at   TEXT    NOT NULL,
		trash_status INTEGER NOT NULL DEFAULT 0,
		trashed_at   TEXT,
		expires_at   TEXT
	);
	`,
}

// migrate runs any pending database migrations.
func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY)`); err != nil {
		return fmt.Errorf("schema_version table: %w", err)
	}

	var currentVersion int
	err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`).Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	for v := currentVersion + 1; v <= schemaVersion; v++ {
		if err := runMigration(db, v); err != nil {
			return err
		}
	}
	return nil
}

func runMigration(db *sql.DB, version int) error {
	ddl, ok := migrations[version]
	if !ok {
		return fmt.Errorf("missing migration %d", version)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("migration %d begin tx: %w", version, err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(ddl); err != nil {
		return fmt.Errorf("migration %d exec: %w", version, err)
	}
	if _, err := tx.Exec(`INSERT INTO schema_version (version) VALUES (?)`, version); err != nil {
		return fmt.Errorf("migration %d record: %w", version, err)
	}

	return tx.Commit()
}
