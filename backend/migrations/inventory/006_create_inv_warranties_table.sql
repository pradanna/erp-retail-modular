-- +goose Up
-- ==============================================================================
-- Migrasi: Modul Inventory - Tabel Kebijakan Garansi Toko & Pabrik
-- File: migrations/inventory/006_create_inv_warranties_table.sql
-- ==============================================================================

-- 1. Tabel Master Kebijakan Garansi (Template)
CREATE TABLE IF NOT EXISTS inv_warranty_policies (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    duration_months INT NOT NULL DEFAULT 0,
    duration_days INT NOT NULL DEFAULT 0,
    coverage TEXT NULL,
    claim_instructions TEXT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_inv_warranty_policies_type (type, is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. Tabel Penetapan Garansi ke Produk (Product Warranty)
CREATE TABLE IF NOT EXISTS inv_product_warranties (
    id VARCHAR(36) PRIMARY KEY,
    product_id VARCHAR(36) NOT NULL,
    warranty_policy_id VARCHAR(36) NOT NULL,
    type VARCHAR(20) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_inv_product_warranties_lookup (product_id, type, is_active),
    CONSTRAINT fk_inv_product_warranties_product FOREIGN KEY (product_id) REFERENCES inv_products(id) ON DELETE CASCADE,
    CONSTRAINT fk_inv_product_warranties_policy FOREIGN KEY (warranty_policy_id) REFERENCES inv_warranty_policies(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS inv_product_warranties;
DROP TABLE IF EXISTS inv_warranty_policies;
