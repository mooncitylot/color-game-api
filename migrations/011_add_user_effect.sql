-- Migration: Add user_effect for active powerup/debuff state (JSON text)
-- Example: {"code":"bonus_scan","polarity":"positive"}
--          {"code":"slow_timer","polarity":"negative","appliedByUserId":"<uuid>"}

ALTER TABLE users
ADD COLUMN IF NOT EXISTS user_effect TEXT;
