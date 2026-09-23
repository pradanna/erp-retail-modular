package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/pkg/uid"
)

type mysqlStockRepository struct {
	db *sql.DB
}

// NewStockRepository membuat implementasi baru domain.StockRepository berbasis database MySQL.
func NewStockRepository(db *sql.DB) domain.StockRepository {
	return &mysqlStockRepository{db: db}
}

// Save menyimpan catatan stok baru ke tabel inv_stocks.
func (r *mysqlStockRepository) Save(ctx context.Context, item *domain.StockItem) error {
	query := `
		INSERT INTO inv_stocks (id, product_id, location_id, quantity, reserved_quantity, min_stock, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		item.ID, item.ProductID, item.LocationID,
		item.Quantity, item.ReservedQuantity, item.MinStock, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert stock item: %w", err)
	}
	return nil
}

// Update memperbarui kuantitas, reservasi, min_stock, dan updated_at pada baris stok yang sudah ada.
func (r *mysqlStockRepository) Update(ctx context.Context, item *domain.StockItem) error {
	query := `
		UPDATE inv_stocks SET
			quantity = ?, reserved_quantity = ?, min_stock = ?, updated_at = ?
		WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query,
		item.Quantity, item.ReservedQuantity, item.MinStock, item.UpdatedAt, item.ID,
	)
	if err != nil {
		return fmt.Errorf("gagal update stock item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa rows affected update stock: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("stok dengan ID %s tidak ditemukan", item.ID)
	}

	return nil
}

// FindByProductAndLocation mencari stok untuk kombinasi produk dan lokasi tertentu.
func (r *mysqlStockRepository) FindByProductAndLocation(ctx context.Context, productID, locationID string) (*domain.StockItem, error) {
	query := `
		SELECT id, product_id, location_id, quantity, reserved_quantity, min_stock, updated_at
		FROM inv_stocks
		WHERE product_id = ? AND location_id = ?`

	var item domain.StockItem
	err := r.db.QueryRowContext(ctx, query, productID, locationID).Scan(
		&item.ID, &item.ProductID, &item.LocationID,
		&item.Quantity, &item.ReservedQuantity, &item.MinStock, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal query stock by product and location: %w", err)
	}

	return &item, nil
}

// ListByLocation mengambil seluruh stok produk pada suatu cabang/gudang.
func (r *mysqlStockRepository) ListByLocation(ctx context.Context, locationID string) ([]*domain.StockItem, error) {
	query := `
		SELECT id, product_id, location_id, quantity, reserved_quantity, min_stock, updated_at
		FROM inv_stocks
		WHERE location_id = ?
		ORDER BY updated_at DESC`

	rows, err := r.db.QueryContext(ctx, query, locationID)
	if err != nil {
		return nil, fmt.Errorf("gagal query list stock by location: %w", err)
	}
	defer rows.Close()

	var items []*domain.StockItem
	for rows.Next() {
		var item domain.StockItem
		if err := rows.Scan(
			&item.ID, &item.ProductID, &item.LocationID,
			&item.Quantity, &item.ReservedQuantity, &item.MinStock, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan stock row: %w", err)
		}
		items = append(items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi stock rows: %w", err)
	}

	return items, nil
}

// ListLowStockAlerts mengambil daftar stok yang kuantitasnya berada di bawah atau sama dengan min_stock.
func (r *mysqlStockRepository) ListLowStockAlerts(ctx context.Context, locationID *string) ([]*domain.StockItem, error) {
	var query string
	var args []any

	if locationID != nil && *locationID != "" {
		query = `
			SELECT id, product_id, location_id, quantity, reserved_quantity, min_stock, updated_at
			FROM inv_stocks
			WHERE location_id = ? AND quantity <= min_stock
			ORDER BY (quantity - min_stock) ASC, updated_at DESC`
		args = append(args, *locationID)
	} else {
		query = `
			SELECT id, product_id, location_id, quantity, reserved_quantity, min_stock, updated_at
			FROM inv_stocks
			WHERE quantity <= min_stock
			ORDER BY (quantity - min_stock) ASC, updated_at DESC`
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal query low stock alerts: %w", err)
	}
	defer rows.Close()

	var items []*domain.StockItem
	for rows.Next() {
		var item domain.StockItem
		if err := rows.Scan(
			&item.ID, &item.ProductID, &item.LocationID,
			&item.Quantity, &item.ReservedQuantity, &item.MinStock, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan low stock row: %w", err)
		}
		items = append(items, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi low stock rows: %w", err)
	}

	return items, nil
}

// AtomicMutate mengeksekusi mutasi stok secara atomik dengan row-level locking (SELECT ... FOR UPDATE).
//
// MENGAPA MENGGUNAKAN POLA CLOSURE INI?
// 1. Isolasi Transaksi: Membuka transaksi dan mengunci baris spesifik produk & lokasi di database.
// 2. Race Condition Prevention: Jika 2 kasir atau proses mencoba mengubah stok di detik yang sama,
//    transaksi kedua akan antre menunggu transaksi pertama selesai di-commit/rollback.
// 3. Domain Purity: Aturan bisnis murni tetap berada di method entity domain (*domain.StockItem),
//    sedangkan detail teknis transaksi database dan FOR UPDATE terkapsulasi di layer infrastructure.
func (r *mysqlStockRepository) AtomicMutate(ctx context.Context, productID, locationID string, mutateFn func(item *domain.StockItem) error) (*domain.StockItem, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi database stok: %w", err)
	}
	defer tx.Rollback() // Aman: jika commit berhasil di akhir, rollback ini diabaikan otomatis

	var item domain.StockItem
	var isNew bool

	query := `
		SELECT id, product_id, location_id, quantity, reserved_quantity, min_stock, updated_at
		FROM inv_stocks
		WHERE product_id = ? AND location_id = ?
		FOR UPDATE`

	err = tx.QueryRowContext(ctx, query, productID, locationID).Scan(
		&item.ID, &item.ProductID, &item.LocationID,
		&item.Quantity, &item.ReservedQuantity, &item.MinStock, &item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Jika baris belum ada di cabang ini, buat objek baru dengan nilai awal 0
			isNew = true
			newItem, err := domain.NewStockItem(uid.New(), productID, locationID, 0, 0)
			if err != nil {
				return nil, fmt.Errorf("gagal menginisialisasi stock item baru: %w", err)
			}
			item = *newItem
		} else {
			return nil, fmt.Errorf("gagal mengunci baris stok untuk update: %w", err)
		}
	}

	// Eksekusi fungsi perubahan bisnis (validasi domain berjalan di sini)
	if err := mutateFn(&item); err != nil {
		return nil, err // Rollback otomatis terpanggil oleh defer
	}

	// Persist hasil mutasi ke database
	if isNew {
		insertQuery := `
			INSERT INTO inv_stocks (id, product_id, location_id, quantity, reserved_quantity, min_stock, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`
		_, err = tx.ExecContext(ctx, insertQuery,
			item.ID, item.ProductID, item.LocationID,
			item.Quantity, item.ReservedQuantity, item.MinStock, item.UpdatedAt,
		)
	} else {
		updateQuery := `
			UPDATE inv_stocks SET
				quantity = ?, reserved_quantity = ?, min_stock = ?, updated_at = ?
			WHERE id = ?`
		_, err = tx.ExecContext(ctx, updateQuery,
			item.Quantity, item.ReservedQuantity, item.MinStock, item.UpdatedAt, item.ID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("gagal menyimpan perubahan stok: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal commit transaksi stok: %w", err)
	}

	return &item, nil
}
