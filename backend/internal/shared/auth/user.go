package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// UserRole merepresentasikan hak akses dan peran staf di sistem ERP ritel.
type UserRole string

const (
	UserRoleOwner      UserRole = "owner"      // Pemilik bisnis, akses absolut
	UserRoleSuperadmin UserRole = "superadmin" // Administrator sistem, manajemen konfigurasi & user
	UserRoleAdmin      UserRole = "admin"      // Staf administrasi cabang
	UserRoleCashier    UserRole = "cashier"    // Kasir POS
	UserRoleWarehouse  UserRole = "warehouse"  // Staf/Admin gudang operasional
	UserRoleCustomer   UserRole = "customer"   // Pelanggan publik e-commerce
)

// IsValid memvalidasi apakah peran sesuai dengan hierarki yang diizinkan.
func (r UserRole) IsValid() bool {
	switch r {
	case UserRoleOwner, UserRoleSuperadmin, UserRoleAdmin, UserRoleCashier, UserRoleWarehouse, UserRoleCustomer:
		return true
	default:
		return false
	}
}

var (
	ErrInvalidUserID    = errors.New("ID user tidak valid")
	ErrInvalidName      = errors.New("nama user tidak boleh kosong")
	ErrInvalidUsername  = errors.New("username minimal 3 karakter tanpa spasi")
	ErrInvalidEmail     = errors.New("format alamat email tidak valid")
	ErrInvalidPassword  = errors.New("kata sandi minimal 6 karakter")
	ErrInvalidRole      = errors.New("peran user tidak valid")
	ErrUserNotFound     = errors.New("pengguna tidak ditemukan")
	ErrUserInactive     = errors.New("akun pengguna sedang dinonaktifkan")
	ErrUsernameExists   = errors.New("username sudah digunakan oleh akun lain")
	ErrEmailExists      = errors.New("email sudah terdaftar di sistem")
	ErrWrongPassword    = errors.New("kata sandi tidak sesuai")
	ErrPasswordRequired = errors.New("kata sandi wajib diisi")
)

// User merepresentasikan entitas akun staf/pengguna dalam Shared Context.
type User struct {
	ID           string    // Primary Key (UUIDv7)
	Name         string    // Nama lengkap pengguna
	Username     string    // Username unik untuk login
	Email        string    // Email unik
	PasswordHash string    // Hash kata sandi (bcrypt)
	Role         UserRole  // Peran (owner, superadmin, admin, cashier, warehouse)
	LocationID   *string   // Afiliasi cabang/gudang bertugas (opsional)
	IsActive     bool      // Status akun aktif
	CreatedAt    time.Time // Waktu registrasi akun
	UpdatedAt    time.Time // Waktu perubahan terakhir
}

// NewUser adalah factory function untuk membuat entitas User baru dengan enkripsi password bcrypt.
func NewUser(id, name, username, email, plainPassword string, role UserRole, locationID *string) (*User, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidUserID
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, ErrInvalidName
	}
	trimmedUsername := strings.ToLower(strings.TrimSpace(username))
	if len(trimmedUsername) < 3 || strings.Contains(trimmedUsername, " ") {
		return nil, ErrInvalidUsername
	}
	trimmedEmail := strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(trimmedEmail); err != nil {
		return nil, ErrInvalidEmail
	}
	if len(plainPassword) < 6 {
		return nil, ErrInvalidPassword
	}
	if !role.IsValid() {
		return nil, ErrInvalidRole
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &User{
		ID:           id,
		Name:         trimmedName,
		Username:     trimmedUsername,
		Email:        trimmedEmail,
		PasswordHash: string(hash),
		Role:         role,
		LocationID:   locationID,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// CheckPassword memvalidasi kecocokan kata sandi plain teks terhadap hash bcrypt.
func (u *User) CheckPassword(plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(plainPassword))
	return err == nil
}

// SetPassword mengganti kata sandi dengan hash bcrypt baru.
func (u *User) SetPassword(newPlainPassword string) error {
	if len(newPlainPassword) < 6 {
		return ErrInvalidPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPlainPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	u.UpdatedAt = time.Now().UTC()
	return nil
}

// Deactivate menonaktifkan akun staf.
func (u *User) Deactivate() {
	u.IsActive = false
	u.UpdatedAt = time.Now().UTC()
}

// Activate mengaktifkan kembali akun staf.
func (u *User) Activate() {
	u.IsActive = true
	u.UpdatedAt = time.Now().UTC()
}

// UserRepository adalah interface persistensi database untuk entitas User.
type UserRepository interface {
	Save(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByUsernameOrEmail(ctx context.Context, identifier string) (*User, error)
	List(ctx context.Context, role *UserRole, locationID *string, isActiveOnly bool) ([]*User, error)
}
