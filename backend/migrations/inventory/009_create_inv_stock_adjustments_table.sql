-- +goose Up
-- ==============================================================================
-- Migrasi: Modul Inventory - Tabel Riwayat Penyesuaian Fisik Stok (Stock Opname)
-- File: migrations/inventory/009_create_inv_stock_adjustments_table.sql
-- ==============================================================================

CREATE TABLE IF NOT EXISTS inv_stock_adjustments (
    id VARCHAR(36) PRIMARY KEY,
    product_id VARCHAR(36) NOT NULL,
    location_id VARCHAR(36) NOT NULL,
    previous_quantity INT NOT NULL,
    new_quantity INT NOT NULL,
    difference INT NOT NULL,
    reason VARCHAR(255) NOT NULL,
    adjusted_by VARCHAR(36) NOT NULL,
    adjusted_by_name VARCHAR(100) NOT NULL DEFAULT 'Staf Toko',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_adj_product (product_id),
    INDEX idx_adj_location (location_id),
    INDEX idx_adj_created (created_at DESC),
    CONSTRAINT fk_adj_product FOREIGN KEY (product_id) REFERENCES inv_products(id) ON DELETE CASCADE,
    CONSTRAINT fk_adj_location FOREIGN KEY (location_id) REFERENCES inv_locations(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS inv_stock_adjustments;
