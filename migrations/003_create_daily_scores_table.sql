-- Migration: Create daily_scores table
-- Tracks all individual attempts users make each day

CREATE TABLE IF NOT EXISTS daily_scores (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    attempt_number INTEGER NOT NULL CHECK (attempt_number >= 1 AND attempt_number <= 5),
    score INTEGER NOT NULL CHECK (score >= 0 AND score <= 100),
    submitted_color_r INTEGER NOT NULL CHECK (submitted_color_r >= 0 AND submitted_color_r <= 255),
    submitted_color_g INTEGER NOT NULL CHECK (submitted_color_g >= 0 AND submitted_color_g <= 255),
    submitted_color_b INTEGER NOT NULL CHECK (submitted_color_b >= 0 AND submitted_color_b <= 255),
    target_color_r INTEGER NOT NULL CHECK (target_color_r >= 0 AND target_color_r <= 255),
    target_color_g INTEGER NOT NULL CHECK (target_color_g >= 0 AND target_color_g <= 255),
    target_color_b INTEGER NOT NULL CHECK (target_color_b >= 0 AND target_color_b <= 255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, date, attempt_number)
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_daily_scores_user_date ON daily_scores(user_id, date);
CREATE INDEX IF NOT EXISTS idx_daily_scores_date ON daily_scores(date);
CREATE INDEX IF NOT EXISTS idx_daily_scores_user_id ON daily_scores(user_id);
