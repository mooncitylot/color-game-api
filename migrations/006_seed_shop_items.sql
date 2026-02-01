-- Migration: Seed shop items with sample data
-- This migration adds example items for testing the shop system

-- Powerups
INSERT INTO shop_items (item_id, item_type, name, description, credit_cost, rarity, metadata, is_active, is_limited_edition, stock_quantity, created_at, updated_at)
VALUES 
    (
        'powerup-double-points-001',
        'powerup',
        'Double Points',
        'Doubles your points for 60 seconds',
        100,
        'common',
        '{"icon_url": "/assets/powerups/double-points.png", "duration_seconds": 60, "effect_type": "points_multiplier", "multiplier": 2.0}'::jsonb,
        true,
        false,
        NULL,
        NOW(),
        NOW()
    ),
    (
        'powerup-triple-points-001',
        'powerup',
        'Triple Points',
        'Triples your points for 30 seconds',
        250,
        'rare',
        '{"icon_url": "/assets/powerups/triple-points.png", "duration_seconds": 30, "effect_type": "points_multiplier", "multiplier": 3.0}'::jsonb,
        true,
        false,
        NULL,
        NOW(),
        NOW()
    ),
    (
        'powerup-hint-001',
        'powerup',
        'Color Hint',
        'Reveals a hint about the target color',
        50,
        'common',
        '{"icon_url": "/assets/powerups/hint.png", "effect_type": "hint", "hint_type": "rgb_range"}'::jsonb,
        true,
        false,
        NULL,
        NOW(),
        NOW()
    );

-- Badges
INSERT INTO shop_items (item_id, item_type, name, description, credit_cost, rarity, metadata, is_active, is_limited_edition, stock_quantity, created_at, updated_at)
VALUES 
    (
        'badge-beginner-001',
        'badge',
        'Beginner Badge',
        'Show everyone you''re just getting started',
        25,
        'common',
        '{"icon_url": "/assets/badges/beginner.png", "display_order": 1}'::jsonb,
        true,
        false,
        NULL,
        NOW(),
        NOW()
    ),
    (
        'badge-champion-001',
        'badge',
        'Champion Badge',
        'For the color matching champions',
        500,
        'epic',
        '{"icon_url": "/assets/badges/champion.png", "display_order": 2}'::jsonb,
        true,
        false,
        NULL,
        NOW(),
        NOW()
    ),
    (
        'badge-legend-001',
        'badge',
        'Legend Badge',
        'The ultimate badge for color masters',
        1000,
        'legendary',
        '{"icon_url": "/assets/badges/legend.png", "display_order": 3}'::jsonb,
        true,
        false,
        NULL,
        NOW(),
        NOW()
    );

-- Avatar Hats
INSERT INTO shop_items (item_id, item_type, name, description, credit_cost, rarity, metadata, is_active, is_limited_edition, stock_quantity, created_at, updated_at)
VALUES 
    (
        'hat-baseball-001',
        'avatar_hat',
        'Baseball Cap',
        'A classic baseball cap for your avatar',
        75,
        'common',
        '{"icon_url": "/assets/hats/baseball-cap.png", "sprite_url": "/assets/hats/baseball-cap-sprite.png", "layer": "head", "color_customizable": true}'::jsonb,
        true,
        false,
        NULL,
        NOW(),
        NOW()
    ),
    (
        'hat-wizard-001',
        'avatar_hat',
        'Wizard Hat',
        'Channel your inner wizard with this magical hat',
        200,
        'rare',
        '{"icon_url": "/assets/hats/wizard-hat.png", "sprite_url": "/assets/hats/wizard-hat-sprite.png", "layer": "head", "color_customizable": false}'::jsonb,
        true,
        false,
        NULL,
        NOW(),
        NOW()
    ),
    (
        'hat-crown-001',
        'avatar_hat',
        'Golden Crown',
        'Only for the true royalty of color matching',
        750,
        'legendary',
        '{"icon_url": "/assets/hats/crown.png", "sprite_url": "/assets/hats/crown-sprite.png", "layer": "head", "color_customizable": false}'::jsonb,
        true,
        false,
        NULL,
        NOW(),
        NOW()
    );

-- Avatar Skins
INSERT INTO shop_items (item_id, item_type, name, description, credit_cost, rarity, metadata, is_active, is_limited_edition, stock_quantity, created_at, updated_at)
VALUES 
    (
        'skin-rainbow-001',
        'avatar_skin',
        'Rainbow Skin',
        'Show off your love for colors with this rainbow skin',
        300,
        'epic',
        '{"icon_url": "/assets/skins/rainbow.png", "sprite_url": "/assets/skins/rainbow-sprite.png", "animated": true}'::jsonb,
        true,
        false,
        NULL,
        NOW(),
        NOW()
    ),
    (
        'skin-golden-001',
        'avatar_skin',
        'Golden Skin',
        'Shine bright like gold',
        600,
        'legendary',
        '{"icon_url": "/assets/skins/golden.png", "sprite_url": "/assets/skins/golden-sprite.png", "animated": true, "particle_effect": "sparkle"}'::jsonb,
        true,
        false,
        NULL,
        NOW(),
        NOW()
    );

-- Limited Edition Items (for special events)
INSERT INTO shop_items (item_id, item_type, name, description, credit_cost, rarity, metadata, is_active, is_limited_edition, stock_quantity, created_at, updated_at)
VALUES 
    (
        'badge-holiday-2024',
        'badge',
        'Holiday 2024 Badge',
        'Limited edition holiday badge - only 100 available!',
        250,
        'epic',
        '{"icon_url": "/assets/badges/holiday-2024.png", "display_order": 99, "event": "holiday_2024"}'::jsonb,
        true,
        true,
        100,
        NOW(),
        NOW()
    );
