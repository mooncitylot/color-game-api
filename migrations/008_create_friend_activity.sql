-- Migration: Track friend activity based on daily scores

CREATE TABLE IF NOT EXISTS friend_activity (
    activity_id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    best_score INTEGER NOT NULL,
    attempts_used INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, date)
);

CREATE INDEX IF NOT EXISTS idx_friend_activity_date ON friend_activity(date);
