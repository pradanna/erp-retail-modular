package audit

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/erp-retail/backend/pkg/uid"
)

type mysqlAuditRepository struct {
	db *sql.DB
}

// NewRepository membuat instance baru audit Repository berbasis MySQL.
func NewRepository(db *sql.DB) Repository {
	return &mysqlAuditRepository{db: db}
}

func (r *mysqlAuditRepository) Insert(ctx context.Context, entry *AuditLogEntry) error {
	if entry.ID == "" {
		entry.ID = uid.New()
	}

	query := `
		INSERT INTO audit_logs (
			id, user_id, user_name, user_role, action, module, target_type, target_id, summary, details, ip_address, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	createdAt := entry.CreatedAt
	if createdAt.IsZero() {
		createdAt = entry.CreatedAt
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		entry.ID,
		entry.UserID,
		entry.UserName,
		entry.UserRole,
		entry.Action,
		entry.Module,
		entry.TargetType,
		entry.TargetID,
		entry.Summary,
		entry.Details,
		entry.IPAddress,
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("gagal insert audit log: %w", err)
	}

	return nil
}

func (r *mysqlAuditRepository) List(ctx context.Context, filter AuditFilter) ([]*AuditLogEntry, int, error) {
	var whereConditions []string
	var args []any

	if filter.Module != "" {
		whereConditions = append(whereConditions, "module = ?")
		args = append(args, filter.Module)
	}
	if filter.Action != "" {
		whereConditions = append(whereConditions, "action = ?")
		args = append(args, filter.Action)
	}
	if filter.UserID != "" {
		whereConditions = append(whereConditions, "user_id = ?")
		args = append(args, filter.UserID)
	}
	if filter.StartDate != nil {
		whereConditions = append(whereConditions, "created_at >= ?")
		args = append(args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		whereConditions = append(whereConditions, "created_at <= ?")
		args = append(args, *filter.EndDate)
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// 1. Hitung total records
	countQuery := "SELECT COUNT(*) FROM audit_logs " + whereClause
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("gagal count audit logs: %w", err)
	}

	// 2. Query paginated records
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
		SELECT id, user_id, user_name, user_role, action, module, target_type, target_id, summary, details, ip_address, created_at
		FROM audit_logs
		%s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, whereClause)

	queryArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal query audit logs: %w", err)
	}
	defer rows.Close()

	var entries []*AuditLogEntry
	for rows.Next() {
		var e AuditLogEntry
		var details, ipAddress sql.NullString
		var targetType, targetID, userID sql.NullString

		if err := rows.Scan(
			&e.ID,
			&userID,
			&e.UserName,
			&e.UserRole,
			&e.Action,
			&e.Module,
			&targetType,
			&targetID,
			&e.Summary,
			&details,
			&ipAddress,
			&e.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("gagal scan audit log row: %w", err)
		}

		if userID.Valid {
			e.UserID = &userID.String
		}
		if targetType.Valid {
			e.TargetType = &targetType.String
		}
		if targetID.Valid {
			e.TargetID = &targetID.String
		}
		if details.Valid {
			e.Details = &details.String
		}
		if ipAddress.Valid {
			e.IPAddress = &ipAddress.String
		}

		entries = append(entries, &e)
	}

	return entries, total, rows.Err()
}
