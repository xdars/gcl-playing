package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	shareddb "shared/database"
)

type ConnectedAccount struct {
	ID                string
	UserID            string
	Provider          string
	ProviderAccountID string
	Email             string
	AccessToken       string
	RefreshToken      string
	TokenExpiry       *int64
	CreatedAt         int64
	UpdatedAt         int64
}

func generateID() string {
	bytes := make([]byte, 11)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func GetConnectedAccountsByUserId(userId string) ([]ConnectedAccount, error) {
	db, err := shareddb.GetDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	rows, err := db.Query(`
		SELECT id, user_id, provider, provider_account_id, email,
		       access_token, refresh_token, token_expiry, created_at, updated_at
		FROM connected_accounts
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to query connected accounts: %w", err)
	}
	defer rows.Close()

	var accounts []ConnectedAccount
	for rows.Next() {
		var acc ConnectedAccount
		var tokenExpiry sql.NullInt64
		var refreshToken sql.NullString

		err := rows.Scan(
			&acc.ID, &acc.UserID, &acc.Provider, &acc.ProviderAccountID,
			&acc.Email, &acc.AccessToken, &refreshToken, &tokenExpiry,
			&acc.CreatedAt, &acc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan connected account: %w", err)
		}

		if tokenExpiry.Valid {
			acc.TokenExpiry = &tokenExpiry.Int64
		}
		if refreshToken.Valid {
			acc.RefreshToken = refreshToken.String
		}

		accounts = append(accounts, acc)
	}

	return accounts, nil
}

func GetConnectedAccountById(id string) (*ConnectedAccount, error) {
	db, err := shareddb.GetDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	var acc ConnectedAccount
	var tokenExpiry sql.NullInt64
	var refreshToken sql.NullString

	err = db.QueryRow(`
		SELECT id, user_id, provider, provider_account_id, email,
		       access_token, refresh_token, token_expiry, created_at, updated_at
		FROM connected_accounts
		WHERE id = ?
	`, id).Scan(
		&acc.ID, &acc.UserID, &acc.Provider, &acc.ProviderAccountID,
		&acc.Email, &acc.AccessToken, &refreshToken, &tokenExpiry,
		&acc.CreatedAt, &acc.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get connected account: %w", err)
	}

	if tokenExpiry.Valid {
		acc.TokenExpiry = &tokenExpiry.Int64
	}
	if refreshToken.Valid {
		acc.RefreshToken = refreshToken.String
	}

	return &acc, nil
}

func GetConnectedAccountByProviderAccountId(userId, provider, providerAccountId string) (*ConnectedAccount, error) {
	db, err := shareddb.GetDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	var acc ConnectedAccount
	var tokenExpiry sql.NullInt64
	var refreshToken sql.NullString

	err = db.QueryRow(`
		SELECT id, user_id, provider, provider_account_id, email,
		       access_token, refresh_token, token_expiry, created_at, updated_at
		FROM connected_accounts
		WHERE user_id = ? AND provider = ? AND provider_account_id = ?
	`, userId, provider, providerAccountId).Scan(
		&acc.ID, &acc.UserID, &acc.Provider, &acc.ProviderAccountID,
		&acc.Email, &acc.AccessToken, &refreshToken, &tokenExpiry,
		&acc.CreatedAt, &acc.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get connected account by provider: %w", err)
	}

	if tokenExpiry.Valid {
		acc.TokenExpiry = &tokenExpiry.Int64
	}
	if refreshToken.Valid {
		acc.RefreshToken = refreshToken.String
	}

	return &acc, nil
}

func CreateConnectedAccount(acc ConnectedAccount) (string, error) {
	db, err := shareddb.GetDB()
	if err != nil {
		return "", fmt.Errorf("failed to get database: %w", err)
	}

	id := generateID()
	now := time.Now().Unix()

	var tokenExpiry interface{} = nil
	if acc.TokenExpiry != nil {
		tokenExpiry = *acc.TokenExpiry
	}

	var refreshToken interface{} = nil
	if acc.RefreshToken != "" {
		refreshToken = acc.RefreshToken
	}

	_, err = db.Exec(`
		INSERT INTO connected_accounts
		(id, user_id, provider, provider_account_id, email, access_token, refresh_token, token_expiry, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, acc.UserID, acc.Provider, acc.ProviderAccountID, acc.Email,
		acc.AccessToken, refreshToken, tokenExpiry, now, now)

	if err != nil {
		return "", fmt.Errorf("failed to create connected account: %w", err)
	}

	return id, nil
}

func UpdateConnectedAccountTokens(id, accessToken, refreshToken string, tokenExpiry *int64) error {
	db, err := shareddb.GetDB()
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	now := time.Now().Unix()

	var expiryVal interface{} = nil
	if tokenExpiry != nil {
		expiryVal = *tokenExpiry
	}

	var refreshVal interface{} = nil
	if refreshToken != "" {
		refreshVal = refreshToken
	}

	_, err = db.Exec(`
		UPDATE connected_accounts
		SET access_token = ?, refresh_token = ?, token_expiry = ?, updated_at = ?
		WHERE id = ?
	`, accessToken, refreshVal, expiryVal, now, id)

	if err != nil {
		return fmt.Errorf("failed to update connected account tokens: %w", err)
	}

	return nil
}

func DeleteConnectedAccount(id string) error {
	db, err := shareddb.GetDB()
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	_, err = db.Exec("DELETE FROM connected_accounts WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete connected account: %w", err)
	}

	return nil
}
