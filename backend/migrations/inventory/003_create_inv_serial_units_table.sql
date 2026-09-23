-- +goose Up
-- Tabel Pelacakan Serial Number / IMEI per Unit Fisik (Khusus Elektronik)
CREATE TABLE IF NOT EXISTS inv_serial_units (
    id VARCHAR(36) PRIMARY KEY,
    product_id VARCHAR(36) NOT NULL,
    location_id VARCHAR(36) NOT NULL,
    serial_number VARCHAR(100) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'tersedia',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_inv_serial_units_product (product_id),
    INDEX idx_inv_serial_units_location (location_id),
    INDEX idx_inv_serial_units_status (status),
    INDEX idx_inv_serial_units_sn (serial_number),
    CONSTRAINT fk_inv_serial_units_product FOREIGN KEY (product_id) REFERENCES inv_products(id) ON DELETE RESTRICT,
    CONSTRAINT fk_inv_serial_units_location FOREIGN KEY (location_id) REFERENCES inv_locations(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS inv_serial_units;
