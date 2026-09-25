package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockPermissionRepo struct {
	roles       []*RoleInfo
	permissions []*Permission
	rolePerms   map[string][]string // roleName -> []permName
}

func (m *mockPermissionRepo) ListPermissions(ctx context.Context) ([]*Permission, error) {
	return m.permissions, nil
}

func (m *mockPermissionRepo) ListRoles(ctx context.Context) ([]*RoleInfo, error) {
	return m.roles, nil
}

func (m *mockPermissionRepo) GetRoleByName(ctx context.Context, name string) (*RoleInfo, error) {
	for _, r := range m.roles {
		if r.Name == name {
			return r, nil
		}
	}
	return nil, nil
}

func (m *mockPermissionRepo) GetRolePermissions(ctx context.Context, roleName string) ([]string, error) {
	return m.rolePerms[roleName], nil
}

func (m *mockPermissionRepo) GetAllRolePermissions(ctx context.Context) (map[string][]string, error) {
	copied := make(map[string][]string)
	for k, v := range m.rolePerms {
		perms := make([]string, len(v))
		copy(perms, v)
		copied[k] = perms
	}
	return copied, nil
}

func (m *mockPermissionRepo) AssignPermissionsToRole(ctx context.Context, roleID string, permissionIDs []string) error {
	var targetRoleName string
	for _, r := range m.roles {
		if r.ID == roleID {
			targetRoleName = r.Name
			break
		}
	}

	var permNames []string
	for _, pid := range permissionIDs {
		for _, p := range m.permissions {
			if p.ID == pid {
				permNames = append(permNames, p.Name)
				break
			}
		}
	}

	m.rolePerms[targetRoleName] = permNames
	return nil
}

func setupMockService(t *testing.T) (PermissionService, *mockPermissionRepo) {
	repo := &mockPermissionRepo{
		roles: []*RoleInfo{
			{ID: "role-1", Name: "owner", DisplayName: "Owner"},
			{ID: "role-2", Name: "superadmin", DisplayName: "Superadmin"},
			{ID: "role-3", Name: "admin", DisplayName: "Admin"},
			{ID: "role-4", Name: "cashier", DisplayName: "Kasir"},
		},
		permissions: []*Permission{
			{ID: "p-1", Name: "inventory.products.view", Module: "inventory"},
			{ID: "p-2", Name: "inventory.products.create", Module: "inventory"},
			{ID: "p-3", Name: "inventory.stocks.adjust", Module: "inventory"},
		},
		rolePerms: map[string][]string{
			"admin":   {"inventory.products.view", "inventory.products.create"},
			"cashier": {"inventory.products.view"},
		},
	}

	svc, err := NewPermissionService(context.Background(), repo)
	if err != nil {
		t.Fatalf("gagal membuat permission service: %v", err)
	}

	return svc, repo
}

func TestHasPermission_BypassForOwnerAndSuperadmin(t *testing.T) {
	svc, _ := setupMockService(t)

	// Owner dan superadmin harus selalu true apapun permission-nya, bahkan permission acak
	if !svc.HasPermission("owner", "inventory.products.create") {
		t.Errorf("expected owner to have all permissions")
	}
	if !svc.HasPermission("owner", "random.undefined.permission") {
		t.Errorf("expected owner to bypass any permission")
	}
	if !svc.HasPermission("superadmin", "inventory.stocks.adjust") {
		t.Errorf("expected superadmin to have all permissions")
	}
}

func TestHasPermission_RoleEvaluations(t *testing.T) {
	svc, _ := setupMockService(t)

	// Admin punya products.view dan products.create, tapi tidak punya stocks.adjust
	if !svc.HasPermission("admin", "inventory.products.view") {
		t.Errorf("expected admin to have inventory.products.view")
	}
	if !svc.HasPermission("admin", "inventory.products.create") {
		t.Errorf("expected admin to have inventory.products.create")
	}
	if svc.HasPermission("admin", "inventory.stocks.adjust") {
		t.Errorf("expected admin NOT to have inventory.stocks.adjust")
	}

	// Cashier hanya punya products.view
	if !svc.HasPermission("cashier", "inventory.products.view") {
		t.Errorf("expected cashier to have inventory.products.view")
	}
	if svc.HasPermission("cashier", "inventory.products.create") {
		t.Errorf("expected cashier NOT to have inventory.products.create")
	}

	// Role tidak terdaftar harus false
	if svc.HasPermission("unknown_role", "inventory.products.view") {
		t.Errorf("expected unknown role to have false")
	}
}

func TestUpdateRolePermissions_SyncCache(t *testing.T) {
	svc, _ := setupMockService(t)
	ctx := context.Background()

	// Sebelum update, cashier tidak punya create
	if svc.HasPermission("cashier", "inventory.products.create") {
		t.Fatalf("cashier should not have create initially")
	}

	// Update hak akses cashier untuk menambahkan create
	newPerms := []string{"inventory.products.view", "inventory.products.create"}
	if err := svc.UpdateRolePermissions(ctx, "cashier", newPerms); err != nil {
		t.Fatalf("gagal update role permissions: %v", err)
	}

	// Setelah update, cache harus langsung sinkron O(1)
	if !svc.HasPermission("cashier", "inventory.products.create") {
		t.Errorf("expected cashier to have inventory.products.create after update")
	}

	// Coba ubah owner -> harus ditolak
	err := svc.UpdateRolePermissions(ctx, "owner", newPerms)
	if err != ErrCannotModifyOwner {
		t.Errorf("expected ErrCannotModifyOwner, got %v", err)
	}
}

func TestRequirePermission_Middleware(t *testing.T) {
	svc, _ := setupMockService(t)

	// Dummy handler yang mengembalikan 200 OK
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`ok`))
	})

	// 1. Tanpa claims di context -> 401 Unauthorized
	middleware := RequirePermission("inventory.products.view", svc)
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	middleware(okHandler).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 without claims, got %d", rec.Code)
	}

	// 2. Claims dengan role yang tidak memiliki izin -> 403 Forbidden
	reqForbidden := httptest.NewRequest("GET", "/test", nil)
	claimsCashier := &Claims{
		UserID: "user-cashier",
		Role:   "cashier",
	}
	ctxCashier := context.WithValue(reqForbidden.Context(), claimsKey, claimsCashier)
	reqForbidden = reqForbidden.WithContext(ctxCashier)

	middlewareStocks := RequirePermission("inventory.stocks.adjust", svc)
	recForbidden := httptest.NewRecorder()
	middlewareStocks(okHandler).ServeHTTP(recForbidden, reqForbidden)

	if recForbidden.Code != http.StatusForbidden {
		t.Errorf("expected status 403 for cashier on stocks.adjust, got %d", recForbidden.Code)
	}

	// 3. Claims dengan izin yang sesuai -> 200 OK
	reqAllowed := httptest.NewRequest("GET", "/test", nil)
	ctxAllowed := context.WithValue(reqAllowed.Context(), claimsKey, claimsCashier)
	reqAllowed = reqAllowed.WithContext(ctxAllowed)

	recAllowed := httptest.NewRecorder()
	middleware(okHandler).ServeHTTP(recAllowed, reqAllowed)

	if recAllowed.Code != http.StatusOK {
		t.Errorf("expected status 200 for cashier on products.view, got %d", recAllowed.Code)
	}
}
