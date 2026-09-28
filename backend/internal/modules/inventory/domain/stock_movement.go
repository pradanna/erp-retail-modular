package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidMovementID      = errors.New("ID pergerakan stok tidak valid")
	ErrInvalidMovementNumber  = errors.New("nomor dokumen pergerakan stok tidak boleh kosong")
	ErrInvalidMovementType    = errors.New("tipe pergerakan stok harus 'in' (masuk) atau 'out' (keluar)")
	ErrEmptyMovementItems     = errors.New("daftar barang pergerakan stok tidak boleh kosong")
	ErrInvalidMovementItemQty = errors.New("kuantitas pergerakan barang harus lebih dari 0")
	ErrMovementSerialMismatch = errors.New("jumlah nomor seri fisik tidak sesuai dengan kuantitas barang")
	ErrMovementItemDuplicate  = errors.New("produk yang sama tidak boleh diinput berulang kali dalam satu dokumen")
)

// StockMovementType mendefinisikan arah pergerakan fisik stok barang.
type StockMovementType string

const (
	MovementTypeIn  StockMovementType = "in"  // Barang Masuk (Inbound / Stock In)
	MovementTypeOut StockMovementType = "out" // Barang Keluar (Outbound / Stock Out)
)

func (t StockMovementType) IsValid() bool {
	return t == MovementTypeIn || t == MovementTypeOut
}

// StockMovementItem merepresentasikan satu baris detail barang dalam transaksi barang masuk/keluar.
type StockMovementItem struct {
	ID            string    // UUIDv7
	MovementID    string    // Foreign Key ke inv_stock_movements
	ProductID     string    // Foreign Key ke inv_products
	ProductName   string    // Denormalisasi nama produk
	ProductSKU    string    // Denormalisasi SKU produk
	Quantity      int       // Jumlah barang (selalu positif)
	Notes         string    // Keterangan khusus per barang
	SerialNumbers []string  // Daftar nomor seri / IMEI yang dilibatkan
	CreatedAt     time.Time // Waktu pembuatan
}

// StockMovement merepresentasikan dokumen transaksi resmi Barang Masuk atau Barang Keluar mandiri.
type StockMovement struct {
	ID              string               // Primary Key (UUIDv7)
	MovementNumber  string               // Nomor unik dokumen (misal: IN-2026-09-0001 atau OUT-2026-09-0001)
	Type            StockMovementType    // 'in' atau 'out'
	MovementDate    time.Time            // Tanggal efektif transaksi / surat jalan fisik
	LocationID      string               // ID cabang/gudang
	LocationName    string               // Denormalisasi nama cabang/gudang
	CategoryReason  string               // Kategori alasan: saldo_awal, pembelian_non_po, rusak_afkir, sampel_display, dll
	ReferenceNumber string               // Nomor surat jalan vendor / memo internal
	Notes           string               // Catatan umum transaksi
	ExecutedBy      string               // User ID operator pembuat
	ExecutedByName  string               // Nama operator pembuat
	Items           []*StockMovementItem // Rincian daftar barang
	CreatedAt       time.Time            // Waktu transaksi dibuat
}

// NewStockMovementItem membuat satu baris item pergerakan stok tervalidasi.
func NewStockMovementItem(id, productID string, qty int, notes string, serialNumbers []string) (*StockMovementItem, error) {
	if id == "" {
		return nil, ErrInvalidMovementID
	}
	if productID == "" {
		return nil, ErrInvalidProductID
	}
	if qty <= 0 {
		return nil, ErrInvalidMovementItemQty
	}
	if len(serialNumbers) > 0 && len(serialNumbers) != qty {
		return nil, ErrMovementSerialMismatch
	}

	return &StockMovementItem{
		ID:            id,
		ProductID:     productID,
		Quantity:      qty,
		Notes:         strings.TrimSpace(notes),
		SerialNumbers: serialNumbers,
		CreatedAt:     time.Now().UTC(),
	}, nil
}

// NewStockMovement membuat entitas dokumen StockMovement baru yang tervalidasi.
func NewStockMovement(
	id, movementNumber string,
	movType StockMovementType,
	movementDate time.Time,
	locationID, categoryReason, refNumber, notes string,
	executedBy, executedByName string,
	items []*StockMovementItem,
) (*StockMovement, error) {
	if id == "" {
		return nil, ErrInvalidMovementID
	}
	if strings.TrimSpace(movementNumber) == "" {
		return nil, ErrInvalidMovementNumber
	}
	if !movType.IsValid() {
		return nil, ErrInvalidMovementType
	}
	if locationID == "" {
		return nil, ErrInvalidLocationID
	}
	if strings.TrimSpace(categoryReason) == "" {
		return nil, errors.New("kategori atau alasan pergerakan barang wajib diisi")
	}
	if len(items) == 0 {
		return nil, ErrEmptyMovementItems
	}
	if executedByName == "" {
		executedByName = "Staf Gudang"
	}
	if movementDate.IsZero() {
		movementDate = time.Now().UTC()
	}

	// Validasi tidak ada produk duplikat di baris yang berbeda
	seenProducts := make(map[string]bool)
	for _, it := range items {
		if seenProducts[it.ProductID] {
			return nil, ErrMovementItemDuplicate
		}
		seenProducts[it.ProductID] = true
		it.MovementID = id
	}

	return &StockMovement{
		ID:              id,
		MovementNumber:  strings.TrimSpace(movementNumber),
		Type:            movType,
		MovementDate:    movementDate,
		LocationID:      locationID,
		CategoryReason:  strings.TrimSpace(categoryReason),
		ReferenceNumber: strings.TrimSpace(refNumber),
		Notes:           strings.TrimSpace(notes),
		ExecutedBy:      executedBy,
		ExecutedByName:  executedByName,
		Items:           items,
		CreatedAt:       time.Now().UTC(),
	}, nil
}
