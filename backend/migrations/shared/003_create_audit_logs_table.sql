-- +goose Up
-- ==============================================================================
-- Migrasi: Shared Context - Tabel Log Audit Sistem (Audit Logs)
-- File: migrations/shared/003_create_audit_logs_table.sql
-- ==============================================================================

CREATE TABLE IF NOT EXISTS audit_logs (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NULL,
    user_name VARCHAR(100) NOT NULL DEFAULT 'System',
    user_role VARCHAR(50) NOT NULL DEFAULT 'system',
    action VARCHAR(100) NOT NULL,
    module VARCHAR(50) NOT NULL,
    target_type VARCHAR(50) NULL,
    target_id VARCHAR(36) NULL,
    summary VARCHAR(255) NOT NULL,
    details JSON NULL,
    ip_address VARCHAR(45) NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_audit_created (created_at DESC),
    INDEX idx_audit_module (module),
    INDEX idx_audit_user (user_id),
    INDEX idx_audit_action (action)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
