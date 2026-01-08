package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	Player = "Player"
	Admin  = "Admin"
)

type Credentials struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	DeviceFingerprint string `json:"deviceFingerprint,omitempty"`
}

type UserSignupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserUpdateRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

type User struct {
	UserID         string    `json:"userId" db:"user_id"`
	Username       string    `json:"username" db:"username"`
	Email          string    `json:"email" db:"email"`
	HashedPassword string    `json:"-" db:"password_hash"`
	Kind           string    `json:"kind" db:"kind"`
	Approved       bool      `json:"approved" db:"approved"`
	Points         int       `json:"points" db:"points"`
	Level          int       `json:"level" db:"level"`
	Credits        int       `json:"credits" db:"credits"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

type UserDevice struct {
	ID          string    `json:"id" db:"id"`
	UserID      string    `json:"userId" db:"user_id"`
	Fingerprint string    `json:"fingerprint" db:"fingerprint"`
	DeviceData  string    `json:"deviceData" db:"device_data"`
	Expiry      time.Time `json:"expiry" db:"expiry"`
}

func (user User) Serialize() ([]byte, error) {
	jsonUser, err := json.Marshal(user)
	if err != nil {
		return []byte{}, fmt.Errorf("error parsing json for User %v", err)
	}
	return []byte(jsonUser), nil
}

func (user User) GenerateKey() string {
	return uuid.New().String()
}

func NewUser(userSignup UserSignupRequest) (User, error) {
	var user User
	userkey := user.GenerateKey()
	hashedPassword, hashErr := user.GenerateHash(userSignup.Password)
	if hashErr != nil {
		return User{}, fmt.Errorf("error hashing password %v", hashErr)
	}
	user = User{
		UserID:         userkey,
		Username:       userSignup.Username,
		Email:          userSignup.Email,
		HashedPassword: hashedPassword,
		Kind:           Player,
		Approved:       true, // Auto-approve for simplicity
		Points:         0,
		Level:          1,
		Credits:        0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	return user, nil
}

func (user User) GenerateHash(password string) (string, error) {
	hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(password), 8)
	if hashErr != nil {
		return "", fmt.Errorf("error hashing password %v", hashErr)
	}

	return string(hashedPassword), nil
}
