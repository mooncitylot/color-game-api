package datastore

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/color-game/api/models"
	_ "github.com/lib/pq"
)

type DailyScoreRepository interface {
	Create(score models.DailyScore) (models.DailyScore, error)
	GetUserScoresByDate(userID string, date time.Time) ([]models.DailyScore, error)
	GetUserAttemptCount(userID string, date time.Time) (int, error)
	GetAllScoresByDate(date time.Time) ([]models.DailyScore, error)
	GetUserScoreHistory(userID string) ([]models.DailyScore, error)
}

type DailyScoreDatabase struct {
	database *sql.DB
}

func NewDailyScoreDatabase(db *sql.DB) (DailyScoreDatabase, error) {
	var dailyScoreDB DailyScoreDatabase
	dailyScoreDB.database = db
	return dailyScoreDB, nil
}

// Create inserts a new daily score
func (dsdb DailyScoreDatabase) Create(score models.DailyScore) (models.DailyScore, error) {
	db := dsdb.database

	sqlStatement := `
		INSERT INTO daily_scores (
			user_id, date, attempt_number, score,
			submitted_color_r, submitted_color_g, submitted_color_b,
			target_color_r, target_color_g, target_color_b,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	err := db.QueryRow(
		sqlStatement,
		score.UserID,
		score.Date,
		score.AttemptNumber,
		score.Score,
		score.SubmittedColorR,
		score.SubmittedColorG,
		score.SubmittedColorB,
		score.TargetColorR,
		score.TargetColorG,
		score.TargetColorB,
		score.CreatedAt,
	).Scan(&score.ID)

	if err != nil {
		return models.DailyScore{}, fmt.Errorf("failed to create daily score: %v", err)
	}

	return score, nil
}

// GetUserScoresByDate retrieves all scores for a user on a specific date
func (dsdb DailyScoreDatabase) GetUserScoresByDate(userID string, date time.Time) ([]models.DailyScore, error) {
	db := dsdb.database

	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	sqlStatement := `
		SELECT id, user_id, date, attempt_number, score,
			submitted_color_r, submitted_color_g, submitted_color_b,
			target_color_r, target_color_g, target_color_b,
			created_at
		FROM daily_scores
		WHERE user_id = $1 AND date = $2
		ORDER BY attempt_number ASC`

	rows, err := db.Query(sqlStatement, userID, normalizedDate)
	if err != nil {
		return []models.DailyScore{}, err
	}
	defer rows.Close()

	var scores []models.DailyScore
	for rows.Next() {
		var score models.DailyScore
		err := rows.Scan(
			&score.ID,
			&score.UserID,
			&score.Date,
			&score.AttemptNumber,
			&score.Score,
			&score.SubmittedColorR,
			&score.SubmittedColorG,
			&score.SubmittedColorB,
			&score.TargetColorR,
			&score.TargetColorG,
			&score.TargetColorB,
			&score.CreatedAt,
		)
		if err != nil {
			return []models.DailyScore{}, err
		}
		scores = append(scores, score)
	}

	return scores, rows.Err()
}

// GetUserAttemptCount returns the number of attempts a user has made on a specific date
func (dsdb DailyScoreDatabase) GetUserAttemptCount(userID string, date time.Time) (int, error) {
	db := dsdb.database

	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	sqlStatement := `
		SELECT COUNT(*)
		FROM daily_scores
		WHERE user_id = $1 AND date = $2`

	var count int
	err := db.QueryRow(sqlStatement, userID, normalizedDate).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetAllScoresByDate retrieves all scores for a specific date
func (dsdb DailyScoreDatabase) GetAllScoresByDate(date time.Time) ([]models.DailyScore, error) {
	db := dsdb.database

	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	sqlStatement := `
		SELECT id, user_id, date, attempt_number, score,
			submitted_color_r, submitted_color_g, submitted_color_b,
			target_color_r, target_color_g, target_color_b,
			created_at
		FROM daily_scores
		WHERE date = $1
		ORDER BY score DESC, created_at ASC`

	rows, err := db.Query(sqlStatement, normalizedDate)
	if err != nil {
		return []models.DailyScore{}, err
	}
	defer rows.Close()

	var scores []models.DailyScore
	for rows.Next() {
		var score models.DailyScore
		err := rows.Scan(
			&score.ID,
			&score.UserID,
			&score.Date,
			&score.AttemptNumber,
			&score.Score,
			&score.SubmittedColorR,
			&score.SubmittedColorG,
			&score.SubmittedColorB,
			&score.TargetColorR,
			&score.TargetColorG,
			&score.TargetColorB,
			&score.CreatedAt,
		)
		if err != nil {
			return []models.DailyScore{}, err
		}
		scores = append(scores, score)
	}

	return scores, rows.Err()
}

// GetUserScoreHistory retrieves all scores for a user across all dates
func (dsdb DailyScoreDatabase) GetUserScoreHistory(userID string) ([]models.DailyScore, error) {
	db := dsdb.database

	sqlStatement := `
		SELECT id, user_id, date, attempt_number, score,
			submitted_color_r, submitted_color_g, submitted_color_b,
			target_color_r, target_color_g, target_color_b,
			created_at
		FROM daily_scores
		WHERE user_id = $1
		ORDER BY date DESC, attempt_number ASC`

	rows, err := db.Query(sqlStatement, userID)
	if err != nil {
		return []models.DailyScore{}, err
	}
	defer rows.Close()

	var scores []models.DailyScore
	for rows.Next() {
		var score models.DailyScore
		err := rows.Scan(
			&score.ID,
			&score.UserID,
			&score.Date,
			&score.AttemptNumber,
			&score.Score,
			&score.SubmittedColorR,
			&score.SubmittedColorG,
			&score.SubmittedColorB,
			&score.TargetColorR,
			&score.TargetColorG,
			&score.TargetColorB,
			&score.CreatedAt,
		)
		if err != nil {
			return []models.DailyScore{}, err
		}
		scores = append(scores, score)
	}

	return scores, rows.Err()
}
