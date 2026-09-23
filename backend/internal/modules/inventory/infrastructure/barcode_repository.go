package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

type mysqlBarcodeRepository struct {
	db *sql.DB
}

// NewBarcodeRepository membuat implementasi baru domain.BarcodeRepository berbasis database MySQL.
func NewBarcodeRepository(db *sql.DB) domain.BarcodeRepository {
	return &mysqlBarcodeRepository{db: db}
}

// Save menyimpan barcode baru ke tabel inv_barcodes.
func (r *mysqlBarcodeRepository) Save(ctx context.Context, b *domain.ProductBarcode) error {
	query := `
		INSERT INTO inv_barcodes (id, product_id, barcode, is_primary, created_at)
		VALUES (?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		b.ID, b.ProductID, b.Barcode, b.IsPrimary, b.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert barcode: %w", err)
	}
	return nil
}

// Delete menghapus barcode berdasarkan ID primary key-nya.
func (r *mysqlBarcodeRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM inv_barcodes WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("gagal delete barcode: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa rows affected delete barcode: %w", err)
	}
	if rows == 0 {
		return domain.ErrBarcodeNotFound
	}

	return nil
}

// ListByProductID mengambil seluruh barcode yang terdaftar pada suatu produk.
func (r *mysqlBarcodeRepository) ListByProductID(ctx context.Context, productID string) ([]*domain.ProductBarcode, error) {
	query := `
		SELECT id, product_id, barcode, is_primary, created_at
		FROM inv_barcodes
		WHERE product_id = ?
		ORDER BY is_primary DESC, created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("gagal query barcodes by product: %w", err)
	}
	defer rows.Close()

	var barcodes []*domain.ProductBarcode
	for rows.Next() {
		var b domain.ProductBarcode
		if err := rows.Scan(&b.ID, &b.ProductID, &b.Barcode, &b.IsPrimary, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("gagal scan barcode row: %w", err)
		}
		barcodes = append(barcodes, &b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi barcode rows: %w", err)
	}

	return barcodes, nil
}

// FindByID mencari barcode berdasarkan ID primary key.
func (r *mysqlBarcodeRepository) FindByID(ctx context.Context, id string) (*domain.ProductBarcode, error) {
	query := `
		SELECT id, product_id, barcode, is_primary, created_at
		FROM inv_barcodes
		WHERE id = ?`

	var b domain.ProductBarcode
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&b.ID, &b.ProductID, &b.Barcode, &b.IsPrimary, &b.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal query barcode by id: %w", err)
	}

	return &b, nil
}

// FindByBarcode mencari barcode berdasarkan kode uniknya.
func (r *mysqlBarcodeRepository) FindByBarcode(ctx context.Context, code string) (*domain.ProductBarcode, error) {
	query := `
		SELECT id, product_id, barcode, is_primary, created_at
		FROM inv_barcodes
		WHERE barcode = ?`

	var b domain.ProductBarcode
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&b.ID, &b.ProductID, &b.Barcode, &b.IsPrimary, &b.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal query barcode by code: %w", err)
	}

	return &b, nil
}

// FindProductByBarcode mencari data produk lengkap beserta info barcodenya dalam 1 query JOIN (Fast Lookup Kasir).
func (r *mysqlBarcodeRepository) FindProductByBarcode(ctx context.Context, code string) (*domain.Product, *domain.ProductBarcode, error) {
	query := `
		SELECT 
			p.id, p.sku, p.category_id, p.name, p.brand, p.description, p.unit,
			p.purchase_price, p.selling_price, p.status,
			p.is_ppn, p.flag_serial_tracking, p.weight_gram, p.atribut_varian,
			p.created_at, p.updated_at,
			b.id, b.product_id, b.barcode, b.is_primary, b.created_at
		FROM inv_barcodes b
		JOIN inv_products p ON b.product_id = p.id
		WHERE b.barcode = ?`

	var (
		pID, sku                              string
		categoryID, description               sql.NullString
		name, brand, unit                     string
		purchasePrice, sellingPrice           float64
		statusStr                             string
		isPPN, flagSerialTracking             bool
		weightGram                            int
		varianJSON                            sql.NullString
		pCreatedAt, pUpdatedAt                time.Time
		bID, bProductID, bCode                string
		bIsPrimary                            bool
		bCreatedAt                            time.Time
	)

	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&pID, &sku, &categoryID, &name, &brand, &description, &unit,
		&purchasePrice, &sellingPrice, &statusStr,
		&isPPN, &flagSerialTracking, &weightGram, &varianJSON,
		&pCreatedAt, &pUpdatedAt,
		&bID, &bProductID, &bCode, &bIsPrimary, &bCreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("gagal query product by barcode: %w", err)
	}

	prod, err := buildProductFromScan(
		pID, sku, categoryID, name, brand, description, unit,
		purchasePrice, sellingPrice, statusStr,
		isPPN, flagSerialTracking, weightGram, varianJSON,
		pCreatedAt, pUpdatedAt,
	)
	if err != nil {
		return nil, nil, err
	}

	barcode := &domain.ProductBarcode{
		ID:        bID,
		ProductID: bProductID,
		Barcode:   bCode,
		IsPrimary: bIsPrimary,
		CreatedAt: bCreatedAt,
	}

	return prod, barcode, nil
}

// ResetPrimary mengubah seluruh barcode milik produk menjadi is_primary = false.
func (r *mysqlBarcodeRepository) ResetPrimary(ctx context.Context, productID string) error {
	query := `UPDATE inv_barcodes SET is_primary = FALSE WHERE product_id = ?`
	_, err := r.db.ExecContext(ctx, query, productID)
	if err != nil {
		return fmt.Errorf("gagal reset is_primary barcodes: %w", err)
	}
	return nil
}
