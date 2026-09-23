package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

// mysqlProductRepository adalah implementasi konkret dari domain.ProductRepository
// menggunakan MySQL dan database/sql standar Go.
//
// MENGAPA IMPLEMENTASI ADA DI INFRASTRUCTURE?
// Layer infrastructure tahu tentang detail teknis: SQL dialect, placeholder '?', JSON serialization, dsb.
// Domain dan use case sama sekali tidak boleh tahu detail ini.
type mysqlProductRepository struct {
	db *sql.DB
}

// NewProductRepository membuat instance repository baru.
// Mengembalikan domain.ProductRepository (interface), bukan tipe konkret struct-nya.
func NewProductRepository(db *sql.DB) domain.ProductRepository {
	return &mysqlProductRepository{db: db}
}

func (r *mysqlProductRepository) Save(ctx context.Context, p *domain.Product) error {
	// Simpan AtributVarian sebagai JSON text ke kolom JSON MySQL
	varianJSON, err := json.Marshal(p.AtributVarian)
	if err != nil {
		return fmt.Errorf("gagal marshal atribut_varian: %w", err)
	}

	query := `
		INSERT INTO inv_products (
			id, sku, category_id, name, brand, description, unit,
			purchase_price, selling_price, status,
			is_ppn, flag_serial_tracking, weight_gram, atribut_varian,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = r.db.ExecContext(ctx, query,
		p.ID, p.SKU, p.CategoryID, p.Name, p.Brand, p.Description, p.Unit,
		p.PurchasePrice, p.SellingPrice, string(p.Status),
		p.IsPPN, p.FlagSerialTracking, p.WeightGram, string(varianJSON),
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert produk ke database: %w", err)
	}
	return nil
}

func (r *mysqlProductRepository) FindByID(ctx context.Context, id string) (*domain.Product, error) {
	query := `
		SELECT id, sku, category_id, name, brand, description, unit,
			   purchase_price, selling_price, status,
			   is_ppn, flag_serial_tracking, weight_gram, atribut_varian,
			   created_at, updated_at
		FROM inv_products
		WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanProduct(row)
}

func (r *mysqlProductRepository) FindBySKU(ctx context.Context, sku string) (*domain.Product, error) {
	query := `
		SELECT id, sku, category_id, name, brand, description, unit,
			   purchase_price, selling_price, status,
			   is_ppn, flag_serial_tracking, weight_gram, atribut_varian,
			   created_at, updated_at
		FROM inv_products
		WHERE sku = ?`

	row := r.db.QueryRowContext(ctx, query, sku)
	p, err := scanProduct(row)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *mysqlProductRepository) Update(ctx context.Context, p *domain.Product) error {
	varianJSON, err := json.Marshal(p.AtributVarian)
	if err != nil {
		return fmt.Errorf("gagal marshal atribut_varian: %w", err)
	}

	query := `
		UPDATE inv_products SET
			name = ?, brand = ?, description = ?, unit = ?,
			purchase_price = ?, selling_price = ?, status = ?,
			is_ppn = ?, flag_serial_tracking = ?, weight_gram = ?,
			atribut_varian = ?, updated_at = ?
		WHERE id = ?`

	_, err = r.db.ExecContext(ctx, query,
		p.Name, p.Brand, p.Description, p.Unit,
		p.PurchasePrice, p.SellingPrice, string(p.Status),
		p.IsPPN, p.FlagSerialTracking, p.WeightGram,
		string(varianJSON), p.UpdatedAt,
		p.ID,
	)
	return err
}

