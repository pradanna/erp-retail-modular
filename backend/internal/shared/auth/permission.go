package auth

import (
	"context"
	"errors"
	"time"
)

var (
	ErrRoleNotFound       = errors.New("peran tidak ditemukan")
	ErrPermissionNotFound = errors.New("izin tidak ditemukan")
	ErrCannotModifyOwner  = errors.New("hak akses owner dan superadmin tidak dapat diubah")
)

// Permission merepresentasikan entitas izin kapabilitas granular di sistem ERP.
type Permission struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`        // Format: module.resource.action (contoh: inventory.products.create)
	Module      string    `json:"module"`      // inventory, purchasing, sales, finance, shared, dst.
	Description string    `json:"description"` // Penjelasan human-readable
	CreatedAt   time.Time `json:"created_at"`
}

// RoleInfo merepresentasikan entitas metadata peran yang tersimpan di database.
type RoleInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`         // owner, superadmin, admin, cashier, warehouse, customer
	DisplayName string    `json:"display_name"` // Owner, Super Admin, Admin Cabang, Kasir, Gudang, Customer
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"` // Role bawaan sistem yang tidak boleh dihapus
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// RolePermissionsMatrix adalah DTO struktur data untuk disajikan di antarmuka matriks checkbox Backoffice.
type RolePermissionsMatrix struct {
	Roles       []*RoleInfo         `json:"roles"`
	Permissions []*Permission       `json:"permissions"`
	Matrix      map[string][]string `json:"matrix"` // Key: role_name, Value: []permission_name
}

// PermissionRepository adalah kontrak antarmuka akses data untuk tabel roles & permissions.
type PermissionRepository interface {
	ListPermissions(ctx context.Context) ([]*Permission, error)
	ListRoles(ctx context.Context) ([]*RoleInfo, error)
	GetRoleByName(ctx context.Context, name string) (*RoleInfo, error)
	GetRolePermissions(ctx context.Context, roleName string) ([]string, error)
	GetAllRolePermissions(ctx context.Context) (map[string][]string, error)
	AssignPermissionsToRole(ctx context.Context, roleID string, permissionIDs []string) error
}

// PermissionService adalah kontrak layanan evaluasi hak akses berkinerja tinggi (in-memory cached).
type PermissionService interface {
	// HasPermission mengecek apakah role tertentu memiliki izin spesifik.
	// Wajib mengeksekusi O(1) in-memory lookup tanpa query SQL ke database.
	HasPermission(role string, permission string) bool

	// GetRolePermissions mengambil daftar permission aktif untuk satu role.
	GetRolePermissions(ctx context.Context, role string) ([]string, error)

	// GetMatrix mengambil seluruh role, permission, dan status centang untuk UI Backoffice.
	GetMatrix(ctx context.Context) (*RolePermissionsMatrix, error)

	// UpdateRolePermissions memperbarui izin role tertentu dan langsung menyinkronkan in-memory cache.
	UpdateRolePermissions(ctx context.Context, role string, permissionNames []string) error

	// ReloadCache membaca ulang seluruh pemetaan role-permission dari database ke RAM.
	ReloadCache(ctx context.Context) error
}
