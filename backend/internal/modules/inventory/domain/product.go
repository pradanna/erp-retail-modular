package domain

import (
	"errors"
	"time"
)

// ProductStatus adalah value object yang merepresentasikan status produk.
// Menggunakan custom type string (bukan plain string) memberikan type-safety:
// compiler Go akan error jika kamu tidak sengaja memasukkan string sembarangan.
type ProductStatus string

const (
	ProductStatusActive       ProductStatus = "active"
	ProductStatusInactive     ProductStatus = "inactive"
	ProductStatusDiscontinued ProductStatus = "discontinued"
)

var (
	ErrProductNotFound = errors.New("produk tidak ditemukan")
)

// Product adalah aggregate root dari modul Inventory.
//
// "Aggregate Root" dalam DDD berarti: semua perubahan state untuk sekumpulan entity
// (Product, ProductBarcode, dll.) harus melewati objek ini.
// Tidak ada yang boleh mengubah ProductBarcode langsung tanpa melewati Product.
//
// Field menggunakan tipe Go asli — tidak ada tag `json`, `db`, atau sejenisnya.
// Layer domain TIDAK TAHU tentang JSON, SQL, atau HTTP. Itu urusan layer lain.
type Product struct {
	ID          string
	SKU         string
	CategoryID  string
	Name        string
	Brand       string
	Description string
	Unit        string

	// Harga disimpan dalam RUPIAH BULAT (integer), bukan sen.
	// Contoh: Rp 150.000 disimpan tepat sebagai 150000.
	// Tipe int64 digunakan untuk mencegah floating-point error dan menampung nominal triliunan rupiah.
	PurchasePrice int64
	SellingPrice  int64

	Status            ProductStatus
	IsPPN             bool    // Apakah produk kena PPN? Default: true
	FlagSerialTracking bool   // Apakah setiap unit dilacak via serial number/IMEI?
	WeightGram        int     // Berat dalam gram (integer), untuk estimasi ongkir storefront tanpa floating-point error
	AtributVarian     map[string]any // JSON fleksibel: {"warna": "merah", "kapasitas": "512GB"}

	Barcodes  []ProductBarcode
	Images    []ProductImage
	PrimaryImageURL *string // URL foto utama produk untuk efisiensi thumbnail katalog

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProductImage adalah entitas anak untuk foto produk.
type ProductImage struct {
	ID        string
	ProductID string
	URL       string
	IsPrimary bool
	SortOrder int
	CreatedAt time.Time
}

// NewProductImage membuat instance valid dari foto produk.
func NewProductImage(id, productID, url string, isPrimary bool, sortOrder int) (*ProductImage, error) {
	if id == "" {
		return nil, errors.New("id gambar tidak boleh kosong")
	}
	if productID == "" {
		return nil, errors.New("product_id tidak boleh kosong")
	}
	if url == "" {
		return nil, errors.New("url gambar tidak boleh kosong")
	}
	return &ProductImage{
		ID:        id,
		ProductID: productID,
		URL:       url,
		IsPrimary: isPrimary,
		SortOrder: sortOrder,
		CreatedAt: time.Now(),
	}, nil
}

// NewProduct adalah constructor domain — satu-satunya cara yang "sah" membuat Product baru.
// Constructor memvalidasi invariant dasar sebelum entity dibuat.
// Menggunakan constructor (bukan struct literal langsung) memastikan tidak ada Product
// yang invalid bisa masuk ke sistem.
func NewProduct(
	id, sku, categoryID, name, brand, unit string,
	purchasePrice, sellingPrice int64,
) (*Product, error) {
	if err := validateProduct(sku, name, unit, purchasePrice, sellingPrice); err != nil {
		return nil, err
	}

	now := time.Now()
	return &Product{
		ID:            id,
		SKU:           sku,
		CategoryID:    categoryID,
		Name:          name,
		Brand:         brand,
		Unit:          unit,
		PurchasePrice: purchasePrice,
		SellingPrice:  sellingPrice,
		Status:        ProductStatusActive, // default: aktif saat dibuat
		IsPPN:         true,               // default: kena PPN
		AtributVarian: map[string]any{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// Deactivate mengubah status produk menjadi inactive.
// Business logic (rules) tetap di domain, bukan di handler atau use case.
func (p *Product) Deactivate() {
	p.Status = ProductStatusInactive
	p.UpdatedAt = time.Now()
}

// Discontinue menandai produk tidak dijual lagi.
func (p *Product) Discontinue() {
	p.Status = ProductStatusDiscontinued
	p.UpdatedAt = time.Now()
}

// Activate mengaktifkan kembali produk yang nonaktif.
func (p *Product) Activate() {
	p.Status = ProductStatusActive
	p.UpdatedAt = time.Now()
}

// ChangeStatus mengubah status produk dengan memvalidasi status yang diperbolehkan.
func (p *Product) ChangeStatus(status ProductStatus) error {
	switch status {
	case ProductStatusActive:
		p.Activate()
	case ProductStatusInactive:
		p.Deactivate()
	case ProductStatusDiscontinued:
		p.Discontinue()
	default:
		return errors.New("status produk tidak valid")
	}
	return nil
}

// SetAtributVarian memperbarui atribut varian produk (generic JSON).
func (p *Product) SetAtributVarian(varian map[string]any) {
	if varian == nil {
		varian = map[string]any{}
	}
	p.AtributVarian = varian
	p.UpdatedAt = time.Now()
}

// SetPPN mengatur apakah produk dikenakan PPN atau tidak.
func (p *Product) SetPPN(isPPN bool) {
	p.IsPPN = isPPN
	p.UpdatedAt = time.Now()
}

// UpdatePrice memperbarui harga beli dan harga jual dengan validasi aturan bisnis.
func (p *Product) UpdatePrice(purchasePrice, sellingPrice int64) error {
	if purchasePrice < 0 {
		return errors.New("harga beli tidak boleh negatif")
	}
	if sellingPrice < 0 {
		return errors.New("harga jual tidak boleh negatif")
	}
	p.PurchasePrice = purchasePrice
	p.SellingPrice = sellingPrice
	p.UpdatedAt = time.Now()
	return nil
}

// AddBarcode menambahkan barcode baru ke produk dengan memastikan tidak ada barcode duplikat.
func (p *Product) AddBarcode(id, barcode string) error {
	if barcode == "" {
		return errors.New("barcode tidak boleh kosong")
	}
	for _, b := range p.Barcodes {
		if b.Barcode == barcode {
			return errors.New("barcode ini sudah terdaftar pada produk")
		}
	}
	p.Barcodes = append(p.Barcodes, ProductBarcode{
		ID:        id,
		ProductID: p.ID,
		Barcode:   barcode,
		CreatedAt: time.Now(),
	})
	p.UpdatedAt = time.Now()
	return nil
}

// RemoveBarcode menghapus barcode tertentu dari produk.
func (p *Product) RemoveBarcode(barcode string) error {
	found := false
	newBarcodes := make([]ProductBarcode, 0, len(p.Barcodes))
	for _, b := range p.Barcodes {
		if b.Barcode == barcode {
			found = true
			continue // Lewati barcode yang mau dihapus
		}
		newBarcodes = append(newBarcodes, b)
	}

	if !found {
		return errors.New("barcode tidak ditemukan pada produk ini")
	}

	p.Barcodes = newBarcodes
	p.UpdatedAt = time.Now()
	return nil
}

// UpdateDetails memperbarui informasi dasar produk.
func (p *Product) UpdateDetails(name, brand, unit, description string, weightGram int) error {
	if name == "" {
		return errors.New("nama produk tidak boleh kosong")
	}
	if unit == "" {
		return errors.New("satuan produk tidak boleh kosong")
	}
	if weightGram < 0 {
		return errors.New("berat produk tidak boleh negatif")
	}

	p.Name = name
	p.Brand = brand
	p.Unit = unit
	p.Description = description
	p.WeightGram = weightGram
	p.UpdatedAt = time.Now()
	return nil
}

// validateProduct memvalidasi field-field wajib yang harus selalu valid saat inisialisasi.
func validateProduct(sku, name, unit string, purchasePrice, sellingPrice int64) error {
	if sku == "" {
		return errors.New("SKU produk tidak boleh kosong")
	}
	if name == "" {
		return errors.New("nama produk tidak boleh kosong")
	}
	if unit == "" {
		return errors.New("satuan produk tidak boleh kosong")
	}
	if purchasePrice < 0 {
		return errors.New("harga beli tidak boleh negatif")
	}
	if sellingPrice < 0 {
		return errors.New("harga jual tidak boleh negatif")
	}
	return nil
}

