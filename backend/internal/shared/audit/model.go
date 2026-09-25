package audit

import (
	"context"
	"time"
)

// AuditLogEntry merepresentasikan satu rekaman jejak aktivitas admin/staf di sistem.
type AuditLogEntry struct {
	ID         string     `json:"id"`
	UserID     *string    `json:"user_id,omitempty"`
	UserName   string     `json:"user_name"`
	UserRole   string     `json:"user_role"`
	Action     string     `json:"action"`
	Module     string     `json:"module"`
	TargetType *string    `json:"target_type,omitempty"`
	TargetID   *string    `json:"target_id,omitempty"`
	Summary    string     `json:"summary"`
	Details    *string    `json:"details,omitempty"` // raw JSON string
	IPAddress  *string    `json:"ip_address,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// AuditFilter adalah kriteria pencarian untuk log audit.
type AuditFilter struct {
	Module    string
	Action    string
	UserID    string
	StartDate *time.Time
	EndDate   *time.Time
	Page      int
	Limit     int
}

// Repository adalah interface untuk persistensi log audit.
type Repository interface {
	Insert(ctx context.Context, entry *AuditLogEntry) error
	List(ctx context.Context, filter AuditFilter) ([]*AuditLogEntry, int, error)
}
