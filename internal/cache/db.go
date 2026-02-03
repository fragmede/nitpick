package cache

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database for HN item caching.
type DB struct {
	db *sql.DB
}

// Open creates or opens the SQLite cache database and runs migrations.
func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating database: %w", err)
	}
	return &DB{db: db}, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// Query executes a query and returns rows.
func (d *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.Query(query, args...)
}

// Exec executes a statement.
func (d *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return d.db.Exec(query, args...)
}

// QueryRow executes a query that returns at most one row.
func (d *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return d.db.QueryRow(query, args...)
}

func migrate(db *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS items (
			id INTEGER PRIMARY KEY,
			type TEXT NOT NULL,
			by_user TEXT,
			time_unix INTEGER,
			text TEXT,
			parent_id INTEGER,
			url TEXT,
			title TEXT,
			score INTEGER DEFAULT 0,
			descendants INTEGER DEFAULT 0,
			kids TEXT,
			dead INTEGER DEFAULT 0,
			deleted INTEGER DEFAULT 0,
			fetched_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_items_parent ON items(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_items_by_user ON items(by_user)`,

		`CREATE TABLE IF NOT EXISTS story_lists (
			list_type TEXT PRIMARY KEY,
			item_ids TEXT NOT NULL,
			fetched_at INTEGER NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			created INTEGER,
			karma INTEGER,
			about TEXT,
			fetched_at INTEGER NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS monitored_comments (
			item_id INTEGER PRIMARY KEY,
			parent_story_id INTEGER,
			known_kids TEXT NOT NULL DEFAULT '[]',
			last_checked INTEGER NOT NULL,
			depth INTEGER DEFAULT 0,
			created_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_monitored_last_checked ON monitored_comments(last_checked)`,

		`CREATE TABLE IF NOT EXISTS notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			item_id INTEGER NOT NULL UNIQUE,
			parent_id INTEGER NOT NULL,
			story_id INTEGER,
			by_user TEXT,
			text_preview TEXT,
			created_at INTEGER NOT NULL,
			read INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(read)`,

		`CREATE TABLE IF NOT EXISTS session (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("executing migration: %w\nSQL: %s", err, m)
		}
	}
	return nil
}
