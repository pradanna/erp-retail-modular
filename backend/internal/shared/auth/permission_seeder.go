package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/erp-retail/backend/pkg/uid"
)

// SystemRoleSeed mendefinisikan metadata peran standar sistem.
type SystemRoleSeed struct {
	Name        string
	DisplayName string
	Description string
	IsSystem    bool
}

// SystemRoles adalah daftar seluruh peran bawaan sistem ERP.
var SystemRoles = []SystemRoleSeed{
	{
		Name:        string(UserRoleOwner),
		DisplayName: "Owner (Pemilik)",
		Description: "Pemilik bisnis dengan akses bypass penuh ke seluruh modul sistem",
		IsSystem:    true,
	},
	{
		Name:        string(UserRoleSuperadmin),
		DisplayName: "Super Administrator",
		Description: "Pengelola teknis sistem dengan hak akses penuh ke seluruh modul",
		IsSystem:    true,
	},
	{
		Name:        string(UserRoleAdmin),
		DisplayName: "Admin Operasional",
		Description: "Pengelola operasional toko dan inventaris harian",
		IsSystem:    true,
	},
	{
		Name:        string(UserRoleWarehouse),
		DisplayName: "Admin Gudang",
		Description: "Penanggung jawab stok fisik, penerimaan, dan pengiriman barang",
		IsSystem:    true,
	},
	{
		Name:        string(UserRoleCashier),
		DisplayName: "Kasir",
		Description: "Operator kasir dan transaksi penjualan langsung di toko",
		IsSystem:    true,
	},
	{
		Name:        string(UserRoleCustomer),
		DisplayName: "Customer",
		Description: "Pelanggan toko untuk akses storefront dan riwayat belanja",
		IsSystem:    true,
	},
}

// PermissionSeed mendefinisikan struktur seeder permission.
type PermissionSeed struct {
	Name        string
	Module      string
	Description string
}

