package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

type mysqlStockAdjustmentRepository struct {
	db *sql.DB
}

// NewStockAdjustmentRepository membuat instance baru StockAdjustmentRepository berbasis MySQL.
func NewStockAdjustmentRepository(db *sql.DB) domain.StockAdjustmentRepository {
	return &mysqlStockAdjustmentRepository{db: db}
}

func (r *mysqlStockAdjustmentRepository) Save(ctx context.Context, adj *domain.StockAdjustment) error {
	query := `
		INSERT INTO inv_stock_adjustments (
			id, product_id, location_id, previous_quantity, new_quantity, difference, reason, adjusted_by, adjusted_by_name, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(
		ctx,
		query,
		adj.ID,
		adj.ProductID,
		adj.LocationID,
		adj.PreviousQuantity,
		adj.NewQuantity,
		adj.Difference,
		adj.Reason,
		adj.AdjustedBy,
		adj.AdjustedByName,
		adj.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert stock adjustment: %w", err)
	}

	return nil
}

func (r *mysqlStockAdjustmentRepository) List(ctx context.Context, filter domain.StockAdjustmentFilter) ([]*domain.StockAdjustment, int, error) {
	var whereConditions []string
	var args []any

	if filter.LocationID != "" {
		whereConditions = append(whereConditions, "a.location_id = ?")
		args = append(args, filter.LocationID)
	}
	if filter.ProductID != "" {
		whereConditions = append(whereConditions, "a.product_id = ?")
		args = append(args, filter.ProductID)
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// 1. Total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM inv_stock_adjustments a %s", whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("gagal count stock adjustments: %w", err)
	}

	// 2. Paginated list
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	query := fmt.Sprintf(`
		SELECT 
			a.id, a.product_id, COALESCE(p.name, ''), COALESCE(p.sku, ''),
			a.location_id, COALESCE(l.name, ''),
			a.previous_quantity, a.new_quantity, a.difference,
			a.reason, a.adjusted_by, a.adjusted_by_name, a.created_at
		FROM inv_stock_adjustments a
		LEFT JOIN inv_products p ON p.id = a.product_id
		LEFT JOIN inv_locations l ON l.id = a.location_id
		%s
		ORDER BY a.created_at DESC
		LIMIT ? OFFSET ?`, whereClause)

	queryArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal query stock adjustments: %w", err)
	}
	defer rows.Close()

	var items []*domain.StockAdjustment
	for rows.Next() {
		var item domain.StockAdjustment
		if err := rows.Scan(
			&item.ID,
			&item.ProductID,
			&item.ProductName,
			&item.ProductSKU,
			&item.LocationID,
			&item.LocationName,
			&item.PreviousQuantity,
			&item.NewQuantity,
			&item.Difference,
			&item.Reason,
			&item.AdjustedBy,
			&item.AdjustedByName,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("gagal scan stock adjustment row: %w", err)
		}
		items = append(items, &item)
	}

	return items, total, rows.Err()
}
