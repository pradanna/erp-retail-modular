package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey adalah tipe private untuk kunci nilai di context.Context.
// Menggunakan tipe custom (bukan string biasa) mencegah collision dengan package lain
// yang mungkin juga menyimpan nilai di context dengan key yang sama.
type contextKey string

const claimsKey contextKey = "claims"

// Claims adalah payload JWT yang akan kita simpan di setiap token.
// Embed jwt.RegisteredClaims untuk mendapatkan field standar (exp, iat, sub, dst).
type Claims struct {
	UserID   string `json:"user_id"`
	Role     string `json:"role"`     // "admin" | "superadmin" | "owner"
	Location string `json:"location"` // location_id default user ini (bisa kosong)
	jwt.RegisteredClaims
}

// Middleware adalah HTTP middleware yang memvalidasi JWT di header Authorization.
// Middleware di net/http standar Go adalah fungsi yang menerima http.Handler
// dan mengembalikan http.Handler — polanya disebut "handler wrapping" atau "decorator".
//
// Alur: Request → Middleware (validasi token) → Handler asli (jika token valid)
func Middleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Ambil token dari header: "Authorization: Bearer <token>"
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"error":"token tidak ditemukan"}`, http.StatusUnauthorized)
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			// Parse dan validasi token menggunakan secret key
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				// Pastikan algoritma yang digunakan adalah HMAC (HS256/384/512)
				// Ini mencegah "algorithm confusion attack" (none algorithm attack)
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("algoritma signing tidak valid: %v", t.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, `{"error":"token tidak valid atau sudah expired"}`, http.StatusUnauthorized)
				return
			}

			// Simpan claims ke context agar handler selanjutnya bisa mengaksesnya
			// tanpa perlu parse ulang token
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims mengambil JWT claims dari context request.
// Mengembalikan nil jika request tidak melewati middleware JWT.
func GetClaims(r *http.Request) *Claims {
	claims, _ := r.Context().Value(claimsKey).(*Claims)
	return claims
}
