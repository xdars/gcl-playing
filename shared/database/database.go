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

func GetDB() (*sql.DB, error) {
	once.Do(func() {
		dbPath := os.Getenv("DATABASE_PATH")
		if dbPath == "" {
			dbPath = "../data.db"
		}

		var err error
		db, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			initErr = fmt.Errorf("failed to open database: %w", err)
			return
		}

		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		if err := db.Ping(); err != nil {
			initErr = fmt.Errorf("failed to ping database: %w", err)
			return
		}

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

func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
