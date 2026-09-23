package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

type mysqlSerialUnitRepository struct {
	db *sql.DB
}

// NewSerialUnitRepository membuat implementasi baru domain.SerialUnitRepository berbasis database MySQL.
func NewSerialUnitRepository(db *sql.DB) domain.SerialUnitRepository {
	return &mysqlSerialUnitRepository{db: db}
}

// Save menyimpan 1 entitas serial unit ke tabel inv_serial_units.
func (r *mysqlSerialUnitRepository) Save(ctx context.Context, u *domain.SerialUnit) error {
	query := `
		INSERT INTO inv_serial_units (id, product_id, location_id, serial_number, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		u.ID, u.ProductID, u.LocationID, u.SerialNumber, string(u.Status), u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			return domain.ErrDuplicateSerialNumber
		}
		return fmt.Errorf("gagal insert serial unit: %w", err)
	}
	return nil
}

// BatchSave menyimpan banyak entitas serial unit dalam satu transaksi atomik.
func (r *mysqlSerialUnitRepository) BatchSave(ctx context.Context, units []*domain.SerialUnit) error {
	if len(units) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi batch insert serial: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO inv_serial_units (id, product_id, location_id, serial_number, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("gagal prepare statement batch insert serial: %w", err)
	}
	defer stmt.Close()

	for _, u := range units {
		_, err := stmt.ExecContext(ctx,
			u.ID, u.ProductID, u.LocationID, u.SerialNumber, string(u.Status), u.CreatedAt, u.UpdatedAt,
		)
		if err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") {
				return fmt.Errorf("%w: %s", domain.ErrDuplicateSerialNumber, u.SerialNumber)
			}
			return fmt.Errorf("gagal batch insert serial unit (%s): %w", u.SerialNumber, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit batch insert serial: %w", err)
	}

	return nil
}

// Update memperbarui status atau lokasi unit fisik.
func (r *mysqlSerialUnitRepository) Update(ctx context.Context, u *domain.SerialUnit) error {
	query := `
		UPDATE inv_serial_units SET
			location_id = ?, status = ?, updated_at = ?
		WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query,
		u.LocationID, string(u.Status), u.UpdatedAt, u.ID,
	)
	if err != nil {
		return fmt.Errorf("gagal update serial unit: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa rows affected update serial unit: %w", err)
	}
	if rows == 0 {
		return domain.ErrSerialNotFound
	}

	return nil
}

// FindByID mencari unit fisik berdasarkan UUID.
func (r *mysqlSerialUnitRepository) FindByID(ctx context.Context, id string) (*domain.SerialUnit, error) {
	query := `
		SELECT id, product_id, location_id, serial_number, status, created_at, updated_at
		FROM inv_serial_units
		WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	return r.scanSerialUnit(row)
}

// FindBySerialNumber mencari unit fisik berdasarkan serial number unik (Fast Lookup untuk scanner kasir).
func (r *mysqlSerialUnitRepository) FindBySerialNumber(ctx context.Context, sn string) (*domain.SerialUnit, error) {
	query := `
		SELECT id, product_id, location_id, serial_number, status, created_at, updated_at
		FROM inv_serial_units
		WHERE serial_number = ?`

	row := r.db.QueryRowContext(ctx, query, strings.TrimSpace(sn))
	return r.scanSerialUnit(row)
}

// ListByProduct mengambil daftar serial unit milik suatu produk.
func (r *mysqlSerialUnitRepository) ListByProduct(ctx context.Context, productID string, status *domain.SerialStatus) ([]*domain.SerialUnit, error) {
	query := `
		SELECT id, product_id, location_id, serial_number, status, created_at, updated_at
		FROM inv_serial_units
		WHERE product_id = ?`
	args := []any{productID}

	if status != nil {
		query += ` AND status = ?`
		args = append(args, string(*status))
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal query list serial by product: %w", err)
	}
	defer rows.Close()

	return r.scanSerialUnits(rows)
}

// ListByLocation mengambil daftar serial unit pada lokasi tertentu.
func (r *mysqlSerialUnitRepository) ListByLocation(ctx context.Context, locationID string, status *domain.SerialStatus) ([]*domain.SerialUnit, error) {
	query := `
		SELECT id, product_id, location_id, serial_number, status, created_at, updated_at
		FROM inv_serial_units
		WHERE location_id = ?`
	args := []any{locationID}

	if status != nil {
		query += ` AND status = ?`
		args = append(args, string(*status))
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal query list serial by location: %w", err)
	}
	defer rows.Close()

	return r.scanSerialUnits(rows)
}

// Helper untuk membaca 1 baris hasil scan row
func (r *mysqlSerialUnitRepository) scanSerialUnit(scanner interface{ Scan(dest ...any) error }) (*domain.SerialUnit, error) {
	var (
		id, productID, locationID, sn, statusStr string
		createdAt, updatedAt                    time.Time
	)

	err := scanner.Scan(&id, &productID, &locationID, &sn, &statusStr, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSerialNotFound
		}
		return nil, fmt.Errorf("gagal scan serial unit: %w", err)
	}

	return &domain.SerialUnit{
		ID:           id,
		ProductID:    productID,
		LocationID:   locationID,
		SerialNumber: sn,
		Status:       domain.SerialStatus(statusStr),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

// Helper untuk membaca banyak baris hasil query rows
func (r *mysqlSerialUnitRepository) scanSerialUnits(rows *sql.Rows) ([]*domain.SerialUnit, error) {
	var units []*domain.SerialUnit
	for rows.Next() {
		var (
			id, productID, locationID, sn, statusStr string
			createdAt, updatedAt                    time.Time
		)
		if err := rows.Scan(&id, &productID, &locationID, &sn, &statusStr, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("gagal scan row serial unit: %w", err)
		}
		units = append(units, &domain.SerialUnit{
			ID:           id,
			ProductID:    productID,
			LocationID:   locationID,
			SerialNumber: sn,
			Status:       domain.SerialStatus(statusStr),
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi rows serial unit: %w", err)
	}

	return units, nil
}