// SystemPermissions adalah daftar seluruh izin kapabilitas granular untuk modul Auth & Inventory.
var SystemPermissions = []PermissionSeed{
	// Shared: Manajemen Pengguna & Peran
	{Name: "users.view", Module: "shared", Description: "Melihat daftar dan profil staf atau pengguna"},
	{Name: "users.create", Module: "shared", Description: "Mendaftarkan akun staf baru"},
	{Name: "users.edit", Module: "shared", Description: "Mengubah status atau informasi staf"},
	{Name: "roles.view", Module: "shared", Description: "Melihat matriks peran dan daftar izin"},
	{Name: "roles.manage", Module: "shared", Description: "Mengatur dan mengubah hak akses peran"},

	// Inventory: Produk
	{Name: "inventory.products.view", Module: "inventory", Description: "Melihat katalog dan detail produk"},
	{Name: "inventory.products.view_cost", Module: "inventory", Description: "Melihat harga pokok penjualan (HPP / Modal) dan valuasi barang"},
	{Name: "inventory.products.create", Module: "inventory", Description: "Menambah data produk baru"},
	{Name: "inventory.products.edit", Module: "inventory", Description: "Memperbarui data produk"},
	{Name: "inventory.products.status", Module: "inventory", Description: "Mengubah status aktif/nonaktif produk"},

	// Inventory: Kategori
	{Name: "inventory.categories.view", Module: "inventory", Description: "Melihat daftar kategori produk"},
	{Name: "inventory.categories.create", Module: "inventory", Description: "Menambah kategori baru"},
	{Name: "inventory.categories.edit", Module: "inventory", Description: "Memperbarui kategori"},
	{Name: "inventory.categories.delete", Module: "inventory", Description: "Menghapus kategori"},

	// Inventory: Lokasi Cabang & Gudang
	{Name: "inventory.locations.view", Module: "inventory", Description: "Melihat daftar lokasi dan cabang toko"},
	{Name: "inventory.locations.create", Module: "inventory", Description: "Menambah lokasi atau cabang baru"},
	{Name: "inventory.locations.edit", Module: "inventory", Description: "Memperbarui informasi lokasi"},
	{Name: "inventory.locations.status", Module: "inventory", Description: "Mengubah status aktif lokasi"},
	{Name: "inventory.locations.delete", Module: "inventory", Description: "Menghapus lokasi cabang"},

	// Inventory: Stok Fisik
	{Name: "inventory.stocks.view", Module: "inventory", Description: "Melihat ketersediaan stok dan peringatan menipis"},
	{Name: "inventory.stocks.adjust", Module: "inventory", Description: "Melakukan penyesuaian stok manual (stock opname)"},
	{Name: "inventory.stocks.min_stock", Module: "inventory", Description: "Mengatur batas peringatan minimum stok"},

	// Inventory: Barcode
	{Name: "inventory.barcodes.view", Module: "inventory", Description: "Melihat dan mencari barcode produk"},
	{Name: "inventory.barcodes.manage", Module: "inventory", Description: "Menambah atau menghapus barcode produk"},

	// Inventory: Serial Number & IMEI
	{Name: "inventory.serials.view", Module: "inventory", Description: "Melihat dan mencari nomor serial atau IMEI unit"},
	{Name: "inventory.serials.register", Module: "inventory", Description: "Mendaftarkan unit fisik dengan nomor serial"},
	{Name: "inventory.serials.status", Module: "inventory", Description: "Memperbarui status unit serial (terjual/retur)"},

	// Inventory: Promosi & Harga Khusus Cabang (Price Override)
	{Name: "inventory.prices.view", Module: "inventory", Description: "Melihat daftar promo dan harga khusus cabang"},
	{Name: "inventory.prices.create", Module: "inventory", Description: "Membuat promo harga khusus cabang"},
	{Name: "inventory.prices.deactivate", Module: "inventory", Description: "Menonaktifkan promo harga cabang"},
	{Name: "inventory.prices.claim", Module: "inventory", Description: "Mengklaim kuota diskon saat transaksi kasir"},

	// Inventory: Mutasi Stok Antar Cabang (Stock Transfer)
	{Name: "inventory.transfers.view", Module: "inventory", Description: "Melihat riwayat dan detail mutasi stok"},
	{Name: "inventory.transfers.create", Module: "inventory", Description: "Membuat permohonan mutasi stok antar cabang"},
	{Name: "inventory.transfers.approve", Module: "inventory", Description: "Menyetujui atau menolak permohonan mutasi stok"},
	{Name: "inventory.transfers.ship", Module: "inventory", Description: "Mengonfirmasi pengiriman barang mutasi"},
	{Name: "inventory.transfers.receive", Module: "inventory", Description: "Mengonfirmasi penerimaan barang mutasi di cabang tujuan"},

	// Inventory: Garansi Produk (Warranty)
	{Name: "inventory.warranties.view", Module: "inventory", Description: "Melihat master kebijakan dan garansi produk"},
	{Name: "inventory.warranties.manage", Module: "inventory", Description: "Membuat kebijakan atau menetapkan garansi ke produk"},
}

// DefaultRolePermissionAssignments adalah konfigurasi default hak akses per peran.
var DefaultRolePermissionAssignments = map[string][]string{
	"admin": {
		"users.view", "roles.view",
		"inventory.products.view", "inventory.products.view_cost", "inventory.products.create", "inventory.products.edit", "inventory.products.status",
		"inventory.categories.view", "inventory.categories.create", "inventory.categories.edit", "inventory.categories.delete",
		"inventory.locations.view", "inventory.locations.create", "inventory.locations.edit", "inventory.locations.status",
		"inventory.stocks.view", "inventory.stocks.adjust", "inventory.stocks.min_stock",
		"inventory.barcodes.view", "inventory.barcodes.manage",
		"inventory.serials.view", "inventory.serials.register", "inventory.serials.status",
		"inventory.prices.view", "inventory.prices.create", "inventory.prices.deactivate", "inventory.prices.claim",
		"inventory.transfers.view", "inventory.transfers.create", "inventory.transfers.ship", "inventory.transfers.receive",
		"inventory.warranties.view", "inventory.warranties.manage",
	},
	"warehouse": {
		"inventory.products.view",
		"inventory.stocks.view",
		"inventory.barcodes.view", "inventory.barcodes.manage",
		"inventory.serials.view", "inventory.serials.register", "inventory.serials.status",
		"inventory.transfers.view", "inventory.transfers.ship", "inventory.transfers.receive",
		"inventory.warranties.view",
	},
	"cashier": {
		"inventory.products.view",
		"inventory.stocks.view",
		"inventory.barcodes.view",
		"inventory.serials.view",
		"inventory.prices.view", "inventory.prices.claim",
		"inventory.warranties.view",
	},
	"customer": {},
}

