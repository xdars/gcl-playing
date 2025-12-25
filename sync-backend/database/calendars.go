package database

import (
	"database/sql"
	"fmt"
	"time"

	shareddb "shared/database"
)

type Calendar struct {
	ID                 string
	UserID             string
	ConnectedAccountID *string
	Provider           string
	ProviderCalendarID string
	Name               string
	Color              *string
	IsPrimary          bool
	WebhookResourceID  *string
	WebhookChannelID   *string
	WebhookExpiry      *int64
	SyncToken          *string
	IsActive           bool
	CreatedAt          int64
	UpdatedAt          int64
}

func GetCalendarsByUserId(userId string) ([]Calendar, error) {
	db, err := shareddb.GetDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	rows, err := db.Query(`
		SELECT id, user_id, connected_account_id, provider, provider_calendar_id,
		       name, color, is_primary, webhook_resource_id, webhook_channel_id,
		       webhook_expiry, sync_token, is_active, created_at, updated_at
		FROM calendars
		WHERE user_id = ? AND is_active = 1
		ORDER BY is_primary DESC, name ASC
	`, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to query calendars: %w", err)
	}
	defer rows.Close()

	var calendars []Calendar
	for rows.Next() {
		var cal Calendar
		var connectedAccountID, color, webhookResourceID, webhookChannelID, syncToken sql.NullString
		var webhookExpiry sql.NullInt64
		var isPrimary, isActive int

		err := rows.Scan(
			&cal.ID, &cal.UserID, &connectedAccountID, &cal.Provider, &cal.ProviderCalendarID,
			&cal.Name, &color, &isPrimary, &webhookResourceID, &webhookChannelID,
			&webhookExpiry, &syncToken, &isActive, &cal.CreatedAt, &cal.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan calendar: %w", err)
		}

		cal.IsPrimary = isPrimary == 1
		cal.IsActive = isActive == 1

		if connectedAccountID.Valid {
			cal.ConnectedAccountID = &connectedAccountID.String
		}
		if color.Valid {
			cal.Color = &color.String
		}
		if webhookResourceID.Valid {
			cal.WebhookResourceID = &webhookResourceID.String
		}
		if webhookChannelID.Valid {
			cal.WebhookChannelID = &webhookChannelID.String
		}
		if webhookExpiry.Valid {
			cal.WebhookExpiry = &webhookExpiry.Int64
		}
		if syncToken.Valid {
			cal.SyncToken = &syncToken.String
		}

		calendars = append(calendars, cal)
	}

	return calendars, nil
}

func GetCalendarById(id string) (*Calendar, error) {
	db, err := shareddb.GetDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	var cal Calendar
	var connectedAccountID, color, webhookResourceID, webhookChannelID, syncToken sql.NullString
	var webhookExpiry sql.NullInt64
	var isPrimary, isActive int

	err = db.QueryRow(`
		SELECT id, user_id, connected_account_id, provider, provider_calendar_id,
		       name, color, is_primary, webhook_resource_id, webhook_channel_id,
		       webhook_expiry, sync_token, is_active, created_at, updated_at
		FROM calendars
		WHERE id = ?
	`, id).Scan(
		&cal.ID, &cal.UserID, &connectedAccountID, &cal.Provider, &cal.ProviderCalendarID,
		&cal.Name, &color, &isPrimary, &webhookResourceID, &webhookChannelID,
		&webhookExpiry, &syncToken, &isActive, &cal.CreatedAt, &cal.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get calendar: %w", err)
	}

	cal.IsPrimary = isPrimary == 1
	cal.IsActive = isActive == 1

	if connectedAccountID.Valid {
		cal.ConnectedAccountID = &connectedAccountID.String
	}
	if color.Valid {
		cal.Color = &color.String
	}
	if webhookResourceID.Valid {
		cal.WebhookResourceID = &webhookResourceID.String
	}
	if webhookChannelID.Valid {
		cal.WebhookChannelID = &webhookChannelID.String
	}
	if webhookExpiry.Valid {
		cal.WebhookExpiry = &webhookExpiry.Int64
	}
	if syncToken.Valid {
		cal.SyncToken = &syncToken.String
	}

	return &cal, nil
}

