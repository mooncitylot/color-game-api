package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/color-game/api/models"
	"github.com/golang-jwt/jwt/v5"
)

func handleCors(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Access-Control-Allow-Credentials, Access-Control-Allow-Origin, Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if r.Method == "OPTIONS" {
			return
		} else {
			h.ServeHTTP(w, r)
		}
	}
}

// getUserFromJWT attempts to get user from JWT access token cookie
func (app *Application) getUserFromJWT(r *http.Request) (models.User, error) {
	// Get JWT access token from cookie
	cookie, err := r.Cookie(models.JWT.ACCESS_COOKIE_NAME)
	if err != nil {
		return models.User{}, errors.New("no JWT cookie found")
	}

	// Parse and validate JWT token
	token, err := jwt.ParseWithClaims(cookie.Value, &models.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(app.Config.JwtSecret), nil
	})

	if err != nil || !token.Valid {
		return models.User{}, errors.New("invalid JWT token")
	}

	claims, ok := token.Claims.(*models.JWTClaims)
	if !ok || claims.Scope != "authentication" {
		return models.User{}, errors.New("invalid token claims")
	}

	// Verify device still exists and is valid
	device, err := app.UserRepo.GetDeviceByFingerprint(claims.UserID, claims.DeviceFingerprint)
	if err != nil {
		return models.User{}, errors.New("device not found")
	}

	if time.Now().After(device.Expiry) {
		return models.User{}, errors.New("device expired")
	}

	// Get user from database
	user, err := app.UserRepo.Get(claims.UserID)
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

func (app *Application) getUserFromToken(w http.ResponseWriter, r *http.Request) (models.User, error) {
	user, err := app.getUserFromJWT(r)
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}

// authenticate that the user exists
func (app *Application) authenticate(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := app.getUserFromToken(w, r)
		if err != nil {
			app.invalidAuthorization(w, r, err)
			return
		}

		// Check if user is approved
		if !user.Approved {
			app.invalidAuthorization(w, r, errors.New("user not approved"))
			return
		}

		h.ServeHTTP(w, r)
	}
}

// Verify user has Admin permissions
func (app *Application) verifyPermissions(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, errGettingUser := app.getUserFromToken(w, r)
		if errGettingUser != nil {
			app.internalServerError(w, r, errGettingUser)
			return
		}

		if user.Kind != models.Admin {
			app.invalidAuthorization(w, r, ErrInvalidPrivelege)
			return
		}

		h.ServeHTTP(w, r)
	}
}
