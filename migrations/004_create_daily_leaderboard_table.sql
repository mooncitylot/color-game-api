-- Migration: Create daily_leaderboard table
-- Stores the best score for each user each day for fast leaderboard queries

CREATE TABLE IF NOT EXISTS daily_leaderboard (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    best_score INTEGER NOT NULL CHECK (best_score >= 0 AND best_score <= 100),
    attempts_used INTEGER NOT NULL CHECK (attempts_used >= 1 AND attempts_used <= 5),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, date)
);

-- Create indexes for leaderboard queries
CREATE INDEX IF NOT EXISTS idx_daily_leaderboard_date_score ON daily_leaderboard(date, best_score DESC);
CREATE INDEX IF NOT EXISTS idx_daily_leaderboard_user_date ON daily_leaderboard(user_id, date);
CREATE INDEX IF NOT EXISTS idx_daily_leaderboard_date ON daily_leaderboard(date);
