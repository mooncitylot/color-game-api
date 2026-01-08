package models

import "time"

// DailyColor represents a color of the day for the game
type DailyColor struct {
	ID        int       `json:"id"`
	Date      time.Time `json:"date"`
	ColorName string    `json:"color_name"`
	R         int       `json:"r"`
	G         int       `json:"g"`
	B         int       `json:"b"`
	CreatedAt time.Time `json:"created_at"`
}

// DailyColorResponse is the simplified response for API endpoints
type DailyColorResponse struct {
	Date      string `json:"date"`
	ColorName string `json:"color_name"`
	RGB       string `json:"rgb"`
	Hex       string `json:"hex"`
}
