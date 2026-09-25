package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/erp-retail/backend/pkg/uid"
)

// UserResolver menyediakan kapabilitas penyelesaian nama pengguna (ID/username -> Full Name) untuk modul lain.
type UserResolver interface {
	ResolveUserName(ctx context.Context, idOrUsername string) string
	ResolveUserNames(ctx context.Context, idsOrUsernames []string) map[string]string
}

// Service menangani use case logika bisnis autentikasi, step-up auth, dan manajemen staf.
type Service struct {
	repo      UserRepository
	jwtSecret string
	tokenTTL  time.Duration
	nameCache sync.Map
}

// NewService membuat instance baru Service autentikasi.
func NewService(repo UserRepository, jwtSecret string, tokenTTL time.Duration) *Service {
	if tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour // Default 24 jam
	}
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		tokenTTL:  tokenTTL,
	}
}

// Login memvalidasi kredensial pengguna dan mengembalikan JWT token bersama profil pengguna.
func (s *Service) Login(ctx context.Context, identifier, password string) (string, *User, error) {
	if identifier == "" || password == "" {
		return "", nil, errors.New("username/email dan kata sandi wajib diisi")
	}

	user, err := s.repo.FindByUsernameOrEmail(ctx, identifier)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", nil, ErrWrongPassword // Samarkan pesan error demi keamanan (mencegah user enumeration)
		}
		return "", nil, err
	}

	if !user.IsActive {
		return "", nil, ErrUserInactive
	}

	if !user.CheckPassword(password) {
		return "", nil, ErrWrongPassword
	}

	var loc string
	if user.LocationID != nil {
		loc = *user.LocationID
	}

	token, err := GenerateToken(s.jwtSecret, user.ID, user.Username, user.Name, string(user.Role), loc, s.tokenTTL)
	if err != nil {
		return "", nil, fmt.Errorf("gagal membuat token autentikasi: %w", err)
	}

	return token, user, nil
}

// VerifyPassword memvalidasi kata sandi pengguna untuk kebutuhan Step-up Authentication.
// Digunakan sebelum mengeksekusi aksi berisiko tinggi (misal: approval PO, mutasi stok, dsb).
func (s *Service) VerifyPassword(ctx context.Context, userID, password string) (bool, error) {
	if password == "" {
		return false, ErrPasswordRequired
	}

	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return false, err
	}

	if !user.IsActive {
		return false, ErrUserInactive
	}

	if !user.CheckPassword(password) {
		return false, ErrWrongPassword
	}

	return true, nil
}

// GetProfile mengambil profil pengguna berdasarkan ID dari JWT claims.
func (s *Service) GetProfile(ctx context.Context, userID string) (*User, error) {
	return s.repo.FindByID(ctx, userID)
}

// CreateUser mendaftarkan staf/pengguna baru oleh superadmin/owner.
func (s *Service) CreateUser(ctx context.Context, name, username, email, plainPassword string, role UserRole, locationID *string) (*User, error) {
	id := uid.New()
	user, err := NewUser(id, name, username, email, plainPassword, role, locationID)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// ListUsers mengambil daftar seluruh staf pengguna.
func (s *Service) ListUsers(ctx context.Context, role *UserRole, locationID *string, isActiveOnly bool) ([]*User, error) {
	return s.repo.List(ctx, role, locationID, isActiveOnly)
}

// SetUserStatus mengubah status aktif/nonaktif akun staf.
func (s *Service) SetUserStatus(ctx context.Context, userID string, isActive bool) (*User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if isActive {
		user.Activate()
	} else {
		user.Deactivate()
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// ChangePassword memproses penggantian kata sandi mandiri pengguna.
func (s *Service) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if !user.CheckPassword(oldPassword) {
		return ErrWrongPassword
	}

	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	return s.repo.Update(ctx, user)
}

// ResolveUserName menyelesaikan nama lengkap pengguna dari idOrUsername.
// Jika idOrUsername cocok dengan ID atau Username user di database, nama lengkap akan dikembalikan.
func (s *Service) ResolveUserName(ctx context.Context, idOrUsername string) string {
	clean := strings.TrimSpace(idOrUsername)
	if clean == "" {
		return ""
	}

	// 1. Cek cache memori (O(1) look-up)
	if val, ok := s.nameCache.Load(clean); ok {
		if name, ok := val.(string); ok && name != "" {
			return name
		}
	}

	// 2. Cek alias/fallback seeder awal jika ada
	if clean == "usr_admin_01" {
		s.nameCache.Store(clean, "Admin Operasional")
		return "Admin Operasional"
	}

	// 3. Coba cari berdasarkan ID (UUID)
	u, err := s.repo.FindByID(ctx, clean)
	if err == nil && u != nil && u.Name != "" {
		s.nameCache.Store(u.ID, u.Name)
		s.nameCache.Store(u.Username, u.Name)
		return u.Name
	}

	// 4. Coba cari berdasarkan Username
	u, err = s.repo.FindByUsernameOrEmail(ctx, clean)
	if err == nil && u != nil && u.Name != "" {
		s.nameCache.Store(u.ID, u.Name)
		s.nameCache.Store(u.Username, u.Name)
		return u.Name
	}

	// 5. Fallback: simpan dan kembalikan identitas aslinya
	s.nameCache.Store(clean, clean)
	return clean
}

// ResolveUserNames menyelesaikan sekumpulan ID/username secara batch.
func (s *Service) ResolveUserNames(ctx context.Context, idsOrUsernames []string) map[string]string {
	result := make(map[string]string, len(idsOrUsernames))
	for _, id := range idsOrUsernames {
		clean := strings.TrimSpace(id)
		if clean == "" {
			continue
		}
		if _, exists := result[clean]; !exists {
			result[clean] = s.ResolveUserName(ctx, clean)
		}
	}
	return result
}
