package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

type mysqlCategoryRepository struct {
	db *sql.DB
}

// NewCategoryRepository membuat implementasi baru CategoryRepository berbasis MySQL.
func NewCategoryRepository(db *sql.DB) domain.CategoryRepository {
	return &mysqlCategoryRepository{db: db}
}

func (r *mysqlCategoryRepository) Save(ctx context.Context, c *domain.Category) error {
	query := `
		INSERT INTO inv_categories (id, name, parent_id, image_url, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	var parentID sql.NullString
	if c.ParentID != nil {
		parentID = sql.NullString{String: *c.ParentID, Valid: true}
	}

	var imageURL sql.NullString
	if c.ImageURL != nil {
		imageURL = sql.NullString{String: *c.ImageURL, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		c.ID, c.Name, parentID, imageURL, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert category: %w", err)
	}
	return nil
}

func (r *mysqlCategoryRepository) FindByID(ctx context.Context, id string) (*domain.Category, error) {
	query := `
		SELECT id, name, parent_id, image_url, created_at, updated_at
		FROM inv_categories
		WHERE id = ?`

	var c domain.Category
	var parentID, imageURL sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.Name, &parentID, &imageURL, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Tidak ditemukan bukan error sistem
		}
		return nil, fmt.Errorf("gagal scan category by ID: %w", err)
	}

	if parentID.Valid {
		c.ParentID = &parentID.String
	}
	if imageURL.Valid {
		c.ImageURL = &imageURL.String
	}

	return &c, nil
}

func (r *mysqlCategoryRepository) Update(ctx context.Context, c *domain.Category) error {
	query := `
		UPDATE inv_categories SET
			name = ?, parent_id = ?, image_url = ?, updated_at = ?
		WHERE id = ?`

	var parentID sql.NullString
	if c.ParentID != nil {
		parentID = sql.NullString{String: *c.ParentID, Valid: true}
	}

	var imageURL sql.NullString
	if c.ImageURL != nil {
		imageURL = sql.NullString{String: *c.ImageURL, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		c.Name, parentID, imageURL, c.UpdatedAt, c.ID,
	)
	if err != nil {
		return fmt.Errorf("gagal update category: %w", err)
	}
	return nil
}

func (r *mysqlCategoryRepository) List(ctx context.Context) ([]*domain.Category, error) {
	query := `
		SELECT id, name, parent_id, image_url, created_at, updated_at
		FROM inv_categories
		ORDER BY name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gagal query list categories: %w", err)
	}
	defer rows.Close()

	var categories []*domain.Category
	for rows.Next() {
		var c domain.Category
		var parentID, imageURL sql.NullString

		if err := rows.Scan(&c.ID, &c.Name, &parentID, &imageURL, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("gagal scan category row: %w", err)
		}

		if parentID.Valid {
			c.ParentID = &parentID.String
		}
		if imageURL.Valid {
			c.ImageURL = &imageURL.String
		}

		categories = append(categories, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi rows categories: %w", err)
	}

	return categories, nil
}

func (r *mysqlCategoryRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM inv_categories WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("gagal delete category: %w", err)
	}
	return nil
}
