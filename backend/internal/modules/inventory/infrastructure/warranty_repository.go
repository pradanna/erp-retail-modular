package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

type mysqlWarrantyPolicyRepository struct {
	db *sql.DB
}

// NewWarrantyPolicyRepository membuat implementasi baru domain.WarrantyPolicyRepository berbasis MySQL.
func NewWarrantyPolicyRepository(db *sql.DB) domain.WarrantyPolicyRepository {
	return &mysqlWarrantyPolicyRepository{db: db}
}

// Save menyimpan master kebijakan garansi baru.
func (r *mysqlWarrantyPolicyRepository) Save(ctx context.Context, policy *domain.WarrantyPolicy) error {
	query := `
		INSERT INTO inv_warranty_policies (
			id, name, type, duration_months, duration_days, coverage, claim_instructions,
			is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		policy.ID, policy.Name, string(policy.Type), policy.DurationMonths, policy.DurationDays,
		policy.Coverage, policy.ClaimInstructions, policy.IsActive, policy.CreatedAt, policy.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert warranty policy: %w", err)
	}
	return nil
}

// Update memperbarui data kebijakan garansi.
func (r *mysqlWarrantyPolicyRepository) Update(ctx context.Context, policy *domain.WarrantyPolicy) error {
	query := `
		UPDATE inv_warranty_policies SET
			name = ?, type = ?, duration_months = ?, duration_days = ?,
			coverage = ?, claim_instructions = ?, is_active = ?, updated_at = ?
		WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query,
		policy.Name, string(policy.Type), policy.DurationMonths, policy.DurationDays,
		policy.Coverage, policy.ClaimInstructions, policy.IsActive, policy.UpdatedAt, policy.ID,
	)
	if err != nil {
		return fmt.Errorf("gagal update warranty policy: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrPolicyNotFound
	}
	return nil
}

// FindByID mencari master kebijakan garansi berdasarkan ID.
func (r *mysqlWarrantyPolicyRepository) FindByID(ctx context.Context, id string) (*domain.WarrantyPolicy, error) {
	query := `
		SELECT id, name, type, duration_months, duration_days,
		       COALESCE(coverage, ''), COALESCE(claim_instructions, ''),
		       is_active, created_at, updated_at
		FROM inv_warranty_policies
		WHERE id = ?`

	var policy domain.WarrantyPolicy
	var pType string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&policy.ID, &policy.Name, &pType, &policy.DurationMonths, &policy.DurationDays,
		&policy.Coverage, &policy.ClaimInstructions, &policy.IsActive, &policy.CreatedAt, &policy.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPolicyNotFound
		}
		return nil, fmt.Errorf("gagal query warranty policy by id: %w", err)
	}
	policy.Type = domain.WarrantyType(pType)
	return &policy, nil
}

// List mengambil seluruh kebijakan garansi sesuai filter.
func (r *mysqlWarrantyPolicyRepository) List(ctx context.Context, warrantyType *domain.WarrantyType, isActiveOnly bool) ([]*domain.WarrantyPolicy, error) {
	query := `
		SELECT id, name, type, duration_months, duration_days,
		       COALESCE(coverage, ''), COALESCE(claim_instructions, ''),
		       is_active, created_at, updated_at
		FROM inv_warranty_policies
		WHERE 1=1`
	var args []interface{}

	if warrantyType != nil {
		query += " AND type = ?"
		args = append(args, string(*warrantyType))
	}
	if isActiveOnly {
		query += " AND is_active = TRUE"
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("gagal list warranty policies: %w", err)
	}
	defer rows.Close()

	var policies []*domain.WarrantyPolicy
	for rows.Next() {
		var policy domain.WarrantyPolicy
		var pType string
		if err := rows.Scan(
			&policy.ID, &policy.Name, &pType, &policy.DurationMonths, &policy.DurationDays,
			&policy.Coverage, &policy.ClaimInstructions, &policy.IsActive, &policy.CreatedAt, &policy.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("gagal scan warranty policy: %w", err)
		}
		policy.Type = domain.WarrantyType(pType)
		policies = append(policies, &policy)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi warranty policies: %w", err)
	}
	return policies, nil
}

type mysqlProductWarrantyRepository struct {
	db *sql.DB
}

// NewProductWarrantyRepository membuat implementasi baru domain.ProductWarrantyRepository berbasis MySQL.
func NewProductWarrantyRepository(db *sql.DB) domain.ProductWarrantyRepository {
	return &mysqlProductWarrantyRepository{db: db}
}