func (r *mysqlProductRepository) List(ctx context.Context, filter domain.ProductFilter) ([]*domain.Product, int, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 20
	}
	offset := (filter.Page - 1) * filter.Limit

	whereClause := " WHERE 1=1"
	var args []any

	if filter.Status != nil {
		whereClause += " AND status = ?"
		args = append(args, string(*filter.Status))
	}
	if filter.CategoryID != nil && *filter.CategoryID != "" {
		whereClause += " AND category_id = ?"
		args = append(args, *filter.CategoryID)
	}
	if filter.Search != nil && *filter.Search != "" {
		whereClause += " AND (name LIKE ? OR sku LIKE ?)"
		pattern := "%" + *filter.Search + "%"
		args = append(args, pattern, pattern)
	}

	// 1. Hitung total
	countQuery := "SELECT COUNT(*) FROM inv_products" + whereClause
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("gagal hitung total produk: %w", err)
	}

	// 2. Ambil data dengan LIMIT dan OFFSET
	selectQuery := `
		SELECT id, sku, category_id, name, brand, description, unit,
			   purchase_price, selling_price, status,
			   is_ppn, flag_serial_tracking, weight_gram, atribut_varian,
			   created_at, updated_at
		FROM inv_products` + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"

	queryArgs := append(args, filter.Limit, offset)
	rows, err := r.db.QueryContext(ctx, selectQuery, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal query list produk: %w", err)
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		p, err := scanProductRow(rows)
		if err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}

	return products, total, nil
}

// Helper untuk membaca 1 row dari QueryRow
func scanProduct(row *sql.Row) (*domain.Product, error) {
	var (
		id, sku, name, brand, unit, statusStr string
		categoryID, description               sql.NullString
		purchasePrice, sellingPrice           float64
		weightGram                            int
		isPPN, flagSerialTracking             bool
		varianJSON                            sql.NullString
		createdAt, updatedAt                  time.Time
	)

	err := row.Scan(
		&id, &sku, &categoryID, &name, &brand, &description, &unit,
		&purchasePrice, &sellingPrice, &statusStr,
		&isPPN, &flagSerialTracking, &weightGram, &varianJSON,
		&createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal scan produk: %w", err)
	}

	return buildProductFromScan(
		id, sku, categoryID, name, brand, description, unit,
		purchasePrice, sellingPrice, statusStr,
		isPPN, flagSerialTracking, weightGram, varianJSON,
		createdAt, updatedAt,
	)
}

// Helper untuk membaca row dalam iterasi rows.Next()
func scanProductRow(rows *sql.Rows) (*domain.Product, error) {
	var (
		id, sku, name, brand, unit, statusStr string
		categoryID, description               sql.NullString
		purchasePrice, sellingPrice           float64
		weightGram                            int
		isPPN, flagSerialTracking             bool
		varianJSON                            sql.NullString
		createdAt, updatedAt                  time.Time
	)

	err := rows.Scan(
		&id, &sku, &categoryID, &name, &brand, &description, &unit,
		&purchasePrice, &sellingPrice, &statusStr,
		&isPPN, &flagSerialTracking, &weightGram, &varianJSON,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal scan baris produk: %w", err)
	}

	return buildProductFromScan(
		id, sku, categoryID, name, brand, description, unit,
		purchasePrice, sellingPrice, statusStr,
		isPPN, flagSerialTracking, weightGram, varianJSON,
		createdAt, updatedAt,
	)
}

func buildProductFromScan(
	id, sku string, categoryID sql.NullString, name, brand string,
	description sql.NullString, unit string,
	purchasePrice, sellingPrice float64, statusStr string,
	isPPN, flagSerialTracking bool, weightGram int,
	varianJSON sql.NullString,
	createdAt, updatedAt time.Time,
) (*domain.Product, error) {
	var catID string
	if categoryID.Valid {
		catID = categoryID.String
	}

	var desc string
	if description.Valid {
		desc = description.String
	}

	atribut := make(map[string]any)
	if varianJSON.Valid && varianJSON.String != "" {
		if err := json.Unmarshal([]byte(varianJSON.String), &atribut); err != nil {
			return nil, fmt.Errorf("gagal unmarshal atribut_varian JSON: %w", err)
		}
	}

	return &domain.Product{
		ID:                 id,
		SKU:                sku,
		CategoryID:         catID,
		Name:               name,
		Brand:              brand,
		Description:        desc,
		Unit:               unit,
		PurchasePrice:      int64(purchasePrice),
		SellingPrice:       int64(sellingPrice),
		Status:             domain.ProductStatus(statusStr),
		IsPPN:              isPPN,
		FlagSerialTracking: flagSerialTracking,
		WeightGram:         weightGram,
		AtributVarian:      atribut,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	}, nil
}
