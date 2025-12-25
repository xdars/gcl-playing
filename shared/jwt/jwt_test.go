package jwt

import (
	"os"
	"sync"
	"testing"
)

func TestGenerateAndParseJWT(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing-purposes")
	defer os.Unsetenv("JWT_SECRET")

	jwtSecret = nil
	once = sync.Once{}

	email := "test@example.com"

	token, err := GenerateJWT(email)
	if err != nil {
		t.Fatalf("GenerateJWT failed: %v", err)
	}

	if token == "" {
		t.Fatal("Expected non-empty token")
	}

	parsed, err := ParseJWT(token)
	if err != nil {
		t.Fatalf("ParseJWT failed: %v", err)
	}

	if parsed != email {
		t.Errorf("Expected email %s, got %s", email, parsed)
	}
}

func TestParseInvalidJWT(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing-purposes")
	defer os.Unsetenv("JWT_SECRET")

	jwtSecret = nil
	once = sync.Once{}

	_, err := ParseJWT("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestParseEmptyJWT(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-secret-key-for-testing-purposes")
	defer os.Unsetenv("JWT_SECRET")

	jwtSecret = nil
	once = sync.Once{}

	_, err := ParseJWT("")
	if err == nil {
		t.Error("Expected error for empty token")
	}
}
