package domain

import (
	"errors"
	"strings"
	"time"
)

// WarrantyType mendefinisikan penerbit/jenis garansi: Toko fisik sendiri atau Garansi Resmi Pabrik (Distributor).
type WarrantyType string

const (
	WarrantyTypeToko   WarrantyType = "toko"
	WarrantyTypePabrik WarrantyType = "pabrik"
)

// IsValid memvalidasi apakah tipe garansi sesuai dengan enum yang diizinkan.
func (t WarrantyType) IsValid() bool {
	return t == WarrantyTypeToko || t == WarrantyTypePabrik
}

var (
	ErrInvalidPolicyID          = errors.New("ID kebijakan garansi tidak valid")
	ErrInvalidWarrantyName      = errors.New("nama kebijakan garansi tidak boleh kosong")
	ErrInvalidWarrantyType      = errors.New("tipe garansi harus 'toko' atau 'pabrik'")
	ErrInvalidWarrantyDuration  = errors.New("durasi garansi harus memiliki durasi bulan atau hari lebih dari 0")
	ErrPolicyNotFound           = errors.New("kebijakan garansi tidak ditemukan")
	ErrProductWarrantyNotFound  = errors.New("garansi produk tidak ditemukan")
	ErrInvalidProductWarrantyID = errors.New("ID garansi produk tidak valid")
)

// WarrantyPolicy merepresentasikan master kebijakan / kartu garansi standar (template garansi).
// Contoh: "Garansi Toko Tukar Baru 7 Hari", "Garansi Resmi Samsung Indonesia 12 Bulan".
type WarrantyPolicy struct {
	ID                string       // Primary Key (UUIDv7)
	Name              string       // Nama kebijakan garansi
	Type              WarrantyType // "toko" atau "pabrik"
	DurationMonths    int          // Durasi dalam bulan
	DurationDays      int          // Durasi tambahan dalam hari
	Coverage          string       // Cakupan garansi (contoh: "Servis gratis, penggantian sparepart")
	ClaimInstructions string       // Prosedur klaim (contoh: "Bawa invoice asli dan unit ke cabang terdekat")
	IsActive          bool         // Status aktif kebijakan
	CreatedAt         time.Time    // Waktu dibuat
	UpdatedAt         time.Time    // Waktu diperbarui
}

// NewWarrantyPolicy adalah factory function untuk membuat WarrantyPolicy dengan validasi invariant domain.
func NewWarrantyPolicy(id, name string, warrantyType WarrantyType, months, days int, coverage, claimInstructions string) (*WarrantyPolicy, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidPolicyID
	}
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return nil, ErrInvalidWarrantyName
	}
	if !warrantyType.IsValid() {
		return nil, ErrInvalidWarrantyType
	}
	if months < 0 || days < 0 || (months == 0 && days == 0) {
		return nil, ErrInvalidWarrantyDuration
	}

	now := time.Now().UTC()
	return &WarrantyPolicy{
		ID:                id,
		Name:              trimmedName,
		Type:              warrantyType,
		DurationMonths:    months,
		DurationDays:      days,
		Coverage:          strings.TrimSpace(coverage),
		ClaimInstructions: strings.TrimSpace(claimInstructions),
		IsActive:          true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

// CalculateExpiryDate menghitung tanggal kedaluwarsa garansi berdasarkan tanggal mulai (misal tanggal pembelian).
func (p *WarrantyPolicy) CalculateExpiryDate(startDate time.Time) time.Time {
	return startDate.AddDate(0, p.DurationMonths, p.DurationDays)
}

// Deactivate menonaktifkan kebijakan garansi ini.
func (p *WarrantyPolicy) Deactivate() {
	p.IsActive = false
	p.UpdatedAt = time.Now().UTC()
}

// ProductWarranty merepresentasikan relasi penugasan kebijakan garansi pada produk tertentu.
// Invariant Kunci: Maksimal SATU garansi aktif per 'Type' ('toko' atau 'pabrik') per produk.
type ProductWarranty struct {
	ID               string          // Primary Key (UUIDv7)
	ProductID        string          // Foreign Key ke inv_products(id)
	WarrantyPolicyID string          // Foreign Key ke inv_warranty_policies(id)
	Type             WarrantyType    // Denormalisasi tipe garansi ('toko' / 'pabrik') untuk query cepat & constraint
	IsActive         bool            // Status aktif
	CreatedAt        time.Time       // Waktu dibuat
	UpdatedAt        time.Time       // Waktu diperbarui
	Policy           *WarrantyPolicy // Detail kebijakan (opsional diisi saat query)
}

// NewProductWarranty membuat penugasan garansi baru ke suatu produk.
func NewProductWarranty(id, productID, policyID string, warrantyType WarrantyType) (*ProductWarranty, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidProductWarrantyID
	}
	if strings.TrimSpace(productID) == "" {
		return nil, ErrInvalidProductID
	}
	if strings.TrimSpace(policyID) == "" {
		return nil, ErrInvalidPolicyID
	}
	if !warrantyType.IsValid() {
		return nil, ErrInvalidWarrantyType
	}

	now := time.Now().UTC()
	return &ProductWarranty{
		ID:               id,
		ProductID:        productID,
		WarrantyPolicyID: policyID,
		Type:             warrantyType,
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// Deactivate menonaktifkan penugasan garansi ini dari produk.
func (pw *ProductWarranty) Deactivate() {
	pw.IsActive = false
	pw.UpdatedAt = time.Now().UTC()
}
