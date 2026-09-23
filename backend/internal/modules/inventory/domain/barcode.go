package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"
)

var (
	ErrInvalidBarcodeID      = errors.New("ID barcode tidak valid")
	ErrInvalidBarcodeCode    = errors.New("kode barcode tidak boleh kosong dan harus antara 3 hingga 50 karakter")
	ErrBarcodeHasInvalidChar = errors.New("kode barcode mengandung karakter yang tidak diizinkan (hanya alfanumerik, '-', '_', '.')")
	ErrDuplicateBarcode      = errors.New("kode barcode sudah terdaftar di sistem")
	ErrBarcodeNotFound       = errors.New("barcode tidak ditemukan")
)

// ProductBarcode merepresentasikan satu barcode fisik pabrik yang menempel pada kemasan produk (Child Entity).
// Satu produk dapat memiliki banyak barcode (misalnya beda batch pabrik atau kemasan baru).
type ProductBarcode struct {
	ID        string    // Primary key (UUIDv7)
	ProductID string    // Foreign Key ke inv_products(id)
	Barcode   string    // Kode barcode internasional (EAN-13, UPC, dll)
	IsPrimary bool      // Menandai apakah ini barcode utama produk
	CreatedAt time.Time // Waktu didaftarkan
}

// NewProductBarcode adalah factory function untuk membuat entitas barcode baru yang tervalidasi.
func NewProductBarcode(id, productID, barcode string, isPrimary bool) (*ProductBarcode, error) {
	if id == "" {
		return nil, ErrInvalidBarcodeID
	}
	if productID == "" {
		return nil, ErrInvalidProductID
	}

	cleanCode := strings.TrimSpace(barcode)
	if len(cleanCode) < 3 || len(cleanCode) > 50 {
		return nil, ErrInvalidBarcodeCode
	}

	for _, r := range cleanCode {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '.' {
			return nil, ErrBarcodeHasInvalidChar
		}
	}

	return &ProductBarcode{
		ID:        id,
		ProductID: productID,
		Barcode:   cleanCode,
		IsPrimary: isPrimary,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// SetPrimary mengubah status apakah barcode ini adalah barcode utama produk.
func (b *ProductBarcode) SetPrimary(isPrimary bool) {
	b.IsPrimary = isPrimary
}
