-- +goose Up
-- Tabel inv_product_images menyimpan metadata dan tautan berkas foto untuk setiap produk.
-- Menghubungkan entity ProductImage ke inv_products dengan penghapusan berantai (CASCADE).
CREATE TABLE IF NOT EXISTS inv_product_images (
    id VARCHAR(36) PRIMARY KEY,
    product_id VARCHAR(36) NOT NULL,
    url VARCHAR(500) NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_inv_product_images_product (product_id),
    INDEX idx_inv_product_images_primary (product_id, is_primary),
    CONSTRAINT fk_inv_product_images_product FOREIGN KEY (product_id) REFERENCES inv_products(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS inv_product_images;
