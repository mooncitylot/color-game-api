package datastore

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/color-game/api/models"
)

type FriendRepository interface {
	CreateFriendRequest(requesterID, addresseeID string) (models.Friendship, error)
	UpdateFriendshipStatus(friendshipID int, status string) (models.Friendship, error)
	GetFriendshipBetween(userID, otherUserID string) (models.Friendship, error)
	ListFriends(userID string) ([]models.FriendSummary, error)
	ListFriendRequests(userID string) ([]models.FriendRequestSummary, error)
	SearchUsersForFriend(userID string, query string, limit int) ([]models.FriendSearchResult, error)
	RecordFriendActivity(userID string, date time.Time, bestScore, attemptsUsed int) error
	GetFriendActivities(userID string, limitDays int) ([]models.FriendActivityEntry, error)
	DeleteFriendship(friendshipID int, userID string) (models.Friendship, error)
}

type FriendDatabase struct {
	database *sql.DB
}

func NewFriendDatabase(db *sql.DB) (FriendDatabase, error) {
	return FriendDatabase{database: db}, nil
}

func (fr FriendDatabase) CreateFriendRequest(requesterID, addresseeID string) (models.Friendship, error) {
	if requesterID == addresseeID {
		return models.Friendship{}, fmt.Errorf("cannot friend yourself")
	}

	sqlStatement := `
		INSERT INTO friendships (requester_id, addressee_id, status)
		VALUES ($1, $2, $3)
		RETURNING friendship_id, requester_id, addressee_id, status, created_at, responded_at`

	var friendship models.Friendship
	err := fr.database.QueryRow(sqlStatement, requesterID, addresseeID, models.FriendshipStatusPending).Scan(
		&friendship.FriendshipID,
		&friendship.RequesterID,
		&friendship.AddresseeID,
		&friendship.Status,
		&friendship.CreatedAt,
		&friendship.RespondedAt,
	)
	if err != nil {
		return models.Friendship{}, err
	}
	return friendship, nil
}

func (fr FriendDatabase) UpdateFriendshipStatus(friendshipID int, status string) (models.Friendship, error) {
	if status != models.FriendshipStatusAccepted && status != models.FriendshipStatusDeclined {
		return models.Friendship{}, fmt.Errorf("invalid status")
	}

	sqlStatement := `
		UPDATE friendships
		SET status = $2, responded_at = NOW()
		WHERE friendship_id = $1
		RETURNING friendship_id, requester_id, addressee_id, status, created_at, responded_at`

	var friendship models.Friendship
	err := fr.database.QueryRow(sqlStatement, friendshipID, status).Scan(
		&friendship.FriendshipID,
		&friendship.RequesterID,
		&friendship.AddresseeID,
		&friendship.Status,
		&friendship.CreatedAt,
		&friendship.RespondedAt,
	)
	if err != nil {
		return models.Friendship{}, err
	}
	return friendship, nil
}

func (fr FriendDatabase) GetFriendshipBetween(userID, otherUserID string) (models.Friendship, error) {
	sqlStatement := `
		SELECT friendship_id, requester_id, addressee_id, status, created_at, responded_at
		FROM friendships
		WHERE (requester_id = $1 AND addressee_id = $2)
			OR (requester_id = $2 AND addressee_id = $1)`

	var friendship models.Friendship
	err := fr.database.QueryRow(sqlStatement, userID, otherUserID).Scan(
		&friendship.FriendshipID,
		&friendship.RequesterID,
		&friendship.AddresseeID,
		&friendship.Status,
		&friendship.CreatedAt,
		&friendship.RespondedAt,
	)
	if err != nil {
		return models.Friendship{}, err
	}
	return friendship, nil
}

func (fr FriendDatabase) ListFriends(userID string) ([]models.FriendSummary, error) {
	sqlStatement := `
		SELECT f.friendship_id, f.created_at, f.responded_at, 
			CASE WHEN f.requester_id = $1 THEN u_addressee.user_id ELSE u_requester.user_id END AS friend_user_id,
			CASE WHEN f.requester_id = $1 THEN u_addressee.username ELSE u_requester.username END AS friend_username,
			CASE WHEN f.requester_id = $1 THEN u_addressee.points ELSE u_requester.points END AS friend_points,
			CASE WHEN f.requester_id = $1 THEN u_addressee.level ELSE u_requester.level END AS friend_level
		FROM friendships f
		JOIN users u_requester ON f.requester_id = u_requester.user_id
		JOIN users u_addressee ON f.addressee_id = u_addressee.user_id
		WHERE (f.requester_id = $1 OR f.addressee_id = $1)
			AND f.status = $2
		ORDER BY f.responded_at DESC NULLS LAST`

	rows, err := fr.database.Query(sqlStatement, userID, models.FriendshipStatusAccepted)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []models.FriendSummary
	for rows.Next() {
		var friend models.FriendSummary
		var summary models.UserSummary
		err := rows.Scan(
			&friend.FriendshipID,
			&friend.CreatedAt,
			&friend.RespondedAt,
			&summary.UserID,
			&summary.Username,
			&summary.Points,
			&summary.Level,
		)
		if err != nil {
			return nil, err
		}
		friend.Friend = summary
		friend.Status = models.FriendshipStatusAccepted
		friends = append(friends, friend)
	}

	return friends, rows.Err()
}

