package datastore

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/color-game/api/models"
	_ "github.com/lib/pq"
)

type DailyColorRepository interface {
	Create(dailyColor models.DailyColor) (models.DailyColor, error)
	GetByDate(date time.Time) (models.DailyColor, error)
	GetToday() (models.DailyColor, error)
	GetAll() ([]models.DailyColor, error)
	Delete(id int) error
}

type DailyColorDatabase struct {
	database *sql.DB
}

func NewDailyColorDatabase(db *sql.DB) (DailyColorDatabase, error) {
	var dailyColorDB DailyColorDatabase
	dailyColorDB.database = db
	return dailyColorDB, nil
}

// Create inserts a new daily color into the database
func (dcdb DailyColorDatabase) Create(dailyColor models.DailyColor) (models.DailyColor, error) {
	db := dcdb.database

	sqlStatement := `
		INSERT INTO daily_color (date, color_name, r, g, b, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	err := db.QueryRow(
		sqlStatement,
		dailyColor.Date,
		dailyColor.ColorName,
		dailyColor.R,
		dailyColor.G,
		dailyColor.B,
		dailyColor.CreatedAt,
	).Scan(&dailyColor.ID)

	if err != nil {
		return models.DailyColor{}, fmt.Errorf("failed to create daily color: %v", err)
	}

	return dailyColor, nil
}

// GetByDate retrieves a daily color by date
func (dcdb DailyColorDatabase) GetByDate(date time.Time) (models.DailyColor, error) {
	db := dcdb.database

	// Normalize date to start of day
	normalizedDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	sqlStatement := `
		SELECT id, date, color_name, r, g, b, created_at
		FROM daily_color
		WHERE date = $1`

	row := db.QueryRow(sqlStatement, normalizedDate)

	var dailyColor models.DailyColor
	err := row.Scan(
		&dailyColor.ID,
		&dailyColor.Date,
		&dailyColor.ColorName,
		&dailyColor.R,
		&dailyColor.G,
		&dailyColor.B,
		&dailyColor.CreatedAt,
	)

	switch err {
	case sql.ErrNoRows:
		return models.DailyColor{}, NoRowsError{true, err}
	case nil:
		return dailyColor, nil
	default:
		return models.DailyColor{}, err
	}
}

// GetToday retrieves today's daily color
func (dcdb DailyColorDatabase) GetToday() (models.DailyColor, error) {
	today := time.Now()
	return dcdb.GetByDate(today)
}

// GetAll retrieves all daily colors
func (dcdb DailyColorDatabase) GetAll() ([]models.DailyColor, error) {
	db := dcdb.database

	sqlStatement := `
		SELECT id, date, color_name, r, g, b, created_at
		FROM daily_color
		ORDER BY date DESC`

	rows, err := db.Query(sqlStatement)
	if err != nil {
		return []models.DailyColor{}, err
	}
	defer rows.Close()

	var dailyColors []models.DailyColor
	for rows.Next() {
		var dc models.DailyColor
		err := rows.Scan(
			&dc.ID,
			&dc.Date,
			&dc.ColorName,
			&dc.R,
			&dc.G,
			&dc.B,
			&dc.CreatedAt,
		)
		if err != nil {
			return []models.DailyColor{}, err
		}
		dailyColors = append(dailyColors, dc)
	}

	if err = rows.Err(); err != nil {
		return []models.DailyColor{}, err
	}

	return dailyColors, nil
}

// Delete removes a daily color by ID
func (dcdb DailyColorDatabase) Delete(id int) error {
	db := dcdb.database

	sqlStatement := `DELETE FROM daily_color WHERE id = $1`
	_, err := db.Exec(sqlStatement, id)

	return err
}
