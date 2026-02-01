-- Migration: Create friendships and friend activity support

CREATE TABLE IF NOT EXISTS friendships (
    friendship_id SERIAL PRIMARY KEY,
    requester_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    addressee_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    responded_at TIMESTAMP,
    CHECK (requester_id <> addressee_id)
);

-- Ensure we only ever have a single friendship row between two users
CREATE UNIQUE INDEX IF NOT EXISTS idx_friendships_unique_pair
    ON friendships (LEAST(requester_id, addressee_id), GREATEST(requester_id, addressee_id));

CREATE INDEX IF NOT EXISTS idx_friendships_requester_status
    ON friendships (requester_id, status);

CREATE INDEX IF NOT EXISTS idx_friendships_addressee_status
    ON friendships (addressee_id, status);