// AssignWarranty menetapkan kebijakan garansi ke produk secara transaksional atomik.
// Invariant Kunci: Menonaktifkan garansi aktif sebelumnya dengan tipe yang sama pada produk tersebut.
func (r *mysqlProductWarrantyRepository) AssignWarranty(ctx context.Context, pw *domain.ProductWarranty) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi assign warranty: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	// 1. Nonaktifkan garansi sebelumnya dengan tipe yang sama untuk produk ini
	deactivateQuery := `
		UPDATE inv_product_warranties
		SET is_active = FALSE, updated_at = ?
		WHERE product_id = ? AND type = ? AND is_active = TRUE`

	if _, err := tx.ExecContext(ctx, deactivateQuery, now, pw.ProductID, string(pw.Type)); err != nil {
		return fmt.Errorf("gagal menonaktifkan garansi sebelumnya: %w", err)
	}

	// 2. Simpan penetapan garansi baru yang aktif
	insertQuery := `
		INSERT INTO inv_product_warranties (
			id, product_id, warranty_policy_id, type, is_active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`

	if _, err := tx.ExecContext(ctx, insertQuery,
		pw.ID, pw.ProductID, pw.WarrantyPolicyID, string(pw.Type), pw.IsActive, pw.CreatedAt, pw.UpdatedAt,
	); err != nil {
		return fmt.Errorf("gagal insert product warranty: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit transaksi assign warranty: %w", err)
	}
	return nil
}

// FindByID mencari penugasan garansi produk berdasarkan ID.
func (r *mysqlProductWarrantyRepository) FindByID(ctx context.Context, id string) (*domain.ProductWarranty, error) {
	query := `
		SELECT pw.id, pw.product_id, pw.warranty_policy_id, pw.type, pw.is_active, pw.created_at, pw.updated_at,
		       wp.id, wp.name, wp.type, wp.duration_months, wp.duration_days,
		       COALESCE(wp.coverage, ''), COALESCE(wp.claim_instructions, ''), wp.is_active, wp.created_at, wp.updated_at
		FROM inv_product_warranties pw
		LEFT JOIN inv_warranty_policies wp ON pw.warranty_policy_id = wp.id
		WHERE pw.id = ?`

	var pw domain.ProductWarranty
	var pwType string
	var polID, polName, polType, polCov, polClaim sql.NullString
	var polMonths, polDays sql.NullInt64
	var polActive sql.NullBool
	var polCreated, polUpdated sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&pw.ID, &pw.ProductID, &pw.WarrantyPolicyID, &pwType, &pw.IsActive, &pw.CreatedAt, &pw.UpdatedAt,
		&polID, &polName, &polType, &polMonths, &polDays, &polCov, &polClaim, &polActive, &polCreated, &polUpdated,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrProductWarrantyNotFound
		}
		return nil, fmt.Errorf("gagal query product warranty by id: %w", err)
	}
	pw.Type = domain.WarrantyType(pwType)

	if polID.Valid {
		pw.Policy = &domain.WarrantyPolicy{
			ID:                polID.String,
			Name:              polName.String,
			Type:              domain.WarrantyType(polType.String),
			DurationMonths:    int(polMonths.Int64),
			DurationDays:      int(polDays.Int64),
			Coverage:          polCov.String,
			ClaimInstructions: polClaim.String,
			IsActive:          polActive.Bool,
			CreatedAt:         polCreated.Time,
			UpdatedAt:         polUpdated.Time,
		}
	}

	return &pw, nil
}

