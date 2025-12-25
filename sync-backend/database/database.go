package database

import (
	"database/sql"
	"fmt"

	shareddb "shared/database"
)

type User struct {
	ID           string
	Email        string
	Token        string
	RefreshToken string
	CreatedAt    int64
}

func GetDB() (*sql.DB, error) {
	return shareddb.GetDB()
}

func GetUser(email string) (*User, error) {
	db, err := shareddb.GetDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	var user User
	err = db.QueryRow(
		"SELECT id, email, token, refresh_token, created_at FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Email, &user.Token, &user.RefreshToken, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user %s: %w", email, err)
	}

	return &user, nil
}

func GetUserById(id string) (*User, error) {
	db, err := shareddb.GetDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	var user User
	err = db.QueryRow(
		"SELECT id, email, token, refresh_token, created_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Email, &user.Token, &user.RefreshToken, &user.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by id %s: %w", id, err)
	}

	return &user, nil
}
