package database

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
)

type Migration struct {
	Version int
	Name    string
	Up      string
}

var migrations = []Migration{
	{
		Version: 1,
		Name:    "create_users_table",
		Up: `
			CREATE TABLE IF NOT EXISTS users (
				id TEXT PRIMARY KEY,
				email TEXT UNIQUE NOT NULL,
				created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now')),
				token TEXT,
				refresh_token TEXT,
				is_outlook INTEGER DEFAULT 0
			);
			CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
		`,
	},
}

func RunMigrations(db *sql.DB) error {
	// Ensure migrations table exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	applied := make(map[int]bool)
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[version] = true
	}

	// Sort and apply pending migrations
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}

		log.Printf("Applying migration %d: %s", m.Version, m.Name)

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		if _, err := tx.Exec(m.Up); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %d: %w", m.Version, err)
		}

		if _, err := tx.Exec(
			"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
			m.Version, m.Name,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", m.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", m.Version, err)
		}
	}

	return nil
}
