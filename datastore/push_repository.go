package datastore

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/color-game/api/models"
)

type PushNotificationDatabase interface {
	CreateSubscription(sub models.PushSubscription) error
	GetSubscriptionByEndpoint(endpoint string) (*models.PushSubscription, error)
	GetUserSubscriptions(userID string) ([]models.PushSubscription, error)
	GetAllSubscriptions() ([]models.PushSubscription, error)
	DeleteSubscription(endpoint string) error
	DeleteExpiredSubscriptions() (int64, error)
}

type pushNotificationDatabase struct {
	db *sql.DB
}

func NewPushNotificationDatabase(db *sql.DB) (PushNotificationDatabase, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	return &pushNotificationDatabase{db: db}, nil
}

func (p *pushNotificationDatabase) CreateSubscription(sub models.PushSubscription) error {
	query := `
		INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth, expiration, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (endpoint) DO UPDATE SET
			p256dh = EXCLUDED.p256dh,
			auth = EXCLUDED.auth,
			expiration = EXCLUDED.expiration,
			updated_at = EXCLUDED.updated_at
	`
	_, err := p.db.Exec(query,
		sub.ID,
		sub.UserID,
		sub.Endpoint,
		sub.P256dh,
		sub.Auth,
		sub.Expiration,
		sub.CreatedAt,
		sub.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create push subscription: %v", err)
	}
	return nil
}

func (p *pushNotificationDatabase) GetSubscriptionByEndpoint(endpoint string) (*models.PushSubscription, error) {
	query := `
		SELECT id, user_id, endpoint, p256dh, auth, expiration, created_at, updated_at
		FROM push_subscriptions
		WHERE endpoint = $1
	`
	sub := &models.PushSubscription{}
	err := p.db.QueryRow(query, endpoint).Scan(
		&sub.ID,
		&sub.UserID,
		&sub.Endpoint,
		&sub.P256dh,
		&sub.Auth,
		&sub.Expiration,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get push subscription: %v", err)
	}
	return sub, nil
}

func (p *pushNotificationDatabase) GetUserSubscriptions(userID string) ([]models.PushSubscription, error) {
	query := `
		SELECT id, user_id, endpoint, p256dh, auth, expiration, created_at, updated_at
		FROM push_subscriptions
		WHERE user_id = $1 AND (expiration IS NULL OR expiration > $2)
	`
	rows, err := p.db.Query(query, userID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get user subscriptions: %v", err)
	}
	defer rows.Close()

	var subscriptions []models.PushSubscription
	for rows.Next() {
		var sub models.PushSubscription
		err := rows.Scan(
			&sub.ID,
			&sub.UserID,
			&sub.Endpoint,
			&sub.P256dh,
			&sub.Auth,
			&sub.Expiration,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan push subscription: %v", err)
		}
		subscriptions = append(subscriptions, sub)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating push subscriptions: %v", err)
	}

	return subscriptions, nil
}

func (p *pushNotificationDatabase) GetAllSubscriptions() ([]models.PushSubscription, error) {
	query := `
		SELECT id, user_id, endpoint, p256dh, auth, expiration, created_at, updated_at
		FROM push_subscriptions
		WHERE expiration IS NULL OR expiration > $1
	`
	rows, err := p.db.Query(query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriptions: %v", err)
	}
	defer rows.Close()

	var subscriptions []models.PushSubscription
	for rows.Next() {
		var sub models.PushSubscription
		err := rows.Scan(
			&sub.ID,
			&sub.UserID,
			&sub.Endpoint,
			&sub.P256dh,
			&sub.Auth,
			&sub.Expiration,
			&sub.CreatedAt,
			&sub.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan push subscription: %v", err)
		}
		subscriptions = append(subscriptions, sub)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating push subscriptions: %v", err)
	}

	return subscriptions, nil
}

func (p *pushNotificationDatabase) DeleteSubscription(endpoint string) error {
	query := `DELETE FROM push_subscriptions WHERE endpoint = $1`
	result, err := p.db.Exec(query, endpoint)
	if err != nil {
		return fmt.Errorf("failed to delete push subscription: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("subscription not found")
	}
	return nil
}

func (p *pushNotificationDatabase) DeleteExpiredSubscriptions() (int64, error) {
	query := `DELETE FROM push_subscriptions WHERE expiration IS NOT NULL AND expiration < $1`
	result, err := p.db.Exec(query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired subscriptions: %v", err)
	}
	return result.RowsAffected()
}
