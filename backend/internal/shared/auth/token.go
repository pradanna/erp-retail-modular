package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateToken membuat token JWT baru yang ditandatangani dengan HMAC-SHA256.
// Fungsi ini berguna untuk autentikasi user saat login maupun untuk keperluan testing.
func GenerateToken(jwtSecret string, userID, username, name, role, locationID string, duration time.Duration) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Name:     name,
		Role:     role,
		Location: locationID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}
