# Daily Color Migration

## Running the Migration

To create the `daily_color` table in your database, run:

```sql
-- From schema.sql (for new databases)
-- Or run the migration file:
psql -U your_username -d colorgame -f migrations/002_create_daily_color_table.sql
```

Or connect to your database and run:

```sql
CREATE TABLE IF NOT EXISTS daily_color (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    color_name VARCHAR(255) NOT NULL,
    r INTEGER NOT NULL CHECK (r >= 0 AND r <= 255),
    g INTEGER NOT NULL CHECK (g >= 0 AND g <= 255),
    b INTEGER NOT NULL CHECK (b >= 0 AND b <= 255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_daily_color_date ON daily_color(date);
```

## Features

The daily color system automatically:
- Generates a new random color every day at midnight
- Stores the color in the database with date, color name, and RGB values
- Prevents duplicate entries for the same date

## API Endpoints

### Get Today's Color
```
GET /v1/colors/daily
```

Response:
```json
{
  "date": "2026-01-07",
  "color_name": "Sunset Orange",
  "rgb": "rgb(255,128,64)",
  "hex": "#FF8040"
}
```

### Get All Daily Colors
```
GET /v1/colors/daily/all
```

Response:
```json
[
  {
    "date": "2026-01-07",
    "color_name": "Sunset Orange",
    "rgb": "rgb(255,128,64)",
    "hex": "#FF8040"
  },
  ...
]
```

## Scheduler

The scheduler automatically runs at midnight every day and:
1. Generates random RGB values
2. Calls thecolorapi.com to get the color name
3. Saves the color to the database with today's date
4. Prevents duplicates if the color already exists for today

The scheduler starts automatically when the server starts.
