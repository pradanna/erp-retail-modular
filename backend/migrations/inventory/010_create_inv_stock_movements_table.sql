-- +goose Up
-- ==============================================================================
-- Migrasi: Modul Inventory - Tabel Transaksi Barang Masuk & Keluar (Stock Movements)
-- File: migrations/inventory/010_create_inv_stock_movements_table.sql
-- ==============================================================================

-- 1. Tabel Header Transaksi Pergerakan Stok (Barang Masuk / Keluar)
CREATE TABLE IF NOT EXISTS inv_stock_movements (
    id VARCHAR(36) PRIMARY KEY,
    movement_number VARCHAR(50) NOT NULL UNIQUE,
    type VARCHAR(10) NOT NULL, -- 'in' (Barang Masuk) atau 'out' (Barang Keluar)
    movement_date DATE NOT NULL DEFAULT (CURRENT_DATE), -- Tanggal dokumen / surat jalan fisik
    location_id VARCHAR(36) NOT NULL,
    category_reason VARCHAR(100) NOT NULL, -- 'saldo_awal', 'pembelian_non_po', 'retur_konsumen', 'bonus_supplier', 'rusak_afkir', 'sampel_display', 'pemakaian_internal', 'penjualan_non_pos', dll
    reference_number VARCHAR(100) NULL, -- No surat jalan supplier / memo pengeluaran internal
    notes TEXT NULL,
    executed_by VARCHAR(36) NOT NULL,
    executed_by_name VARCHAR(100) NOT NULL DEFAULT 'Staf Toko',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_inv_movement_location (location_id, type),
    INDEX idx_inv_movement_type (type),
    INDEX idx_inv_movement_date (movement_date),
    INDEX idx_inv_movement_created (created_at DESC),
    CONSTRAINT fk_inv_movement_location FOREIGN KEY (location_id) REFERENCES inv_locations(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. Tabel Detail Barang dalam Pergerakan Stok (Movement Items)
CREATE TABLE IF NOT EXISTS inv_stock_movement_items (
    id VARCHAR(36) PRIMARY KEY,
    movement_id VARCHAR(36) NOT NULL,
    product_id VARCHAR(36) NOT NULL,
    quantity INT NOT NULL,
    notes VARCHAR(255) NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_inv_mov_items_movement (movement_id),
    INDEX idx_inv_mov_items_product (product_id),
    CONSTRAINT fk_inv_mov_items_movement FOREIGN KEY (movement_id) REFERENCES inv_stock_movements(id) ON DELETE CASCADE,
    CONSTRAINT fk_inv_mov_items_product FOREIGN KEY (product_id) REFERENCES inv_products(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. Tabel Relasi Unit Berserial / IMEI pada Pergerakan Stok
CREATE TABLE IF NOT EXISTS inv_stock_movement_item_serials (
    id VARCHAR(36) PRIMARY KEY,
    movement_item_id VARCHAR(36) NOT NULL,
    serial_unit_id VARCHAR(36) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_inv_mov_serials_item (movement_item_id),
    INDEX idx_inv_mov_serials_unit (serial_unit_id),
    CONSTRAINT fk_inv_mov_serials_item FOREIGN KEY (movement_item_id) REFERENCES inv_stock_movement_items(id) ON DELETE CASCADE,
    CONSTRAINT fk_inv_mov_serials_unit FOREIGN KEY (serial_unit_id) REFERENCES inv_serial_units(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS inv_stock_movement_item_serials;
DROP TABLE IF EXISTS inv_stock_movement_items;
DROP TABLE IF EXISTS inv_stock_movements;
