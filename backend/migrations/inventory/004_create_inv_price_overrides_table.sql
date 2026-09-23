-- +goose Up
-- Tabel Promo dan Harga Khusus per Cabang / Lokasi (Price Override)
-- Mendukung promo reguler (kuota tidak terbatas / max_quantity = NULL)
-- maupun flash sale / siapa cepat dia dapat (max_quantity > 0).
CREATE TABLE IF NOT EXISTS inv_price_overrides (
    id VARCHAR(36) PRIMARY KEY,
    product_id VARCHAR(36) NOT NULL,
    location_id VARCHAR(36) NOT NULL,
    promotional_price BIGINT NOT NULL,
    max_quantity INT NULL DEFAULT NULL,
    claimed_quantity INT NOT NULL DEFAULT 0,
    start_date DATETIME NOT NULL,
    end_date DATETIME NOT NULL,
    reason VARCHAR(255) NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_inv_price_overrides_prod_loc (product_id, location_id),
    INDEX idx_inv_price_overrides_active (is_active),
    INDEX idx_inv_price_overrides_dates (start_date, end_date),
    CONSTRAINT fk_inv_price_overrides_product FOREIGN KEY (product_id) REFERENCES inv_products(id) ON DELETE CASCADE,
    CONSTRAINT fk_inv_price_overrides_location FOREIGN KEY (location_id) REFERENCES inv_locations(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS inv_price_overrides;
