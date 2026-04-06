package datastore

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"

	"github.com/color-game/api/models"
)

type UserRepository interface {
	Create(user models.User) (models.User, error)
	Get(userID string) (models.User, error)
	GetUserEffect(userID string) (*string, error)
	RemoveUserEffect(userID string) error
	GetUserByEmail(email string) (models.User, error)
	GetUserByUsername(username string) (models.User, error)
	DeleteUserByID(userID string) error
	Update(user models.User) (models.User, error)
	ValidateAndGetUser(userLogin models.Credentials) (models.User, error)
	GetAllUsers() ([]models.User, error)

	// Device management
	CreateDevice(device models.UserDevice) error
	GetDeviceByFingerprint(userID string, fingerprint string) (models.UserDevice, error)
	DeleteDevice(deviceID string) error
}

func NewUserDatabase(db *sql.DB) (UserDatabase, error) {
	var UserDatabase UserDatabase
	UserDatabase.database = db
	return UserDatabase, nil
}

type NoRowsError struct {
	NoRows bool
	Err    error
}

func (nr NoRowsError) Error() string {
	return fmt.Sprintf("%v: no rows returned for scan: %v", nr.NoRows, nr.Err)
}

func userEffectFromNull(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

func userEffectArg(effect *string) interface{} {
	if effect == nil {
		return nil
	}
	return *effect
}

type UserDatabase struct {
	database *sql.DB
}

func (pgdb UserDatabase) Create(user models.User) (models.User, error) {
	db := pgdb.database

	_, insertErr := db.Exec(`
		INSERT INTO users (
			user_id, 
			username,
			email, 
			password_hash, 
			kind,
			approved,
			points,
			level,
			credits,
			user_effect,
			created_at,
			updated_at
		) VALUES (
			$1, 
			$2, 
			$3, 
			$4, 
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12
		)`,
		user.UserID,
		user.Username,
		user.Email,
		user.HashedPassword,
		user.Kind,
		user.Approved,
		user.Points,
		user.Level,
		user.Credits,
		userEffectArg(user.UserEffect),
		user.CreatedAt,
		user.UpdatedAt,
	)

	if insertErr != nil {
		return user, insertErr
	}

	return user, nil
}

func (pgdb UserDatabase) Get(userID string) (models.User, error) {
	db := pgdb.database

	sqlStatement := `
	SELECT 
		user_id, 
		username,
		email, 
		password_hash, 
		kind,
		approved,
		points,
		level,
		credits,
		user_effect,
		created_at,
		updated_at
	FROM users 
	WHERE user_id=$1;`

	row := db.QueryRow(sqlStatement, userID)

	var user models.User
	var userEffect sql.NullString
	scanErr := row.Scan(
		&user.UserID,
		&user.Username,
		&user.Email,
		&user.HashedPassword,
		&user.Kind,
		&user.Approved,
		&user.Points,
		&user.Level,
		&user.Credits,
		&userEffect,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	switch scanErr {
	case sql.ErrNoRows:
		return models.User{}, NoRowsError{true, scanErr}
	case nil:
		user.UserEffect = userEffectFromNull(userEffect)
		return user, nil
	default:
		return models.User{}, scanErr
	}
}

func (pgdb UserDatabase) GetUserEffect(userID string) (*string, error) {
	db := pgdb.database
	row := db.QueryRow(`SELECT user_effect FROM users WHERE user_id = $1`, userID)
	var userEffect sql.NullString
	if scanErr := row.Scan(&userEffect); scanErr != nil {
		if scanErr == sql.ErrNoRows {
			return nil, NoRowsError{true, scanErr}
		}
		return nil, scanErr
	}
	return userEffectFromNull(userEffect), nil
}

func (pgdb UserDatabase) RemoveUserEffect(userID string) error {
	db := pgdb.database
	res, err := db.Exec(`
		UPDATE users
		SET user_effect = NULL, updated_at = $2
		WHERE user_id = $1`,
		userID, time.Now())
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return NoRowsError{true, sql.ErrNoRows}
	}
	return nil
}

func (pgdb UserDatabase) GetAllUsers() ([]models.User, error) {
	db := pgdb.database
	sqlStatement := `
	SELECT 
		user_id, 
		username,
		email, 
		password_hash, 
		kind,
		approved,
		points,
		level,
		credits,
		user_effect,
		created_at,
		updated_at
	FROM users
	ORDER BY created_at DESC`

	rows, pgErr := db.Query(sqlStatement)
	if pgErr != nil {
		return []models.User{}, pgErr
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var userEffect sql.NullString
		scanErr := rows.Scan(
			&user.UserID,
			&user.Username,
			&user.Email,
			&user.HashedPassword,
			&user.Kind,
			&user.Approved,
			&user.Points,
			&user.Level,
			&user.Credits,
			&userEffect,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if scanErr != nil {
			return []models.User{}, scanErr
		}
		user.UserEffect = userEffectFromNull(userEffect)
		users = append(users, user)
	}
	if rows.Err() != nil {
		return []models.User{}, rows.Err()
	}

	return users, nil
}

func (pgdb UserDatabase) GetUserByEmail(email string) (models.User, error) {
	db := pgdb.database

	sqlStatement := `
		SELECT
			user_id, 
			username,
			email, 
			password_hash, 
			kind,
			approved,
			points,
			level,
			credits,
			user_effect,
			created_at,
			updated_at
		FROM users
		WHERE email = $1`

	row := db.QueryRow(sqlStatement, email)

	var user models.User
	var userEffect sql.NullString
	scanErr := row.Scan(
		&user.UserID,
		&user.Username,
		&user.Email,
		&user.HashedPassword,
		&user.Kind,
		&user.Approved,
		&user.Points,
		&user.Level,
		&user.Credits,
		&userEffect,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	switch scanErr {
	case sql.ErrNoRows:
		return models.User{}, NoRowsError{true, scanErr}
	case nil:
		user.UserEffect = userEffectFromNull(userEffect)
		return user, nil
	default:
		return models.User{}, scanErr
	}
}

func (pgdb UserDatabase) GetUserByUsername(username string) (models.User, error) {
	db := pgdb.database

	sqlStatement := `
		SELECT
			user_id, 
			username,
			email, 
			password_hash, 
			kind,
			approved,
			points,
			level,
			credits,
			user_effect,
			created_at,
			updated_at
		FROM users
		WHERE username = $1`

	row := db.QueryRow(sqlStatement, username)

	var user models.User
	var userEffect sql.NullString
	scanErr := row.Scan(
		&user.UserID,
		&user.Username,
		&user.Email,
		&user.HashedPassword,
		&user.Kind,
		&user.Approved,
		&user.Points,
		&user.Level,
		&user.Credits,
		&userEffect,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	switch scanErr {
	case sql.ErrNoRows:
		return models.User{}, NoRowsError{true, scanErr}
	case nil:
		user.UserEffect = userEffectFromNull(userEffect)
		return user, nil
	default:
		return models.User{}, scanErr
	}
}

func (pgdb UserDatabase) DeleteUserByID(userID string) error {
	db := pgdb.database
	_, delErr := db.Exec("DELETE FROM users WHERE user_id=$1", userID)
	if delErr != nil {
		return fmt.Errorf("delete failed: %v", delErr)
	}

	return nil
}

func (pgdb UserDatabase) Update(user models.User) (models.User, error) {
	db := pgdb.database

	sqlStatement := `
	UPDATE users
	SET 
		username = $2,
		email = $3,
		kind = $4,
		points = $5,
		level = $6,
		credits = $7,
		user_effect = $8,
		updated_at = $9
	WHERE user_id = $1
	`
	_, insertErr := db.Exec(sqlStatement,
		user.UserID,
		user.Username,
		user.Email,
		user.Kind,
		user.Points,
		user.Level,
		user.Credits,
		userEffectArg(user.UserEffect),
		time.Now(),
	)

	if insertErr != nil {
		return models.User{}, fmt.Errorf("error updating user %v", insertErr)
	}
	return user, nil
}

func (pgdb UserDatabase) ValidateAndGetUser(credentials models.Credentials) (models.User, error) {
	db := pgdb.database
	sqlStatement := `
	SELECT
		user_id, 
		username,
		email, 
		password_hash, 
		kind,
		approved,
		points,
		level,
		credits,
		user_effect,
		created_at,
		updated_at
	FROM users
	WHERE email = $1;
	`
	var user models.User
	var passwordHash string
	var userEffect sql.NullString

	row := db.QueryRow(sqlStatement, credentials.Email)
	scanErr := row.Scan(
		&user.UserID,
		&user.Username,
		&user.Email,
		&passwordHash,
		&user.Kind,
		&user.Approved,
		&user.Points,
		&user.Level,
		&user.Credits,
		&userEffect,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if scanErr != nil {
		return models.User{}, fmt.Errorf("error in row scan %v", scanErr)
	}
	user.UserEffect = userEffectFromNull(userEffect)

	bcryptErr := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(credentials.Password))
	if bcryptErr != nil {
		return models.User{}, fmt.Errorf("error in compare of hash %v", bcryptErr)
	}
	return user, nil
}

// CreateDevice creates a new device record for a user
func (pgdb UserDatabase) CreateDevice(device models.UserDevice) error {
	db := pgdb.database

	sqlStatement := `
		INSERT INTO user_devices (user_id, device_data, fingerprint, expiry)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (fingerprint, user_id) 
		DO UPDATE SET device_data = $2, expiry = $4`

	_, err := db.Exec(sqlStatement, device.UserID, device.DeviceData, device.Fingerprint, device.Expiry)
	return err
}

// GetDeviceByFingerprint retrieves a device by user ID and fingerprint
func (pgdb UserDatabase) GetDeviceByFingerprint(userID string, fingerprint string) (models.UserDevice, error) {
	db := pgdb.database
	var device models.UserDevice

	sqlStatement := `
		SELECT id, user_id, device_data, fingerprint, expiry
		FROM user_devices
		WHERE user_id = $1 AND fingerprint = $2`

	row := db.QueryRow(sqlStatement, userID, fingerprint)
	err := row.Scan(&device.ID, &device.UserID, &device.DeviceData, &device.Fingerprint, &device.Expiry)

	if err != nil {
		return models.UserDevice{}, err
	}

	return device, nil
}

// DeleteDevice removes a device by ID
func (pgdb UserDatabase) DeleteDevice(deviceID string) error {
	db := pgdb.database

	sqlStatement := `DELETE FROM user_devices WHERE id = $1`
	_, err := db.Exec(sqlStatement, deviceID)

	return err
}
