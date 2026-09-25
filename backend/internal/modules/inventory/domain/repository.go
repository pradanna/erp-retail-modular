package domain

import (
	"context"
	"time"
)

// ProductRepository adalah interface (kontrak) untuk operasi persistence Product.
//
// MENGAPA INTERFACE DI LAYER DOMAIN?
// Ini adalah inti dari Dependency Inversion Principle (DIP):
// - Layer domain MENDEFINISIKAN kontrak apa yang dibutuhkannya (interface ini)
// - Layer infrastructure MEMENUHI kontrak tersebut (implementasi SQL konkret)
// - Sehingga domain tidak tergantung pada infrastruktur, tapi sebaliknya
//
// Manfaat praktis:
// 1. Testing: unit test bisa pakai mock/fake repository tanpa koneksi DB
// 2. Flexibility: bisa ganti PostgreSQL ke MySQL tanpa ubah kode domain/application
// 3. Kejelasan: interface ini adalah "daftar kebutuhan" domain — mudah dibaca
//
// Di Go, interface bersifat "implicit" — struct yang punya method yang cocok
// otomatis memenuhi interface ini tanpa keyword "implements".
type ProductRepository interface {
	// Save menyimpan Product baru ke database.
	// Menggunakan context.Context untuk mendukung timeout, cancellation, dan tracing.
	Save(ctx context.Context, product *Product) error

	// FindByID mencari Product berdasarkan ID (UUIDv7).
	// Mengembalikan nil, nil jika produk tidak ditemukan (bukan error).
	FindByID(ctx context.Context, id string) (*Product, error)

	// FindBySKU mencari Product berdasarkan kode SKU (harus unik).
	FindBySKU(ctx context.Context, sku string) (*Product, error)

	// Update memperbarui data Product yang sudah ada.
	Update(ctx context.Context, product *Product) error

	// List mengambil daftar produk dengan pagination dan filter opsional.
	List(ctx context.Context, filter ProductFilter) ([]*Product, int, error)
}

// ProductFilter adalah parameter opsional untuk filtering dan pagination saat listing produk.
// Menggunakan struct (bukan banyak parameter fungsi) agar mudah diperluas di masa depan
// tanpa mengubah signature method interface.
type ProductFilter struct {
	Status     *ProductStatus // nil berarti ambil semua status
	CategoryID *string        // nil berarti semua kategori
	Search     *string        // nil berarti tidak ada filter pencarian nama/SKU
	Page       int            // 1-indexed
	Limit      int            // jumlah item per halaman
}

// LocationRepository adalah interface (kontrak) untuk operasi persistence Location.
type LocationRepository interface {
	Save(ctx context.Context, location *Location) error
	FindByID(ctx context.Context, id string) (*Location, error)
	FindByCode(ctx context.Context, code string) (*Location, error)
	Update(ctx context.Context, location *Location) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, activeOnly bool) ([]*Location, error)
}

// CategoryRepository adalah interface (kontrak) untuk operasi persistence Category.
type CategoryRepository interface {
	Save(ctx context.Context, category *Category) error
	FindByID(ctx context.Context, id string) (*Category, error)
	Update(ctx context.Context, category *Category) error
	List(ctx context.Context) ([]*Category, error)
	Delete(ctx context.Context, id string) error
}

// StockRepository adalah interface (kontrak) untuk operasi persistence StockItem per cabang/lokasi.
type StockRepository interface {
	// Save menyimpan catatan stok baru ke database.
	Save(ctx context.Context, item *StockItem) error

	// Update memperbarui catatan stok yang ada di database.
	Update(ctx context.Context, item *StockItem) error

	// FindByProductAndLocation mencari stok spesifik berdasarkan kombinasi product_id dan location_id.
	FindByProductAndLocation(ctx context.Context, productID, locationID string) (*StockItem, error)

	// ListByLocation mengambil daftar seluruh stok barang pada suatu lokasi cabang/gudang.
	ListByLocation(ctx context.Context, locationID string) ([]*StockItem, error)

	// ListLowStockAlerts mengambil daftar stok yang kuantitasnya <= min_stock.
	// Jika locationID nil, mengambil dari seluruh cabang.
	ListLowStockAlerts(ctx context.Context, locationID *string) ([]*StockItem, error)

	// AtomicMutate mengeksekusi fungsi perubahan stok secara atomik dengan row-level locking (SELECT ... FOR UPDATE).
	// Jika baris stok belum ada di database, fungsi ini menginisialisasi StockItem baru dengan kuantitas awal 0.
	// Jika mutateFn mengembalikan error, transaksi database di-rollback dan perubahan dibatalkan.
	AtomicMutate(ctx context.Context, productID, locationID string, mutateFn func(item *StockItem) error) (*StockItem, error)
}

