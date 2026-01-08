-- Migration: Create daily_color table
-- This table stores one color per day for the color game

CREATE TABLE IF NOT EXISTS daily_color (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    color_name VARCHAR(255) NOT NULL,
    r INTEGER NOT NULL CHECK (r >= 0 AND r <= 255),
    g INTEGER NOT NULL CHECK (g >= 0 AND g <= 255),
    b INTEGER NOT NULL CHECK (b >= 0 AND b <= 255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create index on date for fast lookups
CREATE INDEX IF NOT EXISTS idx_daily_color_date ON daily_color(date);