func (fr FriendDatabase) ListFriendRequests(userID string) ([]models.FriendRequestSummary, error) {
	sqlStatement := `
		SELECT f.friendship_id, f.created_at, f.status,
			CASE WHEN f.addressee_id = $1 THEN 'incoming' ELSE 'outgoing' END AS direction,
			CASE WHEN f.addressee_id = $1 THEN u_requester.user_id ELSE u_addressee.user_id END AS other_user_id,
			CASE WHEN f.addressee_id = $1 THEN u_requester.username ELSE u_addressee.username END AS other_username,
			CASE WHEN f.addressee_id = $1 THEN u_requester.points ELSE u_addressee.points END AS other_points,
			CASE WHEN f.addressee_id = $1 THEN u_requester.level ELSE u_addressee.level END AS other_level
		FROM friendships f
		JOIN users u_requester ON f.requester_id = u_requester.user_id
		JOIN users u_addressee ON f.addressee_id = u_addressee.user_id
		WHERE (f.requester_id = $1 OR f.addressee_id = $1) AND f.status = $2
		ORDER BY f.created_at DESC`

	rows, err := fr.database.Query(sqlStatement, userID, models.FriendshipStatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []models.FriendRequestSummary
	for rows.Next() {
		var request models.FriendRequestSummary
		var summary models.UserSummary
		err := rows.Scan(
			&request.FriendshipID,
			&request.CreatedAt,
			&request.Status,
			&request.Direction,
			&summary.UserID,
			&summary.Username,
			&summary.Points,
			&summary.Level,
		)
		if err != nil {
			return nil, err
		}
		request.User = summary
		requests = append(requests, request)
	}

	return requests, rows.Err()
}

func (fr FriendDatabase) SearchUsersForFriend(userID string, query string, limit int) ([]models.FriendSearchResult, error) {
	if limit <= 0 {
		limit = 10
	}
	searchTerm := fmt.Sprintf("%%%s%%", strings.ToLower(query))

	sqlStatement := `
		WITH friend_status AS (
			SELECT requester_id, addressee_id, status
			FROM friendships
			WHERE requester_id = $1 OR addressee_id = $1
		)
		SELECT u.user_id, u.username, u.points, u.level,
			COALESCE(fs.status, '') AS status,
			CASE 
				WHEN fs.requester_id = $1 THEN 'outgoing'
				WHEN fs.addressee_id = $1 THEN 'incoming'
				ELSE ''
			END AS direction
		FROM users u
		LEFT JOIN friend_status fs
			ON (fs.requester_id = u.user_id OR fs.addressee_id = u.user_id)
		WHERE LOWER(u.username) LIKE $2 AND u.user_id <> $1
		ORDER BY u.username ASC
		LIMIT $3`

	rows, err := fr.database.Query(sqlStatement, userID, searchTerm, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.FriendSearchResult
	for rows.Next() {
		var result models.FriendSearchResult
		err := rows.Scan(
			&result.UserID,
			&result.Username,
			&result.Points,
			&result.Level,
			&result.RelationshipStatus,
			&result.RequestDirection,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, rows.Err()
}

func (fr FriendDatabase) RecordFriendActivity(userID string, date time.Time, bestScore, attemptsUsed int) error {
	sqlStatement := `
		INSERT INTO friend_activity (user_id, date, best_score, attempts_used)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, date)
		DO UPDATE SET best_score = EXCLUDED.best_score, attempts_used = EXCLUDED.attempts_used, created_at = NOW()`

	_, err := fr.database.Exec(sqlStatement, userID, date, bestScore, attemptsUsed)
	return err
}

func (fr FriendDatabase) DeleteFriendship(friendshipID int, userID string) (models.Friendship, error) {
	sqlStatement := `
		DELETE FROM friendships
		WHERE friendship_id = $1
			AND (requester_id = $2 OR addressee_id = $2)
		RETURNING friendship_id, requester_id, addressee_id, status, created_at, responded_at`

	var friendship models.Friendship
	err := fr.database.QueryRow(sqlStatement, friendshipID, userID).Scan(
		&friendship.FriendshipID,
		&friendship.RequesterID,
		&friendship.AddresseeID,
		&friendship.Status,
		&friendship.CreatedAt,
		&friendship.RespondedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Friendship{}, NoRowsError{true, err}
		}
		return models.Friendship{}, err
	}
	return friendship, nil
}

func (fr FriendDatabase) GetFriendActivities(userID string, limitDays int) ([]models.FriendActivityEntry, error) {
	if limitDays <= 0 {
		limitDays = 7
	}
	sqlStatement := `
		SELECT u.user_id, u.username, u.points, u.level,
			fa.best_score, fa.attempts_used, fa.date
		FROM friend_activity fa
		JOIN friendships f 
			ON ((f.requester_id = fa.user_id AND f.addressee_id = $1) OR (f.addressee_id = fa.user_id AND f.requester_id = $1))
		JOIN users u ON u.user_id = fa.user_id
		WHERE f.status = $2 AND fa.date >= NOW()::date - $3 * INTERVAL '1 day'
		ORDER BY fa.date DESC, fa.best_score DESC`

	rows, err := fr.database.Query(sqlStatement, userID, models.FriendshipStatusAccepted, limitDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []models.FriendActivityEntry
	for rows.Next() {
		var activity models.FriendActivityEntry
		err := rows.Scan(
			&activity.UserID,
			&activity.Username,
			&activity.Points,
			&activity.Level,
			&activity.BestScore,
			&activity.AttemptsUsed,
			&activity.Date,
		)
		if err != nil {
			return nil, err
		}
		activity.Date = activity.Date[:10]
		activities = append(activities, activity)
	}

	return activities, rows.Err()
}
