package models

import "time"

// DailyScore represents a single attempt by a user on a specific day
type DailyScore struct {
	ID              int       `json:"id"`
	UserID          string    `json:"user_id"`
	Date            time.Time `json:"date"`
	AttemptNumber   int       `json:"attempt_number"`
	Score           int       `json:"score"`
	SubmittedColorR int       `json:"submitted_color_r"`
	SubmittedColorG int       `json:"submitted_color_g"`
	SubmittedColorB int       `json:"submitted_color_b"`
	TargetColorR    int       `json:"target_color_r"`
	TargetColorG    int       `json:"target_color_g"`
	TargetColorB    int       `json:"target_color_b"`
	CreatedAt       time.Time `json:"created_at"`
}

// DailyLeaderboard represents a user's best score for a specific day
type DailyLeaderboard struct {
	ID           int       `json:"id"`
	UserID       string    `json:"user_id"`
	Date         time.Time `json:"date"`
	BestScore    int       `json:"best_score"`
	AttemptsUsed int       `json:"attempts_used"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ScoreSubmissionRequest represents a request to submit a score
type ScoreSubmissionRequest struct {
	SubmittedColorR int `json:"submitted_color_r"`
	SubmittedColorG int `json:"submitted_color_g"`
	SubmittedColorB int `json:"submitted_color_b"`
}

// ScoreSubmissionResponse represents the response after submitting a score
type ScoreSubmissionResponse struct {
	Score          int    `json:"score"`
	AttemptNumber  int    `json:"attempt_number"`
	AttemptsLeft   int    `json:"attempts_left"`
	BestScore      int    `json:"best_score"`
	IsNewBest      bool   `json:"is_new_best"`
	SubmittedColor string `json:"submitted_color"`
	TargetColor    string `json:"target_color"`
	Message        string `json:"message"`
}

// LeaderboardEntry represents a single entry in the leaderboard
type LeaderboardEntry struct {
	Rank         int    `json:"rank"`
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	BestScore    int    `json:"best_score"`
	AttemptsUsed int    `json:"attempts_used"`
}

// UserScoreHistory represents a user's score history for a specific day
type UserScoreHistory struct {
	Date         string       `json:"date"`
	Attempts     []DailyScore `json:"attempts"`
	BestScore    int          `json:"best_score"`
	AttemptsUsed int          `json:"attempts_used"`
	AttemptsLeft int          `json:"attempts_left"`
}
