package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

type mysqlPriceOverrideRepository struct {
	db *sql.DB
}

// NewPriceOverrideRepository membuat implementasi baru domain.PriceOverrideRepository berbasis MySQL.
func NewPriceOverrideRepository(db *sql.DB) domain.PriceOverrideRepository {
	return &mysqlPriceOverrideRepository{db: db}
}

// Save menyimpan promo harga baru ke tabel inv_price_overrides.
func (r *mysqlPriceOverrideRepository) Save(ctx context.Context, po *domain.PriceOverride) error {
	query := `
		INSERT INTO inv_price_overrides (
			id, product_id, location_id, promotional_price, max_quantity, claimed_quantity,
			start_date, end_date, reason, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		po.ID, po.ProductID, po.LocationID, po.PromotionalPrice, po.MaxQuantity, po.ClaimedQuantity,
		po.StartDate, po.EndDate, po.Reason, po.IsActive, po.CreatedAt, po.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert price override: %w", err)
	}
	return nil
}

// Update memperbarui data promo (misal perpanjang promo atau deactive).
func (r *mysqlPriceOverrideRepository) Update(ctx context.Context, po *domain.PriceOverride) error {
	query := `
		UPDATE inv_price_overrides SET
			promotional_price = ?, max_quantity = ?, claimed_quantity = ?,
			start_date = ?, end_date = ?, reason = ?, is_active = ?, updated_at = ?
		WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query,
		po.PromotionalPrice, po.MaxQuantity, po.ClaimedQuantity,
		po.StartDate, po.EndDate, po.Reason, po.IsActive, po.UpdatedAt, po.ID,
	)
	if err != nil {
		return fmt.Errorf("gagal update price override: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa rows affected update price override: %w", err)
	}
	if rows == 0 {
		return domain.ErrPromoNotFound
	}

	return nil
}

// FindByID mencari promo berdasarkan ID UUID.
func (r *mysqlPriceOverrideRepository) FindByID(ctx context.Context, id string) (*domain.PriceOverride, error) {
	query := `
		SELECT id, product_id, location_id, promotional_price, max_quantity, claimed_quantity,
		       start_date, end_date, reason, is_active, created_at, updated_at
		FROM inv_price_overrides
		WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanPriceOverride(row)
}

// FindActive mencari promo yang sedang aktif berlaku pada waktu 'at' untuk produk & cabang tertentu,
// dan kuotanya belum habis (max_quantity IS NULL ATAU claimed_quantity < max_quantity).
func (r *mysqlPriceOverrideRepository) FindActive(ctx context.Context, productID, locationID string, at time.Time) (*domain.PriceOverride, error) {
	query := `
		SELECT id, product_id, location_id, promotional_price, max_quantity, claimed_quantity,
		       start_date, end_date, reason, is_active, created_at, updated_at
		FROM inv_price_overrides
		WHERE product_id = ? AND location_id = ? AND is_active = TRUE
		  AND start_date <= ? AND end_date >= ?
		  AND (max_quantity IS NULL OR claimed_quantity < max_quantity)
		ORDER BY promotional_price ASC
		LIMIT 1`

	atUTC := at.UTC()
	row := r.db.QueryRowContext(ctx, query, productID, locationID, atUTC, atUTC)
	return r.scanPriceOverride(row)
}

// ListByProduct mengambil seluruh riwayat promo untuk suatu produk.
func (r *mysqlPriceOverrideRepository) ListByProduct(ctx context.Context, productID string, locationID *string) ([]*domain.PriceOverride, error) {
	query := `
		SELECT id, product_id, location_id, promotional_price, max_quantity, claimed_quantity,
		       start_date, end_date, reason, is_active, created_at, updated_at
		FROM inv_price_overrides
		WHERE product_id = ?`
	args := []any{productID}

	if locationID != nil && *locationID != "" {
		query += ` AND location_id = ?`
		args = append(args, *locationID)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal query list price overrides: %w", err)
	}
	defer rows.Close()

	return r.scanPriceOverrides(rows)
}

// HasOverlappingPromo memeriksa apakah ada promo aktif lain yang rentang tanggalnya bertabrakan.
// Formula tabrakan: start_date < end AND end_date > start.
func (r *mysqlPriceOverrideRepository) HasOverlappingPromo(ctx context.Context, productID, locationID string, start, end time.Time, excludeID *string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM inv_price_overrides
		WHERE product_id = ? AND location_id = ? AND is_active = TRUE
		  AND start_date < ? AND end_date > ?`
	args := []any{productID, locationID, end.UTC(), start.UTC()}

	if excludeID != nil && *excludeID != "" {
		query += ` AND id != ?`
		args = append(args, *excludeID)
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("gagal memeriksa tumpang tindih promo: %w", err)
	}

	return count > 0, nil
}