func CreateCalendar(cal Calendar) (string, error) {
	db, err := shareddb.GetDB()
	if err != nil {
		return "", fmt.Errorf("failed to get database: %w", err)
	}

	id := generateID()
	now := time.Now().Unix()

	isPrimary := 0
	if cal.IsPrimary {
		isPrimary = 1
	}

	_, err = db.Exec(`
		INSERT INTO calendars
		(id, user_id, connected_account_id, provider, provider_calendar_id, name, color, is_primary, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)
	`, id, cal.UserID, cal.ConnectedAccountID, cal.Provider, cal.ProviderCalendarID,
		cal.Name, cal.Color, isPrimary, now, now)

	if err != nil {
		return "", fmt.Errorf("failed to create calendar: %w", err)
	}

	return id, nil
}

func DeleteCalendar(id string) error {
	db, err := shareddb.GetDB()
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	_, err = db.Exec("DELETE FROM calendars WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete calendar: %w", err)
	}

	return nil
}

func UpdateCalendarWebhook(id, resourceId, channelId string, expiry int64) error {
	db, err := shareddb.GetDB()
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	now := time.Now().Unix()

	_, err = db.Exec(`
		UPDATE calendars
		SET webhook_resource_id = ?, webhook_channel_id = ?, webhook_expiry = ?, updated_at = ?
		WHERE id = ?
	`, resourceId, channelId, expiry, now, id)

	if err != nil {
		return fmt.Errorf("failed to update calendar webhook: %w", err)
	}

	return nil
}

func UpdateCalendarSyncToken(id, syncToken string) error {
	db, err := shareddb.GetDB()
	if err != nil {
		return fmt.Errorf("failed to get database: %w", err)
	}

	now := time.Now().Unix()

	_, err = db.Exec(`
		UPDATE calendars
		SET sync_token = ?, updated_at = ?
		WHERE id = ?
	`, syncToken, now, id)

	if err != nil {
		return fmt.Errorf("failed to update calendar sync token: %w", err)
	}

	return nil
}

func GetCalendarByProviderCalendarId(connectedAccountId, providerCalendarId string) (*Calendar, error) {
	db, err := shareddb.GetDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	var cal Calendar
	var connectedAccountID, color, webhookResourceID, webhookChannelID, syncToken sql.NullString
	var webhookExpiry sql.NullInt64
	var isPrimary, isActive int

	err = db.QueryRow(`
		SELECT id, user_id, connected_account_id, provider, provider_calendar_id,
		       name, color, is_primary, webhook_resource_id, webhook_channel_id,
		       webhook_expiry, sync_token, is_active, created_at, updated_at
		FROM calendars
		WHERE connected_account_id = ? AND provider_calendar_id = ?
	`, connectedAccountId, providerCalendarId).Scan(
		&cal.ID, &cal.UserID, &connectedAccountID, &cal.Provider, &cal.ProviderCalendarID,
		&cal.Name, &color, &isPrimary, &webhookResourceID, &webhookChannelID,
		&webhookExpiry, &syncToken, &isActive, &cal.CreatedAt, &cal.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get calendar by provider id: %w", err)
	}

	cal.IsPrimary = isPrimary == 1
	cal.IsActive = isActive == 1

	if connectedAccountID.Valid {
		cal.ConnectedAccountID = &connectedAccountID.String
	}
	if color.Valid {
		cal.Color = &color.String
	}
	if webhookResourceID.Valid {
		cal.WebhookResourceID = &webhookResourceID.String
	}
	if webhookChannelID.Valid {
		cal.WebhookChannelID = &webhookChannelID.String
	}
	if webhookExpiry.Valid {
		cal.WebhookExpiry = &webhookExpiry.Int64
	}
	if syncToken.Valid {
		cal.SyncToken = &syncToken.String
	}

	return &cal, nil
}
