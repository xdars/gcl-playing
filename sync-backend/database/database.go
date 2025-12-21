package database

import (
	"fmt"

	shareddb "shared/database"
)

type User struct {
	Token  string
	RToken string
}

func GetUser(email string) (*User, error) {
	db, err := shareddb.GetDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	var token, rtoken string
	err = db.QueryRow(
		"SELECT token, refresh_token FROM users WHERE email = ?",
		email,
	).Scan(&token, &rtoken)

	if err != nil {
		return nil, fmt.Errorf("failed to get user %s: %w", email, err)
	}

	return &User{Token: token, RToken: rtoken}, nil
}
