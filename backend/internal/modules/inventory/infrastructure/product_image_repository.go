package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

type productImageRepository struct {
	db *sql.DB
}

// NewProductImageRepository membuat instance baru dari ProductImageRepository.
func NewProductImageRepository(db *sql.DB) domain.ProductImageRepository {
	return &productImageRepository{db: db}
}

func (r *productImageRepository) Save(ctx context.Context, img *domain.ProductImage) error {
	query := `
		INSERT INTO inv_product_images (id, product_id, url, is_primary, sort_order, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		img.ID,
		img.ProductID,
		img.URL,
		img.IsPrimary,
		img.SortOrder,
		img.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal menyimpan foto produk: %w", err)
	}
	return nil
}

func (r *productImageRepository) FindByProductID(ctx context.Context, productID string) ([]*domain.ProductImage, error) {
	query := `
		SELECT id, product_id, url, is_primary, sort_order, created_at
		FROM inv_product_images
		WHERE product_id = ?
		ORDER BY is_primary DESC, sort_order ASC, created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil foto produk: %w", err)
	}
	defer rows.Close()

	var images []*domain.ProductImage
	for rows.Next() {
		var img domain.ProductImage
		if err := rows.Scan(
			&img.ID,
			&img.ProductID,
			&img.URL,
			&img.IsPrimary,
			&img.SortOrder,
			&img.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan foto produk: %w", err)
		}
		images = append(images, &img)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return images, nil
}

func (r *productImageRepository) FindByID(ctx context.Context, id string) (*domain.ProductImage, error) {
	query := `
		SELECT id, product_id, url, is_primary, sort_order, created_at
		FROM inv_product_images
		WHERE id = ?
	`
	var img domain.ProductImage
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&img.ID,
		&img.ProductID,
		&img.URL,
		&img.IsPrimary,
		&img.SortOrder,
		&img.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal mencari foto produk berdasarkan id: %w", err)
	}
	return &img, nil
}

func (r *productImageRepository) FindPrimaryByProductID(ctx context.Context, productID string) (*domain.ProductImage, error) {
	query := `
		SELECT id, product_id, url, is_primary, sort_order, created_at
		FROM inv_product_images
		WHERE product_id = ? AND is_primary = TRUE
		LIMIT 1
	`
	var img domain.ProductImage
	err := r.db.QueryRowContext(ctx, query, productID).Scan(
		&img.ID,
		&img.ProductID,
		&img.URL,
		&img.IsPrimary,
		&img.SortOrder,
		&img.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("gagal mencari foto utama produk: %w", err)
	}
	return &img, nil
}

func (r *productImageRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM inv_product_images WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("gagal menghapus foto produk: %w", err)
	}
	return nil
}

func (r *productImageRepository) SetPrimary(ctx context.Context, productID, imageID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi set primary foto: %w", err)
	}
	defer tx.Rollback()

	// 1. Reset seluruh is_primary untuk produk ini
	if _, err := tx.ExecContext(ctx, `UPDATE inv_product_images SET is_primary = FALSE WHERE product_id = ?`, productID); err != nil {
		return fmt.Errorf("gagal mereset status primary foto lama: %w", err)
	}

	// 2. Set foto terpilih menjadi is_primary = TRUE
	res, err := tx.ExecContext(ctx, `UPDATE inv_product_images SET is_primary = TRUE WHERE id = ? AND product_id = ?`, imageID, productID)
	if err != nil {
		return fmt.Errorf("gagal mengaktifkan status primary foto baru: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("foto tidak ditemukan pada produk ini")
	}

	return tx.Commit()
}

func (r *productImageRepository) CountByProductID(ctx context.Context, productID string) (int, error) {
	query := `SELECT COUNT(*) FROM inv_product_images WHERE product_id = ?`
	var count int
	err := r.db.QueryRowContext(ctx, query, productID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("gagal menghitung foto produk: %w", err)
	}
	return count, nil
}
