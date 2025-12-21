package database

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestMigrations(t *testing.T) {
	// Use temp database
	tmpDir := t.TempDir()
	tmpDB := filepath.Join(tmpDir, "test.db")
	os.Setenv("DATABASE_PATH", tmpDB)
	defer os.Unsetenv("DATABASE_PATH")

	// Reset singleton for testing
	db = nil
	once = sync.Once{}
	initErr = nil

	conn, err := GetDB()
	if err != nil {
		t.Fatalf("GetDB failed: %v", err)
	}
	defer Close()

	if conn == nil {
		t.Fatal("Expected non-nil database connection")
	}

	// Verify users table exists by inserting a row
	_, err = conn.Exec(
		"INSERT INTO users (id, email) VALUES (?, ?)",
		"test-id", "test@example.com",
	)
	if err != nil {
		t.Fatalf("Failed to insert into users: %v", err)
	}

	// Verify we can query the row
	var email string
	err = conn.QueryRow("SELECT email FROM users WHERE id = ?", "test-id").Scan(&email)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}

	if email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", email)
	}
}

func TestMigrationsIdempotent(t *testing.T) {
	// Use temp database
	tmpDir := t.TempDir()
	tmpDB := filepath.Join(tmpDir, "test.db")
	os.Setenv("DATABASE_PATH", tmpDB)
	defer os.Unsetenv("DATABASE_PATH")

	// Reset singleton for testing
	db = nil
	once = sync.Once{}
	initErr = nil

	conn, err := GetDB()
	if err != nil {
		t.Fatalf("First GetDB failed: %v", err)
	}

	// Running migrations again should not error
	err = RunMigrations(conn)
	if err != nil {
		t.Fatalf("Running migrations twice failed: %v", err)
	}

	Close()
}
