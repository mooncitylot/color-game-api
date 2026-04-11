package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PushSubscription struct {
	ID           string    `json:"id" db:"id"`
	UserID       string    `json:"userId" db:"user_id"`
	Endpoint     string    `json:"endpoint" db:"endpoint"`
	P256dh       string    `json:"p256dh" db:"p256dh"`
	Auth         string    `json:"auth" db:"auth"`
	Expiration   *time.Time `json:"expiration,omitempty" db:"expiration"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

type PushNotificationRequest struct {
	Subscription PushSubscriptionPayload `json:"subscription"`
}

type PushSubscriptionPayload struct {
	Endpoint string                 `json:"endpoint"`
	Keys     PushSubscriptionKeys   `json:"keys"`
}

type PushSubscriptionKeys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

type PushNotificationMessage struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Icon  string `json:"icon,omitempty"`
	Data  any    `json:"data,omitempty"`
}

func NewPushSubscription(userID string, payload PushSubscriptionPayload) PushSubscription {
	now := time.Now()
	return PushSubscription{
		ID:        uuid.New().String(),
		UserID:    userID,
		Endpoint:  payload.Endpoint,
		P256dh:    payload.Keys.P256dh,
		Auth:      payload.Keys.Auth,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (p PushSubscription) Serialize() ([]byte, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return []byte{}, fmt.Errorf("error serializing push subscription: %v", err)
	}
	return data, nil
}
