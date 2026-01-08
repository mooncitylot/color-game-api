-- Migration: Add points, level, and credits to users table
-- Run this if you have an existing database

ALTER TABLE users 
ADD COLUMN IF NOT EXISTS points INTEGER NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS level INTEGER NOT NULL DEFAULT 1,
ADD COLUMN IF NOT EXISTS credits INTEGER NOT NULL DEFAULT 0;
