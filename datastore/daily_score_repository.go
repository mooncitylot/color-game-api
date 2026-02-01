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
	DeleteUserScoresByDate(userID string, date time.Time) (int64, error)
	SetDailyAttemptModifier(userID string, date time.Time, extraAttempts int) (models.DailyAttemptModifier, error)
	GetDailyAttemptModifier(userID string, date time.Time) (models.DailyAttemptModifier, error)
}

type DailyScoreDatabase struct {
	database *sql.DB
}

func NewDailyScoreDatabase(db *sql.DB) (DailyScoreDatabase, error) {
	var dailyScoreDB DailyScoreDatabase
	dailyScoreDB.database = db
	return dailyScoreDB, nil
}

// SetDailyAttemptModifier upserts extra attempt allowances for a user on a date
func (dsdb DailyScoreDatabase) SetDailyAttemptModifier(userID string, date time.Time, extraAttempts int) (models.DailyAttemptModifier, error) {
	db := dsdb.database

	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	query := `
		INSERT INTO daily_attempt_modifiers (user_id, date, extra_attempts, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id, date)
		DO UPDATE SET extra_attempts = daily_attempt_modifiers.extra_attempts + EXCLUDED.extra_attempts,
			updated_at = NOW()
		RETURNING modifier_id, user_id, date, extra_attempts, created_at, updated_at`

	var modifier models.DailyAttemptModifier
	if err := db.QueryRow(query, userID, normalizedDate, extraAttempts).Scan(
		&modifier.ModifierID,
		&modifier.UserID,
		&modifier.Date,
		&modifier.ExtraAttempts,
		&modifier.CreatedAt,
		&modifier.UpdatedAt,
	); err != nil {
		return models.DailyAttemptModifier{}, fmt.Errorf("failed to set attempt modifier: %v", err)
	}

	return modifier, nil
}

// GetDailyAttemptModifier fetches attempt bonuses for a user on a date
func (dsdb DailyScoreDatabase) GetDailyAttemptModifier(userID string, date time.Time) (models.DailyAttemptModifier, error) {
	db := dsdb.database

	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	query := `
		SELECT modifier_id, user_id, date, extra_attempts, created_at, updated_at
		FROM daily_attempt_modifiers
		WHERE user_id = $1 AND date = $2`

	var modifier models.DailyAttemptModifier
	err := db.QueryRow(query, userID, normalizedDate).Scan(
		&modifier.ModifierID,
		&modifier.UserID,
		&modifier.Date,
		&modifier.ExtraAttempts,
		&modifier.CreatedAt,
		&modifier.UpdatedAt,
	)

	switch err {
	case sql.ErrNoRows:
		return models.DailyAttemptModifier{}, NoRowsError{true, err}
	case nil:
		return modifier, nil
	default:
		return models.DailyAttemptModifier{}, err
	}
}

// DeleteUserScoresByDate removes all attempts for a user on a specific date
func (dsdb DailyScoreDatabase) DeleteUserScoresByDate(userID string, date time.Time) (int64, error) {
	db := dsdb.database

	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	result, err := db.Exec(`
		DELETE FROM daily_scores
		WHERE user_id = $1 AND date = $2
	`, userID, normalizedDate)
	if err != nil {
		return 0, fmt.Errorf("failed to delete daily scores: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
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
