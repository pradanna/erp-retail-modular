package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

type mysqlLocationRepository struct {
	db *sql.DB
}

// NewLocationRepository membuat implementasi baru LocationRepository berbasis database MySQL.
func NewLocationRepository(db *sql.DB) domain.LocationRepository {
	return &mysqlLocationRepository{db: db}
}

// Save menyimpan data Location baru ke tabel inv_locations.
func (r *mysqlLocationRepository) Save(ctx context.Context, l *domain.Location) error {
	query := `
		INSERT INTO inv_locations (id, code, name, type, address, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	var address sql.NullString
	if l.Address != "" {
		address = sql.NullString{String: l.Address, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		l.ID, l.Code, l.Name, string(l.Type), address, l.IsActive, l.CreatedAt, l.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert location: %w", err)
	}
	return nil
}

// FindByID mencari Location berdasarkan primary key UUID.
func (r *mysqlLocationRepository) FindByID(ctx context.Context, id string) (*domain.Location, error) {
	query := `
		SELECT id, code, name, type, address, is_active, created_at, updated_at
		FROM inv_locations
		WHERE id = ?`

	return r.queryOne(ctx, query, id)
}

// FindByCode mencari Location berdasarkan kode unik (misal: "CAB-BDG").
func (r *mysqlLocationRepository) FindByCode(ctx context.Context, code string) (*domain.Location, error) {
	query := `
		SELECT id, code, name, type, address, is_active, created_at, updated_at
		FROM inv_locations
		WHERE code = ?`

	return r.queryOne(ctx, query, code)
}

// Update memperbarui kode, nama, tipe, alamat, status aktif, dan updated_at pada lokasi yang ada.
func (r *mysqlLocationRepository) Update(ctx context.Context, l *domain.Location) error {
	query := `
		UPDATE inv_locations SET
			code = ?, name = ?, type = ?, address = ?, is_active = ?, updated_at = ?
		WHERE id = ?`

	var address sql.NullString
	if l.Address != "" {
		address = sql.NullString{String: l.Address, Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query,
		l.Code, l.Name, string(l.Type), address, l.IsActive, l.UpdatedAt, l.ID,
	)
	if err != nil {
		return fmt.Errorf("gagal update location: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa rows affected update location: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("lokasi dengan ID %s tidak ditemukan", l.ID)
	}

	return nil
}

// Delete menghapus fisik lokasi (hard delete).
// Query ini akan gagal jika terkena ON DELETE RESTRICT (misal masih memiliki stok barang).
func (r *mysqlLocationRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM inv_locations WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("gagal delete location: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa rows affected delete location: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("lokasi dengan ID %s tidak ditemukan", lID(id))
	}

	return nil
}

// List mengambil daftar seluruh lokasi dengan opsi filter hanya yang aktif.
func (r *mysqlLocationRepository) List(ctx context.Context, activeOnly bool) ([]*domain.Location, error) {
	var query string
	var args []any

	if activeOnly {
		query = `
			SELECT id, code, name, type, address, is_active, created_at, updated_at
			FROM inv_locations
			WHERE is_active = TRUE
			ORDER BY code ASC`
	} else {
		query = `
			SELECT id, code, name, type, address, is_active, created_at, updated_at
			FROM inv_locations
			ORDER BY code ASC`
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal query list locations: %w", err)
	}
	defer rows.Close()

	var locations []*domain.Location
	for rows.Next() {
		var l domain.Location
		var locType string
		var address sql.NullString

		err := rows.Scan(
			&l.ID, &l.Code, &l.Name, &locType, &address, &l.IsActive, &l.CreatedAt, &l.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("gagal scan location row: %w", err)
		}

		l.Type = domain.LocationType(locType)
		if address.Valid {
			l.Address = address.String
		}

		locations = append(locations, &l)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi rows locations: %w", err)
	}

	return locations, nil
}

// queryOne adalah helper internal untuk menghindari duplikasi kode SELECT satu baris.
func (r *mysqlLocationRepository) queryOne(ctx context.Context, query string, arg any) (*domain.Location, error) {
	var l domain.Location
	var locType string
	var address sql.NullString

	err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&l.ID, &l.Code, &l.Name, &locType, &address, &l.IsActive, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found bukan error sistem tak terduga
		}
		return nil, fmt.Errorf("gagal scan location: %w", err)
	}

	l.Type = domain.LocationType(locType)
	if address.Valid {
		l.Address = address.String
	}

	return &l, nil
}

func lID(id string) string {
	return id
}