// IncrementClaimedQuantity menambah claimed_quantity secara atomik dan aman dari race condition.
// Jika penambahan ini akan melebihi max_quantity, query tidak mengubah baris dan mengembalikan ErrPromoQuotaExhausted.
func (r *mysqlPriceOverrideRepository) IncrementClaimedQuantity(ctx context.Context, id string, delta int) error {
	if delta <= 0 {
		return errors.New("delta klaim kuota harus lebih dari 0")
	}

	query := `
		UPDATE inv_price_overrides
		SET claimed_quantity = claimed_quantity + ?, updated_at = ?
		WHERE id = ? AND is_active = TRUE
		  AND (max_quantity IS NULL OR claimed_quantity + ? <= max_quantity)`

	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, query, delta, now, id, delta)
	if err != nil {
		return fmt.Errorf("gagal increment claimed quantity: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa rows affected increment claimed quantity: %w", err)
	}
	if rows == 0 {
		// Evaluasi apakah promo tidak ditemukan atau kuota habis
		var maxQty *int
		var claimed int
		checkErr := r.db.QueryRowContext(ctx, "SELECT max_quantity, claimed_quantity FROM inv_price_overrides WHERE id = ?", id).Scan(&maxQty, &claimed)
		if checkErr != nil {
			if errors.Is(checkErr, sql.ErrNoRows) {
				return domain.ErrPromoNotFound
			}
			return checkErr
		}
		if maxQty != nil && claimed+delta > *maxQty {
			return domain.ErrPromoQuotaExhausted
		}
		return domain.ErrPromoNotFound
	}

	return nil
}

// Helper untuk membaca 1 baris hasil scan row
func (r *mysqlPriceOverrideRepository) scanPriceOverride(scanner interface{ Scan(dest ...any) error }) (*domain.PriceOverride, error) {
	var (
		id, productID, locationID string
		promoPrice               float64
		maxQty                   sql.NullInt64
		claimedQty               int
		start, end               time.Time
		reason                   sql.NullString
		isActive                 bool
		createdAt, updatedAt     time.Time
	)

	err := scanner.Scan(
		&id, &productID, &locationID, &promoPrice, &maxQty, &claimedQty,
		&start, &end, &reason, &isActive, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPromoNotFound
		}
		return nil, fmt.Errorf("gagal scan price override: %w", err)
	}

	var reasonStr string
	if reason.Valid {
		reasonStr = reason.String
	}

	var maxQuantity *int
	if maxQty.Valid {
		val := int(maxQty.Int64)
		maxQuantity = &val
	}

	return &domain.PriceOverride{
		ID:               id,
		ProductID:        productID,
		LocationID:       locationID,
		PromotionalPrice: int64(promoPrice),
		MaxQuantity:      maxQuantity,
		ClaimedQuantity:  claimedQty,
		StartDate:        start,
		EndDate:          end,
		Reason:           reasonStr,
		IsActive:         isActive,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}, nil
}

// Helper untuk membaca banyak baris hasil query rows
func (r *mysqlPriceOverrideRepository) scanPriceOverrides(rows *sql.Rows) ([]*domain.PriceOverride, error) {
	var list []*domain.PriceOverride
	for rows.Next() {
		var (
			id, productID, locationID string
			promoPrice               float64
			maxQty                   sql.NullInt64
			claimedQty               int
			start, end               time.Time
			reason                   sql.NullString
			isActive                 bool
			createdAt, updatedAt     time.Time
		)
		if err := rows.Scan(
			&id, &productID, &locationID, &promoPrice, &maxQty, &claimedQty,
			&start, &end, &reason, &isActive, &createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan row price override: %w", err)
		}

		var reasonStr string
		if reason.Valid {
			reasonStr = reason.String
		}

		var maxQuantity *int
		if maxQty.Valid {
			val := int(maxQty.Int64)
			maxQuantity = &val
		}

		list = append(list, &domain.PriceOverride{
			ID:               id,
			ProductID:        productID,
			LocationID:       locationID,
			PromotionalPrice: int64(promoPrice),
			MaxQuantity:      maxQuantity,
			ClaimedQuantity:  claimedQty,
			StartDate:        start,
			EndDate:          end,
			Reason:           reasonStr,
			IsActive:         isActive,
			CreatedAt:        createdAt,
			UpdatedAt:        updatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi rows price overrides: %w", err)
	}

	return list, nil
}
