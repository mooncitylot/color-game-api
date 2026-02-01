package models

import "time"

const (
	FriendshipStatusPending  = "pending"
	FriendshipStatusAccepted = "accepted"
	FriendshipStatusDeclined = "declined"
)

// Friendship represents a raw friendship record
type Friendship struct {
	FriendshipID int        `json:"friendshipId" db:"friendship_id"`
	RequesterID  string     `json:"requesterId" db:"requester_id"`
	AddresseeID  string     `json:"addresseeId" db:"addressee_id"`
	Status       string     `json:"status" db:"status"`
	CreatedAt    time.Time  `json:"createdAt" db:"created_at"`
	RespondedAt  *time.Time `json:"respondedAt,omitempty" db:"responded_at"`
}

// FriendSummary represents an accepted friendship with the other user's summary
type FriendSummary struct {
	FriendshipID int         `json:"friendshipId"`
	Friend       UserSummary `json:"friend"`
	Status       string      `json:"status"`
	CreatedAt    time.Time   `json:"createdAt"`
	RespondedAt  *time.Time  `json:"respondedAt,omitempty"`
}

// FriendRequestSummary represents a pending request in either direction
type FriendRequestSummary struct {
	FriendshipID int         `json:"friendshipId"`
	User         UserSummary `json:"user"`
	Direction    string      `json:"direction"` // "incoming" or "outgoing"
	Status       string      `json:"status"`
	CreatedAt    time.Time   `json:"createdAt"`
}

// FriendSearchResult describes a search match including any existing relationship
type FriendSearchResult struct {
	UserID             string `json:"userId"`
	Username           string `json:"username"`
	Points             int    `json:"points"`
	Level              int    `json:"level"`
	RelationshipStatus string `json:"relationshipStatus"`
	RequestDirection   string `json:"requestDirection,omitempty"`
}

// FriendActivityEntry represents a friend's recent activity summary
type FriendActivityEntry struct {
	UserID       string `json:"userId"`
	Username     string `json:"username"`
	Points       int    `json:"points"`
	Level        int    `json:"level"`
	BestScore    int    `json:"bestScore"`
	AttemptsUsed int    `json:"attemptsUsed"`
	Date         string `json:"date"`
}
