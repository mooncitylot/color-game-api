-- Color Game Database Schema

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    user_id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    kind VARCHAR(50) NOT NULL DEFAULT 'Player',
    approved BOOLEAN NOT NULL DEFAULT true,
    points INTEGER NOT NULL DEFAULT 0,
    level INTEGER NOT NULL DEFAULT 1,
    credits INTEGER NOT NULL DEFAULT 0,
    user_effect TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create user_devices table for JWT refresh tokens
CREATE TABLE IF NOT EXISTS user_devices (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    device_data TEXT,
    fingerprint VARCHAR(255) NOT NULL,
    expiry TIMESTAMP NOT NULL,
    UNIQUE(fingerprint, user_id)
);

-- Create daily_color table for storing one color per day
CREATE TABLE IF NOT EXISTS daily_color (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    color_name VARCHAR(255) NOT NULL,
    r INTEGER NOT NULL CHECK (r >= 0 AND r <= 255),
    g INTEGER NOT NULL CHECK (g >= 0 AND g <= 255),
    b INTEGER NOT NULL CHECK (b >= 0 AND b <= 255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_user_devices_user_id ON user_devices(user_id);
CREATE INDEX idx_user_devices_fingerprint ON user_devices(fingerprint);
CREATE INDEX idx_daily_color_date ON daily_color(date);
