package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidOverrideID   = errors.New("ID price override tidak valid")
	ErrInvalidPromoPrice    = errors.New("harga promo harus lebih dari 0")
	ErrInvalidPromoDates    = errors.New("tanggal berakhir promo harus setelah tanggal mulai")
	ErrOverlappingPromo     = errors.New("sudah ada promo aktif lain pada produk dan cabang tersebut di rentang tanggal yang sama")
	ErrPromoNotFound        = errors.New("promo harga khusus tidak ditemukan")
	ErrInvalidPromoQuota    = errors.New("kuota promo harus lebih dari 0")
	ErrPromoQuotaExhausted  = errors.New("kuota promo telah habis")
)

// PriceOverride merepresentasikan penyesuaian harga jual khusus (promo) untuk produk pada cabang/lokasi tertentu.
type PriceOverride struct {
	ID               string    // Primary Key (UUIDv7)
	ProductID        string    // Foreign Key ke inv_products(id)
	LocationID       string    // Foreign Key ke inv_locations(id)
	PromotionalPrice int64     // Harga jual promo dalam rupiah bulat
	MaxQuantity      *int      // Batas kuota promo (nil = tidak terbatas/unlimited, >0 = flash sale)
	ClaimedQuantity  int       // Jumlah kuota promo yang telah terjual/diklaim
	StartDate        time.Time // Waktu dimulainya promo
	EndDate          time.Time // Waktu berakhirnya promo
	Reason           string    // Alasan promo (misal: "Promo Grand Opening", "Flash Sale")
	IsActive         bool      // Status apakah promo ini aktif
	CreatedAt        time.Time // Waktu dibuat
	UpdatedAt        time.Time // Waktu diperbarui
}

// NewPriceOverride adalah factory function untuk membuat entitas PriceOverride baru dengan validasi invariant.
func NewPriceOverride(id, productID, locationID string, price int64, start, end time.Time, reason string, maxQuantity *int) (*PriceOverride, error) {
	if id == "" {
		return nil, ErrInvalidOverrideID
	}
	if productID == "" {
		return nil, ErrInvalidProductID
	}
	if locationID == "" {
		return nil, ErrInvalidLocationID
	}
	if price <= 0 {
		return nil, ErrInvalidPromoPrice
	}
	if !end.After(start) {
		return nil, ErrInvalidPromoDates
	}
	if maxQuantity != nil && *maxQuantity <= 0 {
		return nil, ErrInvalidPromoQuota
	}

	now := time.Now().UTC()
	return &PriceOverride{
		ID:               id,
		ProductID:        productID,
		LocationID:       locationID,
		PromotionalPrice: price,
		MaxQuantity:      maxQuantity,
		ClaimedQuantity:  0,
		StartDate:        start.UTC(),
		EndDate:          end.UTC(),
		Reason:           strings.TrimSpace(reason),
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

// IsActiveAt memeriksa apakah promo sedang aktif berlaku pada waktu t tertentu (termasuk kuota tersisa).
func (o *PriceOverride) IsActiveAt(t time.Time) bool {
	if !o.IsActive {
		return false
	}
	tUTC := t.UTC()
	if tUTC.Before(o.StartDate) || tUTC.After(o.EndDate) {
		return false
	}
	if o.MaxQuantity != nil && o.ClaimedQuantity >= *o.MaxQuantity {
		return false // Kuota telah habis!
	}
	return true
}

// RemainingQuota mengembalikan sisa kuota yang belum diklaim, atau nil jika tidak terbatas.
func (o *PriceOverride) RemainingQuota() *int {
	if o.MaxQuantity == nil {
		return nil
	}
	rem := *o.MaxQuantity - o.ClaimedQuantity
	if rem < 0 {
		rem = 0
	}
	return &rem
}

// ClaimQuota menambah jumlah kuota yang telah diklaim saat transaksi kasir.
func (o *PriceOverride) ClaimQuota(qty int) error {
	if qty <= 0 {
		return errors.New("jumlah klaim kuota harus lebih dari 0")
	}
	if o.MaxQuantity != nil && o.ClaimedQuantity+qty > *o.MaxQuantity {
		return ErrPromoQuotaExhausted
	}
	o.ClaimedQuantity += qty
	o.UpdatedAt = time.Now().UTC()
	return nil
}

// Deactivate menonaktifkan promo ini secara manual sebelum tanggal berakhirnya.
func (o *PriceOverride) Deactivate() {
	o.IsActive = false
	o.UpdatedAt = time.Now().UTC()
}

// OverlapsWith memeriksa apakah rentang waktu [start, end] bertabrakan dengan masa promo ini.
// Rumus tabrakan interval: StartA < EndB AND EndA > StartB.
func (o *PriceOverride) OverlapsWith(start, end time.Time) bool {
	startUTC := start.UTC()
	endUTC := end.UTC()
	return startUTC.Before(o.EndDate) && endUTC.After(o.StartDate)
}
