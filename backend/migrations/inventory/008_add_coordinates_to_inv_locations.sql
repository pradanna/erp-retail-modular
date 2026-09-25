-- +goose Up
-- Menambahkan kolom koordinat geografis (latitude, longitude) ke tabel inv_locations
-- DECIMAL(10, 8) untuk Latitude (-90 s/d +90 derajat)
-- DECIMAL(11, 8) untuk Longitude (-180 s/d +180 derajat)
ALTER TABLE inv_locations
    ADD COLUMN latitude DECIMAL(10, 8) NULL AFTER address,
    ADD COLUMN longitude DECIMAL(11, 8) NULL AFTER latitude;

-- +goose Down
ALTER TABLE inv_locations
    DROP COLUMN longitude,
    DROP COLUMN latitude;
