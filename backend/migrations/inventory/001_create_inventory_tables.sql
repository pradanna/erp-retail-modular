-- +goose Up
-- ============================================================
-- Migrasi: Tabel dasar modul Inventory (MySQL Version)
-- File: migrations/inventory/001_create_inventory_tables.sql
-- ============================================================

-- 1. Tabel Kategori Produk
CREATE TABLE IF NOT EXISTS inv_categories (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    parent_id VARCHAR(36) NULL,
    image_url VARCHAR(500) NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_inv_categories_parent FOREIGN KEY (parent_id) REFERENCES inv_categories(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. Tabel Produk Utama
CREATE TABLE IF NOT EXISTS inv_products (
    id VARCHAR(36) PRIMARY KEY,
    sku VARCHAR(50) NOT NULL UNIQUE,
    category_id VARCHAR(36) NULL,
    name VARCHAR(255) NOT NULL,
    brand VARCHAR(100) NOT NULL DEFAULT '',
    description TEXT NULL,
    unit VARCHAR(20) NOT NULL DEFAULT 'pcs',
    purchase_price DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    selling_price DECIMAL(15, 2) NOT NULL DEFAULT 0.00,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    is_ppn BOOLEAN NOT NULL DEFAULT FALSE,
    flag_serial_tracking BOOLEAN NOT NULL DEFAULT FALSE,
    weight_gram INT NOT NULL DEFAULT 0,
    atribut_varian JSON NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_inv_products_status (status),
    INDEX idx_inv_products_category (category_id),
    CONSTRAINT fk_inv_products_category FOREIGN KEY (category_id) REFERENCES inv_categories(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. Tabel Lokasi (Gudang / Toko)
CREATE TABLE IF NOT EXISTS inv_locations (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(30) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'physical',
    address TEXT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 4. Tabel Stok per Lokasi
CREATE TABLE IF NOT EXISTS inv_stocks (
    id VARCHAR(36) PRIMARY KEY,
    product_id VARCHAR(36) NOT NULL,
    location_id VARCHAR(36) NOT NULL,
    quantity INT NOT NULL DEFAULT 0,
    reserved_quantity INT NOT NULL DEFAULT 0,
    min_stock INT NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_product_location (product_id, location_id),
    INDEX idx_inv_stocks_product (product_id),
    INDEX idx_inv_stocks_location (location_id),
    CONSTRAINT fk_inv_stocks_product FOREIGN KEY (product_id) REFERENCES inv_products(id) ON DELETE RESTRICT,
    CONSTRAINT fk_inv_stocks_location FOREIGN KEY (location_id) REFERENCES inv_locations(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- +goose Down
DROP TABLE IF EXISTS inv_stocks;
DROP TABLE IF EXISTS inv_locations;
DROP TABLE IF EXISTS inv_products;
DROP TABLE IF EXISTS inv_categories;

