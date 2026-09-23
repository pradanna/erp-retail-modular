package auth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/erp-retail/backend/internal/shared/auth"
)

func TestGenerateToken(t *testing.T) {
	secret := "super-secret-jwt-key-retail-erp-development-12345"
	token, err := auth.GenerateToken(secret, "usr_admin_01", "superadmin", "loc_pusat_01", 365*24*time.Hour)
	if err != nil {
		t.Fatalf("gagal generate token: %v", err)
	}
	fmt.Println("DEV_TOKEN:", token)
}
