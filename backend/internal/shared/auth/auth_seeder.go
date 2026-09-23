package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/erp-retail/backend/pkg/uid"
	"golang.org/x/crypto/bcrypt"
)

type DefaultUserSeed struct {
	Name     string
	Username string
	Email    string
	Password string
	Role     UserRole
}

// DefaultUsers adalah daftar akun standar bawaan sistem untuk keperluan initial setup dan testing.
var DefaultUsers = []DefaultUserSeed{
	{
		Name:     "Pemilik Bisnis (Owner)",
		Username: "owner",
		Email:    "owner@retail.com",
		Password: "password123",
		Role:     UserRoleOwner,
	},
	{
		Name:     "Administrator Sistem Utama",
		Username: "superadmin",
		Email:    "admin@retail.com",
		Password: "password123",
		Role:     UserRoleSuperadmin,
	},
	{
		Name:     "Admin Operasional Cabang",
		Username: "admin_pusat",
		Email:    "admin.pusat@retail.com",
		Password: "password123",
		Role:     UserRoleAdmin,
	},
	{
		Name:     "Kasir Shift Pagi",
		Username: "kasir_01",
		Email:    "kasir01@retail.com",
		Password: "password123",
		Role:     UserRoleCashier,
	},
	{
		Name:     "Admin Gudang & Penerimaan",
		Username: "gudang_01",
		Email:    "gudang01@retail.com",
		Password: "password123",
		Role:     UserRoleWarehouse,
	},
}

// SeedDefaultUsers menyuntikkan akun-akun staf bawaan ke tabel users secara idempoten.
func SeedDefaultUsers(ctx context.Context, db *sql.DB) (int, error) {
	now := time.Now().UTC()
	insertedCount := 0

	queryCheck := `SELECT id FROM users WHERE username = ? OR email = ? LIMIT 1`
	queryInsert := `
		INSERT INTO users (
			id, name, username, email, password_hash, role, location_id, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, NULL, TRUE, ?, ?)`

	for _, item := range DefaultUsers {
		var existingID string
		err := db.QueryRowContext(ctx, queryCheck, item.Username, item.Email).Scan(&existingID)
		if err == nil {
			// Akun sudah ada, skip
			continue
		}
		if err != sql.ErrNoRows {
			return insertedCount, fmt.Errorf("gagal cek user '%s': %w", item.Username, err)
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(item.Password), bcrypt.DefaultCost)
		if err != nil {
			return insertedCount, fmt.Errorf("gagal hash password user '%s': %w", item.Username, err)
		}

		id := uid.New()
		_, err = db.ExecContext(ctx, queryInsert,
			id, item.Name, item.Username, item.Email, string(hash), string(item.Role), now, now,
		)
		if err != nil {
			return insertedCount, fmt.Errorf("gagal insert user '%s': %w", item.Username, err)
		}
		insertedCount++
	}

	return insertedCount, nil
}
