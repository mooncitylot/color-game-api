# Color Game API

A simple Go REST API for the Color Game application with user authentication and JWT-based authorization.

## Features

- User registration and authentication
- JWT-based authentication with access and refresh tokens
- Device fingerprinting for enhanced security
- PostgreSQL database
- CORS configuration
- Role-based access control (Player/Admin)

## Prerequisites

- Go 1.23 or higher
- PostgreSQL 12 or higher

## Setup

1. **Clone the repository**

2. **Install dependencies**

   ```bash
   go mod download
   ```

3. **Set up PostgreSQL database**

   Create a new database:

   ```bash
   createdb colorgame
   ```

   Run the schema:

   ```bash
   psql -d colorgame -f schema.sql
   ```

4. **Configure environment variables**

   Copy `.env.template` to `.env` and update the values:

   ```bash
   cp .env.template .env
   ```

   Update the following variables in `.env`:
   - `DB_PASSWORD`: Your PostgreSQL password
   - `JWT_SECRET`: A strong secret key for JWT signing
   - `ALLOWED_ORIGINS`: Comma-separated list of allowed frontend origins

5. **Run the server**

   ```bash
   go run main.go
   ```

   The server will start on `http://localhost:8080`

## API Endpoints

### Public Endpoints

- `GET /` - Health check endpoint
- `POST /v1/auth/signup` - User registration

  ```json
  {
    "username": "player1",
    "email": "player1@example.com",
    "password": "securepassword"
  }
  ```

- `POST /v1/auth/login` - User login
  ```json
  {
    "email": "player1@example.com",
    "password": "securepassword",
    "deviceFingerprint": "unique-device-id"
  }
  ```

### Authenticated Endpoints

- `GET /v1/users/me` - Get current user profile

### Admin Endpoints

- `GET /v1/users` - Get all users (Admin only)

## Shop item metadata

Shop rows store a JSON **`metadata`** column (`shop_items.metadata`). The client sees it on each item; **`POST /v1/inventory/use`** reads it to run special behavior.

### Extra daily attempts (powerup)

Used for items such as “Extra Scan”. When present, using the item adjusts that user’s extra attempts for **today** via `effect_type` and optional `extra_attempts`.

```json
{
  "effect_type": "extra_attempt",
  "extra_attempts": 1
}
```

- **`effect_type`**: must be `"extra_attempt"` to trigger this path.
- **`extra_attempts`**: optional positive integer; defaults to `1`.

### Persisted user effect (`users.user_effect`)

When **`metadata.effect`** is set to a JSON object, using the item writes that payload (as a JSON string) onto **`users.user_effect`** for either the user who used the item or a chosen friend.

```json
{
  "effect": {
    "code": "bonus_scan",
    "polarity": "positive"
  },
  "effect_target": "self"
}
```

- **`effect`**: required for this path; arbitrary object. The game client interprets `code`, `polarity`, and other keys.
- **`effect_target`**: optional; default **`"self"`**.
  - **`"self"`**: updates **`user_effect`** for the authenticated user.
  - **`"other"`**: updates **`user_effect`** for another user. The use request **must** include **`targetUserId`**, and the target must be an **accepted friend**. The stored payload includes **`appliedByUserId`** (the user who consumed the item).

Example for a debuff aimed at a friend (still requires `targetUserId` in the use request body):

```json
{
  "effect": {
    "code": "hidden_hint",
    "polarity": "negative"
  },
  "effect_target": "other"
}
```

An item may include **both** `effect_type` / `extra_attempts` and **`effect`** if you want multiple behaviors on one use.

### Using an item from the API

`POST /v1/inventory/use`

```json
{
  "inventoryId": 1,
  "targetUserId": "optional-friend-uuid-for-effect_target-other"
}
```

The response may include **`effectRecipientUserId`** when a **`user_effect`** was written.

### Reading and clearing the active effect

- `GET /v1/users/me/effect` returns `{ "userEffect": "<json string or null>" }`.
- `DELETE /v1/users/me/effect` clears **`user_effect`** (e.g. after the client has consumed the effect in gameplay).

## Authentication

The API uses JWT-based authentication with two types of tokens:

1. **Access Token**: Short-lived (15 minutes by default), used for API requests
2. **Refresh Token**: Long-lived (7 days by default), used to obtain new access tokens

Both tokens are set as HTTP-only cookies for security.

## Development

### Project Structure

```
color-game-api/
├── api/              # HTTP handlers and routing
├── datastore/        # Database layer
├── models/           # Data models
├── main.go           # Application entry point
├── schema.sql        # Database schema
├── .env.template     # Environment variables template
└── README.md         # This file
```

### Building for Production

```bash
go build -o color-game-api main.go
```

## Environment Variables

| Variable             | Description                      | Default               |
| -------------------- | -------------------------------- | --------------------- |
| HTTP_PORT            | Server port                      | :8080                 |
| DB_TYPE              | Database type                    | postgres              |
| DB_USER              | Database user                    | postgres              |
| DB_PASSWORD          | Database password                | (required)            |
| DB_NAME              | Database name                    | colorgame             |
| SSL_MODE             | PostgreSQL SSL mode              | disable               |
| JWT_SECRET           | JWT signing secret               | (required)            |
| JWT_ACCESS_DURATION  | Access token duration (seconds)  | 900                   |
| JWT_REFRESH_DURATION | Refresh token duration (seconds) | 604800                |
| JWT_DOMAIN           | Cookie domain                    | (empty for localhost) |
| ALLOWED_ORIGINS      | Comma-separated allowed origins  | http://localhost:3000 |
| DEV_MODE             | Development mode flag            | true                  |

## License

MIT
