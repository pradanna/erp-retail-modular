-- +goose Up
-- Tabel Surat Jalan / Header Mutasi Stok Antar Cabang (Stock Transfer)
CREATE TABLE IF NOT EXISTS inv_stock_transfers (
    id VARCHAR(36) PRIMARY KEY,
    transfer_number VARCHAR(50) NOT NULL UNIQUE,
    from_location_id VARCHAR(36) NOT NULL,
    to_location_id VARCHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending_approval',
    notes TEXT NULL,
    rejection_reason VARCHAR(255) NULL,
    requested_by VARCHAR(36) NOT NULL,
    approved_by VARCHAR(36) NULL,
    received_by VARCHAR(36) NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_inv_stock_transfers_from_status (from_location_id, status),
    INDEX idx_inv_stock_transfers_to_status (to_location_id, status),
    INDEX idx_inv_stock_transfers_status (status),
    CONSTRAINT fk_inv_stock_transfers_from FOREIGN KEY (from_location_id) REFERENCES inv_locations(id) ON DELETE RESTRICT,
    CONSTRAINT fk_inv_stock_transfers_to FOREIGN KEY (to_location_id) REFERENCES inv_locations(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Tabel Detail Barang yang Dimutasi (Stock Transfer Items)
CREATE TABLE IF NOT EXISTS inv_stock_transfer_items (
    id VARCHAR(36) PRIMARY KEY,
    transfer_id VARCHAR(36) NOT NULL,
    product_id VARCHAR(36) NOT NULL,
    quantity INT NOT NULL,
    received_quantity INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_inv_transfer_items_transfer (transfer_id),
    INDEX idx_inv_transfer_items_product (product_id),
    CONSTRAINT fk_inv_transfer_items_transfer FOREIGN KEY (transfer_id) REFERENCES inv_stock_transfers(id) ON DELETE CASCADE,
    CONSTRAINT fk_inv_transfer_items_product FOREIGN KEY (product_id) REFERENCES inv_products(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Tabel Keterkaitan Unit Fisik Berserial / IMEI yang Dimutasi
CREATE TABLE IF NOT EXISTS inv_stock_transfer_item_serials (
    id VARCHAR(36) PRIMARY KEY,
    transfer_item_id VARCHAR(36) NOT NULL,
    serial_unit_id VARCHAR(36) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_inv_transfer_serials_item (transfer_item_id),
    INDEX idx_inv_transfer_serials_unit (serial_unit_id),
    CONSTRAINT fk_inv_transfer_serials_item FOREIGN KEY (transfer_item_id) REFERENCES inv_stock_transfer_items(id) ON DELETE CASCADE,
    CONSTRAINT fk_inv_transfer_serials_unit FOREIGN KEY (serial_unit_id) REFERENCES inv_serial_units(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS inv_stock_transfer_item_serials;
DROP TABLE IF EXISTS inv_stock_transfer_items;
DROP TABLE IF EXISTS inv_stock_transfers;
