package auth_test

import (
	"testing"

	"github.com/erp-retail/backend/internal/shared/auth"
)

func TestNewUser(t *testing.T) {
	loc := "loc-01"

	t.Run("sukses membuat user valid", func(t *testing.T) {
		u, err := auth.NewUser("usr-1", "Budi Santoso", "budi", "budi@retail.com", "password123", auth.UserRoleAdmin, &loc)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if u.ID != "usr-1" || u.Username != "budi" || u.Email != "budi@retail.com" {
			t.Errorf("mismatch user fields: %+v", u)
		}
		if u.Role != auth.UserRoleAdmin {
			t.Errorf("expected role admin, got %v", u.Role)
		}
		if !u.CheckPassword("password123") {
			t.Errorf("expected CheckPassword to return true")
		}
		if u.CheckPassword("wrongpassword") {
			t.Errorf("expected CheckPassword to return false for wrong password")
		}
		if !u.IsActive {
			t.Errorf("expected user to be active initially")
		}
	})

	t.Run("gagal jika email tidak valid", func(t *testing.T) {
		_, err := auth.NewUser("usr-2", "Budi", "budi2", "invalid-email", "password123", auth.UserRoleAdmin, nil)
		if err != auth.ErrInvalidEmail {
			t.Errorf("expected ErrInvalidEmail, got %v", err)
		}
	})

	t.Run("gagal jika password kurang dari 6 karakter", func(t *testing.T) {
		_, err := auth.NewUser("usr-3", "Budi", "budi3", "budi3@test.com", "12345", auth.UserRoleAdmin, nil)
		if err != auth.ErrInvalidPassword {
			t.Errorf("expected ErrInvalidPassword, got %v", err)
		}
	})

	t.Run("gagal jika role tidak valid", func(t *testing.T) {
		_, err := auth.NewUser("usr-4", "Budi", "budi4", "budi4@test.com", "password123", auth.UserRole("presiden"), nil)
		if err != auth.ErrInvalidRole {
			t.Errorf("expected ErrInvalidRole, got %v", err)
		}
	})

	t.Run("ganti password berhasil", func(t *testing.T) {
		u, err := auth.NewUser("usr-5", "Budi", "budi5", "budi5@test.com", "password123", auth.UserRoleCashier, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		err = u.SetPassword("newsecretpass")
		if err != nil {
			t.Fatalf("expected set password success, got %v", err)
		}
		if !u.CheckPassword("newsecretpass") {
			t.Errorf("expected new password to match")
		}
		if u.CheckPassword("password123") {
			t.Errorf("expected old password to no longer match")
		}
	})
}
