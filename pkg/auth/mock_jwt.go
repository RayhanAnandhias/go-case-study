package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateMockJWT generates a mock JSON Web Token for testing purposes.
func GenerateMockJWT(userID string, secret string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
