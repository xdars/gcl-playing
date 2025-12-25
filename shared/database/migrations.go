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
	{
		Version: 2,
		Name:    "create_connected_accounts_table",
		Up: `
			CREATE TABLE IF NOT EXISTS connected_accounts (
				id TEXT PRIMARY KEY,
				user_id TEXT NOT NULL,
				provider TEXT NOT NULL DEFAULT 'google',
				provider_account_id TEXT NOT NULL,
				email TEXT NOT NULL,
				access_token TEXT NOT NULL,
				refresh_token TEXT,
				token_expiry INTEGER,
				created_at INTEGER NOT NULL,
				updated_at INTEGER NOT NULL,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
				UNIQUE(user_id, provider, provider_account_id)
			);
			CREATE INDEX IF NOT EXISTS idx_connected_accounts_user_id ON connected_accounts(user_id);
		`,
	},
	{
		Version: 3,
		Name:    "create_calendars_table",
		Up: `
			CREATE TABLE IF NOT EXISTS calendars (
				id TEXT PRIMARY KEY,
				user_id TEXT NOT NULL,
				connected_account_id TEXT,
				provider TEXT NOT NULL DEFAULT 'google',
				provider_calendar_id TEXT NOT NULL,
				name TEXT NOT NULL,
				color TEXT,
				is_primary INTEGER DEFAULT 0,
				webhook_resource_id TEXT,
				webhook_channel_id TEXT,
				webhook_expiry INTEGER,
				sync_token TEXT,
				is_active INTEGER DEFAULT 1,
				created_at INTEGER NOT NULL,
				updated_at INTEGER NOT NULL,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
				FOREIGN KEY (connected_account_id) REFERENCES connected_accounts(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_calendars_user_id ON calendars(user_id);
			CREATE INDEX IF NOT EXISTS idx_calendars_connected_account_id ON calendars(connected_account_id);
		`,
	},
	{
		Version: 4,
		Name:    "create_events_table",
		Up: `
			CREATE TABLE IF NOT EXISTS events (
				id TEXT PRIMARY KEY,
				calendar_id TEXT NOT NULL,
				provider_event_id TEXT NOT NULL,
				title TEXT,
				description TEXT,
				location TEXT,
				start_time INTEGER NOT NULL,
				end_time INTEGER NOT NULL,
				is_all_day INTEGER DEFAULT 0,
				status TEXT DEFAULT 'confirmed',
				recurrence TEXT,
				attendees TEXT,
				etag TEXT,
				raw_data TEXT,
				created_at INTEGER NOT NULL,
				updated_at INTEGER NOT NULL,
				FOREIGN KEY (calendar_id) REFERENCES calendars(id) ON DELETE CASCADE
			);
			CREATE INDEX IF NOT EXISTS idx_events_calendar_id ON events(calendar_id);
			CREATE INDEX IF NOT EXISTS idx_events_start_time ON events(start_time);
			CREATE INDEX IF NOT EXISTS idx_events_provider_event_id ON events(calendar_id, provider_event_id);
		`,
	},
}

func RunMigrations(db *sql.DB) error {
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
