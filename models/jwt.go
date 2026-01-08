package models

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var JWT = struct {
	ACCESS_COOKIE_NAME  string
	REFRESH_COOKIE_NAME string
}{
	ACCESS_COOKIE_NAME:  "access_token",
	REFRESH_COOKIE_NAME: "refresh_token",
}

type JWTClaims struct {
	UserID            string `json:"userId"`
	Email             string `json:"email"`
	Kind              string `json:"kind"`
	DeviceFingerprint string `json:"deviceFingerprint"`
	Scope             string `json:"scope"`
	TokenType         string `json:"tokenType"`
	jwt.RegisteredClaims
}

type JWTRefreshResponse struct {
	Expiry  time.Time `json:"expiry"`
	Refresh string    `json:"refresh"`
}

func ValidateJWTToken(tokenString string, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