// StockAdjustmentFilter adalah kriteria pencarian untuk riwayat penyesuaian stok.
type StockAdjustmentFilter struct {
	LocationID string
	ProductID  string
	Page       int
	Limit      int
}

// StockAdjustmentRepository adalah interface untuk operasi persistence riwayat Stock Opname.
type StockAdjustmentRepository interface {
	Save(ctx context.Context, adj *StockAdjustment) error
	List(ctx context.Context, filter StockAdjustmentFilter) ([]*StockAdjustment, int, error)
}

// BarcodeRepository adalah interface (kontrak) untuk operasi persistence ProductBarcode.
type BarcodeRepository interface {
	// Save menyimpan barcode baru ke database.
	Save(ctx context.Context, barcode *ProductBarcode) error

	// Delete menghapus barcode berdasarkan ID (UUID).
	Delete(ctx context.Context, id string) error

	// ListByProductID mengambil seluruh barcode yang terdaftar pada satu produk.
	ListByProductID(ctx context.Context, productID string) ([]*ProductBarcode, error)

	// FindByID mencari barcode berdasarkan ID baris.
	FindByID(ctx context.Context, id string) (*ProductBarcode, error)

	// FindByBarcode mencari barcode berdasarkan kode uniknya.
	FindByBarcode(ctx context.Context, code string) (*ProductBarcode, error)

	// FindProductByBarcode mencari data produk lengkap beserta info barcode-nya (Fast Lookup untuk kasir).
	FindProductByBarcode(ctx context.Context, code string) (*Product, *ProductBarcode, error)

	// ResetPrimary menonaktifkan status is_primary pada seluruh barcode milik produk tertentu.
	ResetPrimary(ctx context.Context, productID string) error
}

// SerialUnitRepository adalah interface (kontrak) untuk operasi persistence SerialUnit.
type SerialUnitRepository interface {
	// Save menyimpan satu unit serial baru ke database.
	Save(ctx context.Context, unit *SerialUnit) error

	// BatchSave menyimpan banyak unit serial sekaligus (bulk insert) saat barang datang.
	BatchSave(ctx context.Context, units []*SerialUnit) error

	// Update memperbarui status atau lokasi unit fisik yang sudah ada.
	Update(ctx context.Context, unit *SerialUnit) error

	// FindByID mencari unit fisik berdasarkan UUID.
	FindByID(ctx context.Context, id string) (*SerialUnit, error)

	// FindBySerialNumber mencari unit fisik berdasarkan serial number / IMEI unik.
	FindBySerialNumber(ctx context.Context, sn string) (*SerialUnit, error)

	// ListByProduct mengambil seluruh unit fisik milik suatu produk, dengan filter status opsional.
	ListByProduct(ctx context.Context, productID string, status *SerialStatus) ([]*SerialUnit, error)

	// ListByLocation mengambil seluruh unit fisik yang berada di suatu lokasi cabang/gudang.
	ListByLocation(ctx context.Context, locationID string, status *SerialStatus) ([]*SerialUnit, error)

	// List mengambil seluruh unit fisik dengan filter fleksibel (produk, cabang, status, pencarian serial).
	List(ctx context.Context, productID, locationID *string, status *SerialStatus, search *string) ([]*SerialUnit, error)
}

// PriceOverrideRepository adalah interface (kontrak) untuk operasi persistence PriceOverride.
type PriceOverrideRepository interface {
	// Save menyimpan price override baru ke database.
	Save(ctx context.Context, po *PriceOverride) error

	// Update memperbarui data promo (misal: penonaktifan manual atau perpanjangan tanggal).
	Update(ctx context.Context, po *PriceOverride) error

	// FindByID mencari promo berdasarkan UUID.
	FindByID(ctx context.Context, id string) (*PriceOverride, error)

	// FindActive mencari promo yang sedang aktif berlaku pada waktu 'at' untuk produk dan cabang tertentu.
	FindActive(ctx context.Context, productID, locationID string, at time.Time) (*PriceOverride, error)

	// ListByProduct mengambil daftar seluruh promo milik produk tertentu (opsional filter per cabang).
	ListByProduct(ctx context.Context, productID string, locationID *string) ([]*PriceOverride, error)

	// HasOverlappingPromo memeriksa apakah ada promo aktif lain yang rentang tanggalnya bertabrakan.
	// Jika excludeID tidak nil, baris dengan ID tersebut diabaikan (berguna saat update).
	HasOverlappingPromo(ctx context.Context, productID, locationID string, start, end time.Time, excludeID *string) (bool, error)

	// IncrementClaimedQuantity menambah claimed_quantity secara atomik dan aman dari race condition.
	// Mengembalikan ErrPromoQuotaExhausted jika penambahan ini akan melebihi max_quantity.
	IncrementClaimedQuantity(ctx context.Context, id string, delta int) error

	// List mengambil daftar seluruh promo harga khusus dengan filter opsional (produk, cabang, status aktif).
	List(ctx context.Context, productID, locationID *string, isActive *bool) ([]*PriceOverride, error)
}

