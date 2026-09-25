package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidAdjustmentID = errors.New("ID penyesuaian stok tidak valid")
	ErrEmptyAdjustmentReason = errors.New("alasan penyesuaian stok tidak boleh kosong")
)

// StockAdjustment merepresentasikan rekaman audit trail setiap kali terjadi Stock Opname / penyesuaian stok fisik.
type StockAdjustment struct {
	ID               string    // Primary key (UUIDv7)
	ProductID        string    // ID produk
	ProductName      string    // Nama produk (denormalisasi/join)
	ProductSKU       string    // SKU produk (denormalisasi/join)
	LocationID       string    // ID lokasi/cabang
	LocationName     string    // Nama lokasi (denormalisasi/join)
	PreviousQuantity int       // Jumlah fisik sebelum opname
	NewQuantity      int       // Jumlah fisik sesudah opname
	Difference       int       // Selisih (NewQuantity - PreviousQuantity)
	Reason           string    // Alasan perubahan stok (misal: "Hasil Stock Opname Rutin", "Barang Hilang", dll)
	AdjustedBy       string    // User ID staf/admin
	AdjustedByName   string    // Nama staf/admin
	CreatedAt        time.Time // Waktu transaksi
}

// NewStockAdjustment membuat entity StockAdjustment baru yang valid.
func NewStockAdjustment(
	id string,
	productID, locationID string,
	previousQty, newQty int,
	reason string,
	adjustedBy, adjustedByName string,
) (*StockAdjustment, error) {
	if id == "" {
		return nil, ErrInvalidAdjustmentID
	}
	if productID == "" {
		return nil, ErrInvalidProductID
	}
	if locationID == "" {
		return nil, ErrInvalidLocationID
	}
	if reason == "" {
		return nil, ErrEmptyAdjustmentReason
	}
	if adjustedByName == "" {
		adjustedByName = "Staf Toko"
	}

	return &StockAdjustment{
		ID:               id,
		ProductID:        productID,
		LocationID:       locationID,
		PreviousQuantity: previousQty,
		NewQuantity:      newQty,
		Difference:       newQty - previousQty,
		Reason:           reason,
		AdjustedBy:       adjustedBy,
		AdjustedByName:   adjustedByName,
		CreatedAt:        time.Now().UTC(),
	}, nil
}
