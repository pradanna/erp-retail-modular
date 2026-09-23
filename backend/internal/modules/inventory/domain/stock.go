package domain

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalidStockID       = errors.New("ID stok tidak valid")
	ErrInvalidProductID     = errors.New("ID produk tidak valid")
	ErrInvalidLocationID    = errors.New("ID lokasi tidak valid")
	ErrNegativeQuantity     = errors.New("kuantitas stok tidak boleh negatif")
	ErrNegativeReserved     = errors.New("kuantitas reservasi tidak boleh negatif")
	ErrNegativeMinStock     = errors.New("minimum stok tidak boleh negatif")
	ErrReservedExceedsStock = errors.New("kuantitas reservasi tidak boleh melebihi kuantitas stok fisik")
	ErrInsufficientStock    = errors.New("stok tersedia tidak mencukupi untuk reservasi")
	ErrInvalidReserveQty    = errors.New("kuantitas mutasi harus lebih dari 0")
	ErrExceedsReservedQty   = errors.New("kuantitas melebihi kuantitas yang sedang direservasi")
)

// StockItem merepresentasikan catatan persediaan fisik untuk 1 produk di 1 lokasi (cabang/gudang).
//
// INVARIANT & ATURAN BISNIS KUNCI:
// 1. Quantity >= 0: Kuantitas fisik di gudang tidak boleh negatif.
// 2. ReservedQuantity >= 0: Reservasi tidak boleh negatif.
// 3. ReservedQuantity <= Quantity: Tidak boleh membooking barang lebih dari stok fisik yang ada.
// 4. AvailableQuantity = Quantity - ReservedQuantity: Stok bebas yang siap dijual oleh kasir.
// 5. MinStock >= 0: Ambang batas minimum untuk memicu peringatan stok menipis (Low Stock Alert).
type StockItem struct {
	ID               string    // Primary key (UUIDv7)
	ProductID        string    // ID produk (relasi ke inv_products)
	LocationID       string    // ID cabang/gudang (relasi ke inv_locations)
	Quantity         int       // Jumlah stok fisik di lokasi
	ReservedQuantity int       // Jumlah stok yang di-booking untuk order berjalan
	MinStock         int       // Batas minimum peringatan stok
	UpdatedAt        time.Time // Waktu pembaruan terakhir
}

// NewStockItem adalah factory function untuk membuat entity StockItem baru yang valid.
func NewStockItem(id, productID, locationID string, quantity, minStock int) (*StockItem, error) {
	if id == "" {
		return nil, ErrInvalidStockID
	}
	if productID == "" {
		return nil, ErrInvalidProductID
	}
	if locationID == "" {
		return nil, ErrInvalidLocationID
	}
	if quantity < 0 {
		return nil, ErrNegativeQuantity
	}
	if minStock < 0 {
		return nil, ErrNegativeMinStock
	}

	return &StockItem{
		ID:               id,
		ProductID:        productID,
		LocationID:       locationID,
		Quantity:         quantity,
		ReservedQuantity: 0,
		MinStock:         minStock,
		UpdatedAt:        time.Now().UTC(),
	}, nil
}

// AvailableQuantity menghitung jumlah stok bebas yang dapat dibeli oleh pelanggan atau diproses kasir.
func (s *StockItem) AvailableQuantity() int {
	available := s.Quantity - s.ReservedQuantity
	if available < 0 {
		return 0
	}
	return available
}

// AdjustQuantity melakukan penyesuaian kuantitas fisik (misal: Stock Opname / Penyesuaian Barang Rusak).
// Menolak jika kuantitas baru negatif atau lebih kecil dari stok yang sedang direservasi.
func (s *StockItem) AdjustQuantity(newQty int) error {
	if newQty < 0 {
		return ErrNegativeQuantity
	}
	if newQty < s.ReservedQuantity {
		return fmt.Errorf("%w: kuantitas baru (%d) lebih kecil dari reservasi aktif (%d)",
			ErrReservedExceedsStock, newQty, s.ReservedQuantity)
	}

	s.Quantity = newQty
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// Reserve mengalokasikan (membooking) sejumlah stok untuk pesanan penjualan atau mutasi keluar.
// Kuantitas fisik belum berkurang, tetapi AvailableQuantity berkurang.
func (s *StockItem) Reserve(qty int) error {
	if qty <= 0 {
		return ErrInvalidReserveQty
	}
	if s.AvailableQuantity() < qty {
		return fmt.Errorf("%w: butuh %d, tersedia %d (fisik: %d, terbooking: %d)",
			ErrInsufficientStock, qty, s.AvailableQuantity(), s.Quantity, s.ReservedQuantity)
	}

	s.ReservedQuantity += qty
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// ReleaseReservation membatalkan pemesanan dan mengembalikan kuantitas yang direservasi ke stok bebas.
// Dipanggil saat order penjualan dibatalkan atau waktu pembayaran kedaluwarsa.
func (s *StockItem) ReleaseReservation(qty int) error {
	if qty <= 0 {
		return ErrInvalidReserveQty
	}
	if qty > s.ReservedQuantity {
		return fmt.Errorf("%w: melepas %d padahal reservasi aktif hanya %d",
			ErrExceedsReservedQty, qty, s.ReservedQuantity)
	}

	s.ReservedQuantity -= qty
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// DeductReserved mengeksekusi pemotongan stok fisik setelah barang yang direservasi benar-benar dikirim/diserahkan.
// Kuantitas fisik berkurang, dan kuantitas reservasi juga berkurang secara bersamaan.
func (s *StockItem) DeductReserved(qty int) error {
	if qty <= 0 {
		return ErrInvalidReserveQty
	}
	if qty > s.ReservedQuantity {
		return fmt.Errorf("%w: memotong %d padahal reservasi hanya %d",
			ErrExceedsReservedQty, qty, s.ReservedQuantity)
	}

	s.Quantity -= qty
	s.ReservedQuantity -= qty
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateMinStock memperbarui ambang batas minimum stok untuk peringatan (Low Stock Alert).
func (s *StockItem) UpdateMinStock(minStock int) error {
	if minStock < 0 {
		return ErrNegativeMinStock
	}

	s.MinStock = minStock
	s.UpdatedAt = time.Now().UTC()
	return nil
}

// IsLowStock memeriksa apakah kuantitas fisik sudah mencapai atau berada di bawah ambang batas minimum.
func (s *StockItem) IsLowStock() bool {
	return s.Quantity <= s.MinStock
}
