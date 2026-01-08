# Database Migrations

This API now includes automatic database migrations that run on startup.

## How It Works

When you run `./main`, the application will:

1. Connect to the database
2. Create a `schema_migrations` table if it doesn't exist
3. Check which migrations have already been applied
4. Run all pending migrations in order (000, 001, 002, etc.)
5. Track each applied migration in the `schema_migrations` table
6. Start the API server

## Migration Files

Migrations are located in the `migrations/` directory and follow this naming convention:

```
XXX_description.sql
```

Where `XXX` is a three-digit version number (e.g., 000, 001, 002).

### Current Migrations

- **000_create_base_schema.sql** - Creates users and user_devices tables
- **001_add_user_game_fields.sql** - Adds points, level, and credits columns to users
- **002_create_daily_color_table.sql** - Creates daily_color table for the color of the day feature

## Creating New Migrations

To create a new migration:

1. Create a new file in the `migrations/` directory
2. Name it with the next sequential number: `003_your_description.sql`
3. Write your SQL migration code
4. The migration will automatically run on next startup

Example:

```sql
-- migrations/003_add_user_stats.sql
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS total_games_played INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_users_games_played ON users(total_games_played);
```

## Migration Tracking

The system creates a `schema_migrations` table to track which migrations have been applied:

```sql
CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

Each migration is applied in a transaction, so if a migration fails, it will be rolled back.

## Fresh Database Setup

If you're setting up a fresh database, simply:

1. Create an empty PostgreSQL database
2. Update your `.env` file with the database credentials
3. Run `./main`

The application will automatically create all tables and apply all migrations.

## Benefits

- **Automatic**: No manual SQL file execution needed
- **Safe**: Migrations are tracked and never run twice
- **Transactional**: Each migration runs in a transaction
- **Ordered**: Migrations always run in the correct order
- **Idempotent**: Safe to run multiple times (uses IF NOT EXISTS where appropriate)

## Troubleshooting

If a migration fails:

1. Check the error message in the console
2. Fix the migration SQL file
3. Manually remove the failed migration from `schema_migrations` if it was partially applied
4. Restart the application

To check which migrations have been applied:

```sql
SELECT * FROM schema_migrations ORDER BY version;
```