// FindActiveByProduct mengambil seluruh garansi aktif milik produk tertentu.
func (r *mysqlProductWarrantyRepository) FindActiveByProduct(ctx context.Context, productID string) ([]*domain.ProductWarranty, error) {
	query := `
		SELECT pw.id, pw.product_id, pw.warranty_policy_id, pw.type, pw.is_active, pw.created_at, pw.updated_at,
		       wp.id, wp.name, wp.type, wp.duration_months, wp.duration_days,
		       COALESCE(wp.coverage, ''), COALESCE(wp.claim_instructions, ''), wp.is_active, wp.created_at, wp.updated_at
		FROM inv_product_warranties pw
		LEFT JOIN inv_warranty_policies wp ON pw.warranty_policy_id = wp.id
		WHERE pw.product_id = ? AND pw.is_active = TRUE
		ORDER BY pw.type ASC`

	rows, err := r.db.QueryContext(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("gagal query active product warranties: %w", err)
	}
	defer rows.Close()

	var results []*domain.ProductWarranty
	for rows.Next() {
		var pw domain.ProductWarranty
		var pwType string
		var polID, polName, polType, polCov, polClaim sql.NullString
		var polMonths, polDays sql.NullInt64
		var polActive sql.NullBool
		var polCreated, polUpdated sql.NullTime

		if err := rows.Scan(
			&pw.ID, &pw.ProductID, &pw.WarrantyPolicyID, &pwType, &pw.IsActive, &pw.CreatedAt, &pw.UpdatedAt,
			&polID, &polName, &polType, &polMonths, &polDays, &polCov, &polClaim, &polActive, &polCreated, &polUpdated,
		); err != nil {
			return nil, fmt.Errorf("gagal scan active product warranty: %w", err)
		}
		pw.Type = domain.WarrantyType(pwType)

		if polID.Valid {
			pw.Policy = &domain.WarrantyPolicy{
				ID:                polID.String,
				Name:              polName.String,
				Type:              domain.WarrantyType(polType.String),
				DurationMonths:    int(polMonths.Int64),
				DurationDays:      int(polDays.Int64),
				Coverage:          polCov.String,
				ClaimInstructions: polClaim.String,
				IsActive:          polActive.Bool,
				CreatedAt:         polCreated.Time,
				UpdatedAt:         polUpdated.Time,
			}
		}
		results = append(results, &pw)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterasi active product warranties: %w", err)
	}
	return results, nil
}

// FindActiveByProductAndType mencari garansi aktif spesifik untuk produk dan tipe tertentu.
func (r *mysqlProductWarrantyRepository) FindActiveByProductAndType(ctx context.Context, productID string, warrantyType domain.WarrantyType) (*domain.ProductWarranty, error) {
	query := `
		SELECT pw.id, pw.product_id, pw.warranty_policy_id, pw.type, pw.is_active, pw.created_at, pw.updated_at,
		       wp.id, wp.name, wp.type, wp.duration_months, wp.duration_days,
		       COALESCE(wp.coverage, ''), COALESCE(wp.claim_instructions, ''), wp.is_active, wp.created_at, wp.updated_at
		FROM inv_product_warranties pw
		LEFT JOIN inv_warranty_policies wp ON pw.warranty_policy_id = wp.id
		WHERE pw.product_id = ? AND pw.type = ? AND pw.is_active = TRUE
		LIMIT 1`

	var pw domain.ProductWarranty
	var pwType string
	var polID, polName, polType, polCov, polClaim sql.NullString
	var polMonths, polDays sql.NullInt64
	var polActive sql.NullBool
	var polCreated, polUpdated sql.NullTime

	err := r.db.QueryRowContext(ctx, query, productID, string(warrantyType)).Scan(
		&pw.ID, &pw.ProductID, &pw.WarrantyPolicyID, &pwType, &pw.IsActive, &pw.CreatedAt, &pw.UpdatedAt,
		&polID, &polName, &polType, &polMonths, &polDays, &polCov, &polClaim, &polActive, &polCreated, &polUpdated,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Tidak ada garansi aktif untuk tipe ini (bukan error fatal)
		}
		return nil, fmt.Errorf("gagal query active product warranty by type: %w", err)
	}
	pw.Type = domain.WarrantyType(pwType)

	if polID.Valid {
		pw.Policy = &domain.WarrantyPolicy{
			ID:                polID.String,
			Name:              polName.String,
			Type:              domain.WarrantyType(polType.String),
			DurationMonths:    int(polMonths.Int64),
			DurationDays:      int(polDays.Int64),
			Coverage:          polCov.String,
			ClaimInstructions: polClaim.String,
			IsActive:          polActive.Bool,
			CreatedAt:         polCreated.Time,
			UpdatedAt:         polUpdated.Time,
		}
	}
	return &pw, nil
}

// Update memperbarui status penugasan garansi produk.
func (r *mysqlProductWarrantyRepository) Update(ctx context.Context, pw *domain.ProductWarranty) error {
	query := `
		UPDATE inv_product_warranties
		SET is_active = ?, updated_at = ?
		WHERE id = ?`

	res, err := r.db.ExecContext(ctx, query, pw.IsActive, pw.UpdatedAt, pw.ID)
	if err != nil {
		return fmt.Errorf("gagal update product warranty: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrProductWarrantyNotFound
	}
	return nil
}
