package auth

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type mysqlPermissionRepository struct {
	db *sql.DB
}

// NewPermissionRepository membuat instance baru PermissionRepository berbasis MySQL.
func NewPermissionRepository(db *sql.DB) PermissionRepository {
	return &mysqlPermissionRepository{db: db}
}

func (r *mysqlPermissionRepository) ListPermissions(ctx context.Context) ([]*Permission, error) {
	query := `
		SELECT id, name, module, COALESCE(description, ''), created_at 
		FROM shared_permissions 
		ORDER BY module ASC, name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gagal query daftar permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*Permission
	for rows.Next() {
		p := &Permission{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Module, &p.Description, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("gagal scan row permission: %w", err)
		}
		permissions = append(permissions, p)
	}

	return permissions, rows.Err()
}

func (r *mysqlPermissionRepository) ListRoles(ctx context.Context) ([]*RoleInfo, error) {
	query := `
		SELECT id, name, display_name, COALESCE(description, ''), is_system, created_at, updated_at 
		FROM shared_roles 
		ORDER BY is_system DESC, created_at ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gagal query daftar roles: %w", err)
	}
	defer rows.Close()

	var roles []*RoleInfo
	for rows.Next() {
		role := &RoleInfo{}
		if err := rows.Scan(&role.ID, &role.Name, &role.DisplayName, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("gagal scan row role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, rows.Err()
}

func (r *mysqlPermissionRepository) GetRoleByName(ctx context.Context, name string) (*RoleInfo, error) {
	query := `
		SELECT id, name, display_name, COALESCE(description, ''), is_system, created_at, updated_at 
		FROM shared_roles 
		WHERE name = ? LIMIT 1`

	role := &RoleInfo{}
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&role.ID, &role.Name, &role.DisplayName, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal query role by name: %w", err)
	}
	return role, nil
}

func (r *mysqlPermissionRepository) GetRolePermissions(ctx context.Context, roleName string) ([]string, error) {
	query := `
		SELECT p.name 
		FROM shared_permissions p
		JOIN shared_role_permissions rp ON p.id = rp.permission_id
		JOIN shared_roles r ON rp.role_id = r.id
		WHERE r.name = ?
		ORDER BY p.name ASC`

	rows, err := r.db.QueryContext(ctx, query, roleName)
	if err != nil {
		return nil, fmt.Errorf("gagal query role permissions: %w", err)
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var perm string
		if err := rows.Scan(&perm); err != nil {
			return nil, fmt.Errorf("gagal scan permission name: %w", err)
		}
		perms = append(perms, perm)
	}
	return perms, rows.Err()
}

func (r *mysqlPermissionRepository) GetAllRolePermissions(ctx context.Context) (map[string][]string, error) {
	query := `
		SELECT r.name, p.name 
		FROM shared_roles r
		JOIN shared_role_permissions rp ON r.id = rp.role_id
		JOIN shared_permissions p ON rp.permission_id = p.id
		ORDER BY r.name ASC, p.name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gagal query all role permissions: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var roleName, permName string
		if err := rows.Scan(&roleName, &permName); err != nil {
			return nil, fmt.Errorf("gagal scan role permission pair: %w", err)
		}
		result[roleName] = append(result[roleName], permName)
	}
	return result, rows.Err()
}

func (r *mysqlPermissionRepository) AssignPermissionsToRole(ctx context.Context, roleID string, permissionIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi assign permissions: %w", err)
	}
	defer tx.Rollback()

	// 1. Hapus semua permission lama milik role ini
	deleteQuery := `DELETE FROM shared_role_permissions WHERE role_id = ?`
	if _, err := tx.ExecContext(ctx, deleteQuery, roleID); err != nil {
		return fmt.Errorf("gagal menghapus permission lama: %w", err)
	}

	// 2. Suntikkan permission baru (jika ada)
	if len(permissionIDs) > 0 {
		var valueStrings []string
		var valueArgs []any
		for _, permID := range permissionIDs {
			valueStrings = append(valueStrings, "(?, ?)")
			valueArgs = append(valueArgs, roleID, permID)
		}

		insertQuery := fmt.Sprintf(
			"INSERT INTO shared_role_permissions (role_id, permission_id) VALUES %s",
			strings.Join(valueStrings, ", "),
		)

		if _, err := tx.ExecContext(ctx, insertQuery, valueArgs...); err != nil {
			return fmt.Errorf("gagal insert role permissions baru: %w", err)
		}
	}

	return tx.Commit()
}
