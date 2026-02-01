-- Migration: Create shop system tables
-- This migration adds shop items, user inventory, and purchase history

-- Create shop_items table
CREATE TABLE IF NOT EXISTS shop_items (
    item_id VARCHAR(255) PRIMARY KEY,
    item_type VARCHAR(50) NOT NULL, -- 'powerup', 'badge', 'avatar_hat', 'avatar_skin', etc.
    name VARCHAR(255) NOT NULL,
    description TEXT,
    credit_cost INTEGER NOT NULL,
    rarity VARCHAR(50), -- 'common', 'rare', 'epic', 'legendary'
    metadata JSONB, -- Flexible storage for item-specific properties
    is_active BOOLEAN NOT NULL DEFAULT true, -- Can disable items without deleting
    is_limited_edition BOOLEAN NOT NULL DEFAULT false,
    stock_quantity INTEGER, -- NULL = unlimited, otherwise limited stock
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create user_inventory table
CREATE TABLE IF NOT EXISTS user_inventory (
    inventory_id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    item_id VARCHAR(255) NOT NULL REFERENCES shop_items(item_id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL DEFAULT 1, -- For stackable items like powerups
    is_equipped BOOLEAN NOT NULL DEFAULT false, -- For cosmetics (badges, hats)
    acquired_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP, -- For temporary items
    used_count INTEGER NOT NULL DEFAULT 0, -- Track usage for consumables
    UNIQUE(user_id, item_id) -- Prevent duplicate entries (update quantity instead)
);

-- Create purchase_history table
CREATE TABLE IF NOT EXISTS purchase_history (
    purchase_id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    item_id VARCHAR(255) NOT NULL REFERENCES shop_items(item_id),
    quantity INTEGER NOT NULL DEFAULT 1,
    credits_spent INTEGER NOT NULL,
    purchased_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for shop_items
CREATE INDEX IF NOT EXISTS idx_shop_items_type ON shop_items(item_type);
CREATE INDEX IF NOT EXISTS idx_shop_items_active ON shop_items(is_active);
CREATE INDEX IF NOT EXISTS idx_shop_items_rarity ON shop_items(rarity);

-- Create indexes for user_inventory
CREATE INDEX IF NOT EXISTS idx_user_inventory_user_id ON user_inventory(user_id);
CREATE INDEX IF NOT EXISTS idx_user_inventory_item_id ON user_inventory(item_id);
CREATE INDEX IF NOT EXISTS idx_user_inventory_equipped ON user_inventory(user_id, is_equipped);
CREATE INDEX IF NOT EXISTS idx_user_inventory_expires_at ON user_inventory(expires_at);

-- Create indexes for purchase_history
CREATE INDEX IF NOT EXISTS idx_purchase_history_user_id ON purchase_history(user_id);
CREATE INDEX IF NOT EXISTS idx_purchase_history_item_id ON purchase_history(item_id);
CREATE INDEX IF NOT EXISTS idx_purchase_history_date ON purchase_history(purchased_at);
