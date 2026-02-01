-- Migration: enable extra daily attempts and powerup tracking

-- Relax attempt limits to allow powerups to grant additional attempts
ALTER TABLE daily_scores
    DROP CONSTRAINT IF EXISTS daily_scores_attempt_number_check;
ALTER TABLE daily_scores
    ADD CONSTRAINT daily_scores_attempt_number_check
    CHECK (attempt_number >= 1 AND attempt_number <= 10);

ALTER TABLE daily_leaderboard
    DROP CONSTRAINT IF EXISTS daily_leaderboard_attempts_used_check;
ALTER TABLE daily_leaderboard
    ADD CONSTRAINT daily_leaderboard_attempts_used_check
    CHECK (attempts_used >= 1 AND attempts_used <= 10);

-- Track daily extra attempts granted from consumable powerups
CREATE TABLE IF NOT EXISTS daily_attempt_modifiers (
    modifier_id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    extra_attempts INTEGER NOT NULL DEFAULT 0 CHECK (extra_attempts >= 0),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, date)
);

CREATE INDEX IF NOT EXISTS idx_daily_attempt_modifiers_user_date
    ON daily_attempt_modifiers(user_id, date);