// StockTransferFilter adalah kriteria pencarian dokumen mutasi stok.
type StockTransferFilter struct {
	FromLocationID *string
	ToLocationID   *string
	Status         *TransferStatus
}

// StockTransferRepository adalah interface (kontrak) untuk operasi persistence StockTransfer.
type StockTransferRepository interface {
	// Save menyimpan dokumen transfer baru beserta detail itemnya secara transaksional.
	Save(ctx context.Context, transfer *StockTransfer) error

	// Update memperbarui status dokumen transfer (approved, in_transit, received, rejected).
	Update(ctx context.Context, transfer *StockTransfer) error

	// FindByID mencari dokumen transfer lengkap dengan item-itemnya berdasarkan UUID.
	FindByID(ctx context.Context, id string) (*StockTransfer, error)

	// FindByNumber mencari dokumen transfer berdasarkan nomor surat jalan (contoh: TRF-202609-0001).
	FindByNumber(ctx context.Context, number string) (*StockTransfer, error)

	// List mengambil daftar transfer berdasarkan kriteria filter (asal, tujuan, status).
	List(ctx context.Context, filter StockTransferFilter) ([]*StockTransfer, error)

	// GenerateTransferNumber membuat nomor urut dokumen transfer baru secara atomik.
	GenerateTransferNumber(ctx context.Context) (string, error)
}

// WarrantyPolicyRepository adalah interface (kontrak) persistence untuk master kebijakan garansi.
type WarrantyPolicyRepository interface {
	// Save menyimpan kebijakan garansi baru.
	Save(ctx context.Context, policy *WarrantyPolicy) error

	// Update memperbarui data kebijakan garansi.
	Update(ctx context.Context, policy *WarrantyPolicy) error

	// FindByID mencari kebijakan garansi berdasarkan ID.
	FindByID(ctx context.Context, id string) (*WarrantyPolicy, error)

	// List mengambil seluruh kebijakan garansi (opsional filter tipe dan status aktif).
	List(ctx context.Context, warrantyType *WarrantyType, isActiveOnly bool) ([]*WarrantyPolicy, error)
}

// ProductWarrantyRepository adalah interface (kontrak) persistence untuk penugasan garansi produk.
type ProductWarrantyRepository interface {
	// AssignWarranty menetapkan garansi ke produk secara atomik.
	// Invariant: Otomatis menonaktifkan garansi aktif sebelumnya dengan tipe ('toko'/'pabrik') yang sama.
	AssignWarranty(ctx context.Context, pw *ProductWarranty) error

	// FindByID mencari penugasan garansi produk berdasarkan ID.
	FindByID(ctx context.Context, id string) (*ProductWarranty, error)

	// FindActiveByProduct mengambil seluruh garansi aktif milik suatu produk (maksimal 1 toko dan 1 pabrik).
	FindActiveByProduct(ctx context.Context, productID string) ([]*ProductWarranty, error)

	// FindActiveByProductAndType mencari garansi aktif spesifik berdasarkan produk dan tipe ('toko' / 'pabrik').
	FindActiveByProductAndType(ctx context.Context, productID string, warrantyType WarrantyType) (*ProductWarranty, error)

	// Update memperbarui penugasan garansi produk (misal: penonaktifan).
	Update(ctx context.Context, pw *ProductWarranty) error
}

// ProductImageRepository adalah kontrak persistence untuk foto produk.
type ProductImageRepository interface {
	Save(ctx context.Context, img *ProductImage) error
	FindByProductID(ctx context.Context, productID string) ([]*ProductImage, error)
	FindByID(ctx context.Context, id string) (*ProductImage, error)
	FindPrimaryByProductID(ctx context.Context, productID string) (*ProductImage, error)
	Delete(ctx context.Context, id string) error
	SetPrimary(ctx context.Context, productID, imageID string) error
	CountByProductID(ctx context.Context, productID string) (int, error)
}

