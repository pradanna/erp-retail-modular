package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"
)

var (
	ErrInvalidSerialID           = errors.New("ID serial unit tidak valid")
	ErrInvalidSerialNumber       = errors.New("nomor seri tidak boleh kosong dan harus antara 3 hingga 100 karakter")
	ErrSerialHasInvalidChar      = errors.New("nomor seri mengandung karakter yang tidak diizinkan (hanya alfanumerik, '-', '_', '/', '.')")
	ErrSerialAlreadySold         = errors.New("unit fisik dengan nomor seri ini sudah terjual")
	ErrSerialNotSold             = errors.New("unit fisik dengan nomor seri ini belum terjual sehingga tidak dapat diretur")
	ErrSerialNotAvailable        = errors.New("unit fisik tidak berstatus tersedia sehingga tidak dapat dipindahkan atau dijual")
	ErrDuplicateSerialNumber     = errors.New("nomor seri sudah terdaftar di sistem")
	ErrSerialNotFound            = errors.New("serial number tidak ditemukan")
	ErrProductNotTrackedBySerial = errors.New("produk ini tidak mengaktifkan pelacakan serial number (flag_serial_tracking = false)")
)

// SerialStatus merepresentasikan siklus hidup unit fisik di toko ritel.
type SerialStatus string

const (
	SerialStatusAvailable SerialStatus = "tersedia" // Ada di etalase/gudang toko
	SerialStatusSold      SerialStatus = "terjual"  // Sudah terjual ke pelanggan
	SerialStatusReturned  SerialStatus = "retur"    // Diretur oleh pelanggan karena rusak/klaim garansi
)

// NormalizeSerialStatus mengubah alias status (misal: 'available' -> 'tersedia') ke format resmi.
func NormalizeSerialStatus(s string) SerialStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "tersedia", "available":
		return SerialStatusAvailable
	case "terjual", "sold":
		return SerialStatusSold
	case "retur", "returned", "defective":
		return SerialStatusReturned
	default:
		return SerialStatus(s)
	}
}

// IsValid memvalidasi apakah status serial sesuai dengan enum yang diizinkan.
func (s SerialStatus) IsValid() bool {
	switch NormalizeSerialStatus(string(s)) {
	case SerialStatusAvailable, SerialStatusSold, SerialStatusReturned:
		return true
	default:
		return false
	}
}

// SerialUnit merepresentasikan 1 unit fisik barang spesifik di gudang/toko (Child Entity dari Product).
type SerialUnit struct {
	ID           string       // Primary Key (UUIDv7)
	ProductID    string       // Foreign Key ke inv_products(id)
	LocationID   string       // Foreign Key ke inv_locations(id)
	SerialNumber string       // Nomor seri unik pabrik / IMEI (contoh: SN-LG-2026-001)
	Status       SerialStatus // Status siklus hidup: 'tersedia', 'terjual', 'retur'
	CreatedAt    time.Time    // Waktu pertama kali dicatat masuk sistem
	UpdatedAt    time.Time    // Waktu status atau lokasi terakhir diperbarui
}

// NewSerialUnit adalah factory function untuk membuat entitas SerialUnit baru yang tervalidasi.
// Unit baru selalu berstatus awal 'tersedia'.
func NewSerialUnit(id, productID, locationID, serialNumber string) (*SerialUnit, error) {
	if id == "" {
		return nil, ErrInvalidSerialID
	}
	if productID == "" {
		return nil, ErrInvalidProductID
	}
	if locationID == "" {
		return nil, ErrInvalidLocationID
	}

	cleanSN := strings.TrimSpace(serialNumber)
	if len(cleanSN) < 3 || len(cleanSN) > 100 {
		return nil, ErrInvalidSerialNumber
	}

	for _, r := range cleanSN {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '/' && r != '.' {
			return nil, ErrSerialHasInvalidChar
		}
	}

	now := time.Now().UTC()
	return &SerialUnit{
		ID:           id,
		ProductID:    productID,
		LocationID:   locationID,
		SerialNumber: cleanSN,
		Status:       SerialStatusAvailable,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// MarkAsSold mengubah status unit fisik menjadi 'terjual'.
// Invariant: Hanya unit yang berstatus 'tersedia' yang boleh dijual.
func (u *SerialUnit) MarkAsSold() error {
	if u.Status != SerialStatusAvailable {
		return ErrSerialAlreadySold
	}
	u.Status = SerialStatusSold
	u.UpdatedAt = time.Now().UTC()
	return nil
}

// MarkAsReturned mengubah status unit fisik menjadi 'retur' (pengembalian barang oleh pembeli).
// Invariant: Hanya unit yang berstatus 'terjual' yang boleh diretur.
func (u *SerialUnit) MarkAsReturned() error {
	if u.Status != SerialStatusSold {
		return ErrSerialNotSold
	}
	u.Status = SerialStatusReturned
	u.UpdatedAt = time.Now().UTC()
	return nil
}

// TransferLocation memindahkan unit fisik ke lokasi cabang/gudang baru.
// Invariant: Hanya unit yang 'tersedia' di toko yang boleh dimutasi antar cabang.
func (u *SerialUnit) TransferLocation(newLocationID string) error {
	if newLocationID == "" {
		return ErrInvalidLocationID
	}
	if u.Status != SerialStatusAvailable {
		return ErrSerialNotAvailable
	}
	u.LocationID = newLocationID
	u.UpdatedAt = time.Now().UTC()
	return nil
}

// MarkAsAvailable mengembalikan unit fisik (misal setelah inspeksi retur/re-stock) menjadi 'tersedia'.
func (u *SerialUnit) MarkAsAvailable() error {
	if u.Status != SerialStatusReturned {
		return errors.New("hanya unit berstatus 'retur' yang dapat dikembalikan menjadi 'tersedia'")
	}
	u.Status = SerialStatusAvailable
	u.UpdatedAt = time.Now().UTC()
	return nil
}

