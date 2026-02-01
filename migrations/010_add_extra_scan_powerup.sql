-- Migration: Add Extra Scan powerup item
-- This powerup grants users one additional scan attempt for the daily color challenge

INSERT INTO shop_items (item_id, item_type, name, description, credit_cost, rarity, metadata, is_active, is_limited_edition, stock_quantity, created_at, updated_at)
VALUES 
    (
        'e7fae62a-b456-40f7-bc0a-8bff747c6783',
        'powerup',
        'Extra Scan',
        'Gives you an extra scan. ',
        100,
        'common',
        '{"effect_type": "extra_attempt", "extra_attempts": 1}'::jsonb,
        true,
        false,
        NULL,
        NOW(),
        NOW()
    )
ON CONFLICT (item_id) DO UPDATE SET
    description = EXCLUDED.description,
    metadata = EXCLUDED.metadata,
    updated_at = NOW();