// SeedRoles menyuntikkan daftar peran standar ke tabel shared_roles secara idempoten.
func SeedRoles(ctx context.Context, db *sql.DB) (int, error) {
	now := time.Now().UTC()
	inserted := 0

	queryCheck := `SELECT id FROM shared_roles WHERE name = ? LIMIT 1`
	queryInsert := `
		INSERT INTO shared_roles (id, name, display_name, description, is_system, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	for _, r := range SystemRoles {
		var id string
		err := db.QueryRowContext(ctx, queryCheck, r.Name).Scan(&id)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			return inserted, fmt.Errorf("gagal cek role '%s': %w", r.Name, err)
		}

		newID := uid.New()
		_, err = db.ExecContext(ctx, queryInsert, newID, r.Name, r.DisplayName, r.Description, r.IsSystem, now, now)
		if err != nil {
			return inserted, fmt.Errorf("gagal insert role '%s': %w", r.Name, err)
		}
		inserted++
	}

	return inserted, nil
}

// SeedPermissions menyuntikkan daftar izin kapabilitas standar ke tabel shared_permissions secara idempoten.
func SeedPermissions(ctx context.Context, db *sql.DB) (int, error) {
	now := time.Now().UTC()
	inserted := 0

	queryCheck := `SELECT id FROM shared_permissions WHERE name = ? LIMIT 1`
	queryInsert := `
		INSERT INTO shared_permissions (id, name, module, description, created_at)
		VALUES (?, ?, ?, ?, ?)`

	for _, p := range SystemPermissions {
		var id string
		err := db.QueryRowContext(ctx, queryCheck, p.Name).Scan(&id)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			return inserted, fmt.Errorf("gagal cek permission '%s': %w", p.Name, err)
		}

		newID := uid.New()
		_, err = db.ExecContext(ctx, queryInsert, newID, p.Name, p.Module, p.Description, now)
		if err != nil {
			return inserted, fmt.Errorf("gagal insert permission '%s': %w", p.Name, err)
		}
		inserted++
	}

	return inserted, nil
}

// SeedDefaultRolePermissions menghubungkan peran dengan izin default-nya secara idempoten.
func SeedDefaultRolePermissions(ctx context.Context, db *sql.DB) (int, error) {
	now := time.Now().UTC()
	totalInserted := 0

	// Ambil peta role name -> role_id
	roleRows, err := db.QueryContext(ctx, `SELECT id, name FROM shared_roles`)
	if err != nil {
		return 0, fmt.Errorf("gagal mengambil roles: %w", err)
	}
	defer roleRows.Close()

	roleNameToID := make(map[string]string)
	for roleRows.Next() {
		var id, name string
		if err := roleRows.Scan(&id, &name); err != nil {
			return 0, err
		}
		roleNameToID[name] = id
	}

	// Ambil peta permission name -> permission_id
	permRows, err := db.QueryContext(ctx, `SELECT id, name FROM shared_permissions`)
	if err != nil {
		return 0, fmt.Errorf("gagal mengambil permissions: %w", err)
	}
	defer permRows.Close()

	permNameToID := make(map[string]string)
	for permRows.Next() {
		var id, name string
		if err := permRows.Scan(&id, &name); err != nil {
			return 0, err
		}
		permNameToID[name] = id
	}

	queryCheck := `SELECT 1 FROM shared_role_permissions WHERE role_id = ? AND permission_id = ? LIMIT 1`
	queryInsert := `INSERT INTO shared_role_permissions (role_id, permission_id, created_at) VALUES (?, ?, ?)`

	for roleName, permNames := range DefaultRolePermissionAssignments {
		roleID, ok := roleNameToID[roleName]
		if !ok {
			continue
		}

		for _, pName := range permNames {
			permID, ok := permNameToID[pName]
			if !ok {
				continue
			}

			var dummy int
			err := db.QueryRowContext(ctx, queryCheck, roleID, permID).Scan(&dummy)
			if err == nil {
				// Sudah terhubung
				continue
			}
			if err != sql.ErrNoRows {
				return totalInserted, fmt.Errorf("gagal cek relasi role-perm (%s - %s): %w", roleName, pName, err)
			}

			_, err = db.ExecContext(ctx, queryInsert, roleID, permID, now)
			if err != nil {
				return totalInserted, fmt.Errorf("gagal insert relasi role-perm (%s - %s): %w", roleName, pName, err)
			}
			totalInserted++
		}
	}

	return totalInserted, nil
}
