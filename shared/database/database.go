package database

import (
	"database/sql"
	"fmt"
	"os"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db   *sql.DB
	once sync.Once
	initErr error
)

// GetDB returns a singleton database connection
func GetDB() (*sql.DB, error) {
	once.Do(func() {
		dbPath := os.Getenv("DATABASE_PATH")
		if dbPath == "" {
			dbPath = "./data.db"
		}

		var err error
		db, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			initErr = fmt.Errorf("failed to open database: %w", err)
			return
		}

		// Configure connection pool for SQLite
		db.SetMaxOpenConns(1) // SQLite single-writer
		db.SetMaxIdleConns(1)

		if err := db.Ping(); err != nil {
			initErr = fmt.Errorf("failed to ping database: %w", err)
			return
		}

		// Run migrations
		if err := RunMigrations(db); err != nil {
			initErr = fmt.Errorf("failed to run migrations: %w", err)
			return
		}
	})

	if initErr != nil {
		return nil, initErr
	}
	return db, nil
}

// Close closes the database connection
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
