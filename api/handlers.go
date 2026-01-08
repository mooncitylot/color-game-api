package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/color-game/api/models"
	"github.com/golang-jwt/jwt/v5"
)

// GET /
func (app *Application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Color Game API")
}

// POST /v1/auth/signup
func (app *Application) signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	userSignup := &models.UserSignupRequest{}
	errParsingJson := json.NewDecoder(r.Body).Decode(userSignup)
	if errParsingJson != nil {
		app.badJSONRequest(w, r, errParsingJson)
		return
	}

	// Create new user
	newUser, newUserErr := models.NewUser(*userSignup)
	if newUserErr != nil {
		app.internalServerError(w, r, newUserErr)
		return
	}

	// Check if user already exists
	_, getErr := app.UserRepo.GetUserByEmail(newUser.Email)
	if getErr == nil {
		app.userAlreadyExists(w, r, getErr)
		return
	}

	// Store new user in database
	storedUser, errStoringNewUser := app.UserRepo.Create(newUser)
	if errStoringNewUser != nil {
		app.internalServerError(w, r, errStoringNewUser)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(storedUser)
}

// POST /v1/auth/login
func (app *Application) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	// Parse credentials with device fingerprint
	creds := &models.Credentials{}
	if err := json.NewDecoder(r.Body).Decode(creds); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	// Validate device fingerprint is provided
	if creds.DeviceFingerprint == "" {
		app.badJSONRequest(w, r, errors.New("deviceFingerprint is required"))
		return
	}

	// Validate user credentials
	user, err := app.UserRepo.ValidateAndGetUser(*creds)
	if err != nil {
		app.invalidCredentials(w, r, err)
		return
	}

	if !user.Approved {
		app.invalidCredentials(w, r, errors.New("user not yet approved"))
		return
	}

	// Create/update device record
	deviceExpiry := time.Now().Add(time.Second * time.Duration(app.Config.JwtRefreshDuration))
	device := models.UserDevice{
		UserID:      user.UserID,
		Fingerprint: creds.DeviceFingerprint,
		DeviceData:  r.Header.Get("User-Agent"),
		Expiry:      deviceExpiry,
	}

	if err := app.UserRepo.CreateDevice(device); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Generate JWT access token
	accessExpiry := time.Now().Add(time.Second * time.Duration(app.Config.JwtAccessDuration))

	// Create access token claims
	accessClaims := models.JWTClaims{
		UserID:            user.UserID,
		Email:             user.Email,
		Kind:              user.Kind,
		DeviceFingerprint: creds.DeviceFingerprint,
		Scope:             "authentication",
		TokenType:         models.JWT.ACCESS_COOKIE_NAME,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(app.Config.JwtSecret))
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	sameSite := http.SameSiteStrictMode
	if app.Config.JwtDomain == "" {
		sameSite = http.SameSiteNoneMode
	}

	// Set access token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     models.JWT.ACCESS_COOKIE_NAME,
		Value:    accessTokenString,
		HttpOnly: true,
		Secure:   true,
		SameSite: sameSite,
		Path:     "/",
		Domain:   app.Config.JwtDomain,
		Expires:  accessExpiry,
	})

	// Generate refresh token
	refreshExpiry := deviceExpiry
	refreshClaims := models.JWTClaims{
		UserID:            user.UserID,
		Email:             user.Email,
		Kind:              user.Kind,
		DeviceFingerprint: creds.DeviceFingerprint,
		Scope:             "refresh",
		TokenType:         models.JWT.REFRESH_COOKIE_NAME,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(app.Config.JwtSecret))
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Set refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     models.JWT.REFRESH_COOKIE_NAME,
		Value:    refreshTokenString,
		HttpOnly: true,
		Secure:   true,
		SameSite: sameSite,
		Path:     "/",
		Domain:   app.Config.JwtDomain,
		Expires:  refreshExpiry,
	})

	w.WriteHeader(http.StatusOK)
}

// GET /v1/users/me - Get current authenticated user
func (app *Application) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// PUT /v1/users/me - Update current authenticated user
func (app *Application) updateCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		app.requirePutMethod(w, r, ErrPUT)
		return
	}

	// Get current user from token
	currentUser, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	// Parse update request
	updateReq := &models.UserUpdateRequest{}
	if err := json.NewDecoder(r.Body).Decode(updateReq); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	// Update user fields
	currentUser.Username = updateReq.Username
	currentUser.Email = updateReq.Email
	currentUser.UpdatedAt = time.Now()

	// Save to database
	updatedUser, updateErr := app.UserRepo.Update(currentUser)
	if updateErr != nil {
		app.internalServerError(w, r, updateErr)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedUser)
}

// GET /v1/users - Get all users
func (app *Application) getAllUsers(w http.ResponseWriter, r *http.Request) {
	users, retrieveErr := app.UserRepo.GetAllUsers()
	if retrieveErr != nil {
		app.internalServerError(w, r, retrieveErr)
		return
	}

	json.NewEncoder(w).Encode(users)
}
