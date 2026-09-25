package auth

import (
	"context"
	"fmt"
	"sync"
)

type permissionServiceImpl struct {
	repo  PermissionRepository
	mu    sync.RWMutex
	cache map[string]map[string]bool // key: role_name, key: permission_name -> true
}

// NewPermissionService membuat service otorisasi PBAC dan langsung menghangatkan (warm-up) in-memory cache.
func NewPermissionService(ctx context.Context, repo PermissionRepository) (PermissionService, error) {
	service := &permissionServiceImpl{
		repo:  repo,
		cache: make(map[string]map[string]bool),
	}

	if err := service.ReloadCache(ctx); err != nil {
		return nil, fmt.Errorf("gagal inisialisasi permission cache: %w", err)
	}

	return service, nil
}

func (s *permissionServiceImpl) HasPermission(role string, permission string) bool {
	// Aturan Inti: Owner dan Superadmin selalu memiliki akses penuh (wildcard bypass)
	if role == string(UserRoleOwner) || role == string(UserRoleSuperadmin) {
		return true
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	rolePerms, exists := s.cache[role]
	if !exists {
		return false
	}

	return rolePerms[permission]
}

func (s *permissionServiceImpl) GetRolePermissions(ctx context.Context, role string) ([]string, error) {
	return s.repo.GetRolePermissions(ctx, role)
}

func (s *permissionServiceImpl) GetMatrix(ctx context.Context) (*RolePermissionsMatrix, error) {
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil roles: %w", err)
	}

	permissions, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil permissions: %w", err)
	}

	allRolePerms, err := s.repo.GetAllRolePermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil relasi role permissions: %w", err)
	}

	// Pastikan semua role terdaftar di matriks (meskipun belum punya izin)
	matrix := make(map[string][]string)
	for _, r := range roles {
		if perms, ok := allRolePerms[r.Name]; ok {
			matrix[r.Name] = perms
		} else {
			matrix[r.Name] = []string{}
		}
	}

	return &RolePermissionsMatrix{
		Roles:       roles,
		Permissions: permissions,
		Matrix:      matrix,
	}, nil
}

func (s *permissionServiceImpl) UpdateRolePermissions(ctx context.Context, roleName string, permissionNames []string) error {
	// Proteksi integritas: owner dan superadmin tidak boleh diubah
	if roleName == string(UserRoleOwner) || roleName == string(UserRoleSuperadmin) {
		return ErrCannotModifyOwner
	}

	role, err := s.repo.GetRoleByName(ctx, roleName)
	if err != nil {
		return fmt.Errorf("gagal mencari role: %w", err)
	}
	if role == nil {
		return ErrRoleNotFound
	}

	// Ambil semua permission untuk mencocokkan name -> id
	allPerms, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return fmt.Errorf("gagal memuat daftar permission: %w", err)
	}

	nameToID := make(map[string]string)
	for _, p := range allPerms {
		nameToID[p.Name] = p.ID
	}

	var permIDs []string
	for _, name := range permissionNames {
		id, exists := nameToID[name]
		if !exists {
			return fmt.Errorf("%w: %s", ErrPermissionNotFound, name)
		}
		permIDs = append(permIDs, id)
	}

	// 1. Simpan perubahan ke database MySQL
	if err := s.repo.AssignPermissionsToRole(ctx, role.ID, permIDs); err != nil {
		return fmt.Errorf("gagal assign permissions ke role: %w", err)
	}

	// 2. Perbarui in-memory cache seketika
	return s.ReloadCache(ctx)
}

func (s *permissionServiceImpl) ReloadCache(ctx context.Context) error {
	allRolePerms, err := s.repo.GetAllRolePermissions(ctx)
	if err != nil {
		return fmt.Errorf("gagal membaca all role permissions: %w", err)
	}

	newCache := make(map[string]map[string]bool)
	for role, perms := range allRolePerms {
		permMap := make(map[string]bool)
		for _, p := range perms {
			permMap[p] = true
		}
		newCache[role] = permMap
	}

	s.mu.Lock()
	s.cache = newCache
	s.mu.Unlock()

	return nil
}
