-- +goose Up
-- Tabel Multi-Barcode Pabrik per Produk
CREATE TABLE IF NOT EXISTS inv_barcodes (
    id VARCHAR(36) PRIMARY KEY,
    product_id VARCHAR(36) NOT NULL,
    barcode VARCHAR(50) NOT NULL UNIQUE,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_inv_barcodes_product (product_id),
    INDEX idx_inv_barcodes_code (barcode),
    CONSTRAINT fk_inv_barcodes_product FOREIGN KEY (product_id) REFERENCES inv_products(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS inv_barcodes;
