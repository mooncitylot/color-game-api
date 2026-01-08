package datastore

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/color-game/api/models"
	_ "github.com/lib/pq"
)

type DailyLeaderboardRepository interface {
	CreateOrUpdate(entry models.DailyLeaderboard) (models.DailyLeaderboard, error)
	GetByUserAndDate(userID string, date time.Time) (models.DailyLeaderboard, error)
	GetLeaderboardByDate(date time.Time, limit int) ([]models.LeaderboardEntry, error)
	GetUserRankByDate(userID string, date time.Time) (int, error)
}

type DailyLeaderboardDatabase struct {
	database *sql.DB
}

func NewDailyLeaderboardDatabase(db *sql.DB) (DailyLeaderboardDatabase, error) {
	var dailyLeaderboardDB DailyLeaderboardDatabase
	dailyLeaderboardDB.database = db
	return dailyLeaderboardDB, nil
}

// CreateOrUpdate inserts or updates a leaderboard entry
func (dldb DailyLeaderboardDatabase) CreateOrUpdate(entry models.DailyLeaderboard) (models.DailyLeaderboard, error) {
	db := dldb.database

	sqlStatement := `
		INSERT INTO daily_leaderboard (user_id, date, best_score, attempts_used, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, date)
		DO UPDATE SET
			best_score = EXCLUDED.best_score,
			attempts_used = EXCLUDED.attempts_used,
			updated_at = EXCLUDED.updated_at
		RETURNING id`

	err := db.QueryRow(
		sqlStatement,
		entry.UserID,
		entry.Date,
		entry.BestScore,
		entry.AttemptsUsed,
		entry.CreatedAt,
		entry.UpdatedAt,
	).Scan(&entry.ID)

	if err != nil {
		return models.DailyLeaderboard{}, fmt.Errorf("failed to create or update leaderboard entry: %v", err)
	}

	return entry, nil
}

// GetByUserAndDate retrieves a leaderboard entry for a user on a specific date
func (dldb DailyLeaderboardDatabase) GetByUserAndDate(userID string, date time.Time) (models.DailyLeaderboard, error) {
	db := dldb.database

	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	sqlStatement := `
		SELECT id, user_id, date, best_score, attempts_used, created_at, updated_at
		FROM daily_leaderboard
		WHERE user_id = $1 AND date = $2`

	var entry models.DailyLeaderboard
	err := db.QueryRow(sqlStatement, userID, normalizedDate).Scan(
		&entry.ID,
		&entry.UserID,
		&entry.Date,
		&entry.BestScore,
		&entry.AttemptsUsed,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)

	switch err {
	case sql.ErrNoRows:
		return models.DailyLeaderboard{}, NoRowsError{true, err}
	case nil:
		return entry, nil
	default:
		return models.DailyLeaderboard{}, err
	}
}

// GetLeaderboardByDate retrieves the leaderboard for a specific date with rank
func (dldb DailyLeaderboardDatabase) GetLeaderboardByDate(date time.Time, limit int) ([]models.LeaderboardEntry, error) {
	db := dldb.database

	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	sqlStatement := `
		SELECT 
			ROW_NUMBER() OVER (ORDER BY dl.best_score DESC, dl.attempts_used ASC, dl.created_at ASC) as rank,
			dl.user_id,
			u.username,
			dl.best_score,
			dl.attempts_used
		FROM daily_leaderboard dl
		JOIN users u ON dl.user_id = u.user_id
		WHERE dl.date = $1
		ORDER BY dl.best_score DESC, dl.attempts_used ASC, dl.created_at ASC
		LIMIT $2`

	rows, err := db.Query(sqlStatement, normalizedDate, limit)
	if err != nil {
		return []models.LeaderboardEntry{}, err
	}
	defer rows.Close()

	var entries []models.LeaderboardEntry
	for rows.Next() {
		var entry models.LeaderboardEntry
		err := rows.Scan(
			&entry.Rank,
			&entry.UserID,
			&entry.Username,
			&entry.BestScore,
			&entry.AttemptsUsed,
		)
		if err != nil {
			return []models.LeaderboardEntry{}, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// GetUserRankByDate retrieves a user's rank for a specific date
func (dldb DailyLeaderboardDatabase) GetUserRankByDate(userID string, date time.Time) (int, error) {
	db := dldb.database

	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	sqlStatement := `
		WITH ranked_leaderboard AS (
			SELECT 
				user_id,
				ROW_NUMBER() OVER (ORDER BY best_score DESC, attempts_used ASC, created_at ASC) as rank
			FROM daily_leaderboard
			WHERE date = $1
		)
		SELECT rank
		FROM ranked_leaderboard
		WHERE user_id = $2`

	var rank int
	err := db.QueryRow(sqlStatement, normalizedDate, userID).Scan(&rank)

	switch err {
	case sql.ErrNoRows:
		return 0, NoRowsError{true, err}
	case nil:
		return rank, nil
	default:
		return 0, err
	}
}
