package interfaces

import "time"

// --- Request DTOs (data dari client ke server) ---

// CreateProductRequest adalah struktur JSON yang dikirim client untuk membuat produk baru.
//
// DTO (Data Transfer Object) didesain untuk kebutuhan HTTP/JSON — berbeda dari domain entity.
// Domain entity punya method bisnis, validasi, dan state machine.
// DTO hanya membawa data mentah dari/ke client.
//
// Tag `json:"..."` mendefinisikan nama field di JSON request/response.
// Gunakan snake_case di JSON (konvensi REST API) dan PascalCase di Go struct.
type CreateProductRequest struct {
	SKU           string         `json:"sku"`
	CategoryID    string         `json:"category_id"`
	Name          string         `json:"name"`
	Brand         string         `json:"brand,omitempty"`
	Description   string         `json:"description,omitempty"`
	Unit          string         `json:"unit"`
	PurchasePrice int64          `json:"purchase_price"`
	SellingPrice  int64          `json:"selling_price"`
	IsPPN              bool           `json:"is_ppn"`
	FlagSerialTracking bool           `json:"flag_serial_tracking"`
	WeightGram         int            `json:"weight_gram,omitempty"`
	AtributVarian      map[string]any `json:"atribut_varian,omitempty"`
}

// UpdateProductRequest adalah struktur JSON untuk memperbarui data produk.
// SKU dan CategoryID tidak dimasukkan di sini karena tidak boleh diubah sembarangan
// demi menjaga integritas data dan riwayat audit.
type UpdateProductRequest struct {
	Name          string         `json:"name"`
	Brand         string         `json:"brand,omitempty"`
	Description   string         `json:"description,omitempty"`
	Unit          string         `json:"unit"`
	PurchasePrice int64          `json:"purchase_price"`
	SellingPrice  int64          `json:"selling_price"`
	IsPPN              bool           `json:"is_ppn"`
	FlagSerialTracking bool           `json:"flag_serial_tracking"`
	WeightGram         int            `json:"weight_gram,omitempty"`
	AtributVarian      map[string]any `json:"atribut_varian,omitempty"`
}

// SetProductStatusRequest adalah struktur JSON untuk mengubah status aktif produk.
type SetProductStatusRequest struct {
	Status string `json:"status"`
}

// ListProductRequest adalah query parameter untuk endpoint list produk.
type ListProductRequest struct {
	Status     string `json:"status,omitempty"`
	CategoryID string `json:"category_id,omitempty"`
	Search     string `json:"search,omitempty"`
	Page       int    `json:"page,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// --- Response DTOs (data dari server ke client) ---

// ProductResponse adalah data produk yang dikirim ke client.
// Perhatikan: field ini mungkin berbeda dari domain entity —
// respons bisa menyembunyikan field internal atau menambah field computed.
type ProductResponse struct {
	ID            string         `json:"id"`
	SKU           string         `json:"sku"`
	CategoryID    string         `json:"category_id"`
	Name          string         `json:"name"`
	Brand         string         `json:"brand"`
	Description   string         `json:"description"`
	Unit          string         `json:"unit"`
	PurchasePrice int64          `json:"purchase_price"`
	SellingPrice  int64          `json:"selling_price"`
	Status             string         `json:"status"`
	IsPPN              bool           `json:"is_ppn"`
	FlagSerialTracking bool           `json:"flag_serial_tracking"`
	WeightGram         int            `json:"weight_gram"`
	AtributVarian      map[string]any `json:"atribut_varian"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// ListProductResponse adalah respons untuk endpoint list produk.
type ListProductResponse struct {
	Data       []*ProductResponse `json:"data"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}

// CreateProductResponse adalah respons setelah produk berhasil dibuat.
type CreateProductResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// ErrorResponse adalah format error standar yang dikembalikan ke client.
type ErrorResponse struct {
	Error string `json:"error"`
}

// --- Category DTOs ---

type CreateCategoryRequest struct {
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id,omitempty"`
	ImageURL *string `json:"image_url,omitempty"`
}

type UpdateCategoryRequest struct {
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id,omitempty"`
	ImageURL *string `json:"image_url,omitempty"`
}

type CategoryResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ParentID  *string   `json:"parent_id,omitempty"`
	ImageURL  *string   `json:"image_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Location DTOs ---

type CreateLocationRequest struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Type    string `json:"type"` // "physical" atau "online"
	Address string `json:"address,omitempty"`
}

type UpdateLocationRequest struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Type    string `json:"type"` // "physical" atau "online"
	Address string `json:"address,omitempty"`
}

type SetLocationStatusRequest struct {
	IsActive bool `json:"is_active"`
}

type LocationResponse struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Address   string    `json:"address"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Stock DTOs ---

// AdjustStockRequest adalah struktur JSON untuk penyesuaian stok manual (Stock Opname).
type AdjustStockRequest struct {
	ProductID   string `json:"product_id"`
	LocationID  string `json:"location_id"`
	NewQuantity int    `json:"new_quantity"`
	Reason      string `json:"reason,omitempty"`
}

// UpdateMinStockRequest adalah struktur JSON untuk memperbarui batas minimum stok (alert threshold).
type UpdateMinStockRequest struct {
	ProductID  string `json:"product_id"`
	LocationID string `json:"location_id"`
	MinStock   int    `json:"min_stock"`
}

// StockResponse adalah data representasi stok per cabang yang dikembalikan ke client.
type StockResponse struct {
	ID                string    `json:"id"`
	ProductID         string    `json:"product_id"`
	LocationID        string    `json:"location_id"`
	Quantity          int       `json:"quantity"`
	ReservedQuantity  int       `json:"reserved_quantity"`
	AvailableQuantity int       `json:"available_quantity"`
	MinStock          int       `json:"min_stock"`
	IsLowStock        bool      `json:"is_low_stock"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// --- Barcode DTOs ---

// AddBarcodeRequest adalah struktur JSON untuk mendaftarkan barcode baru ke suatu produk.
type AddBarcodeRequest struct {
	Barcode   string `json:"barcode"`
	IsPrimary bool   `json:"is_primary"`
}

// BarcodeResponse adalah representasi JSON data barcode produk.
type BarcodeResponse struct {
	ID        string    `json:"id"`
	ProductID string    `json:"product_id"`
	Barcode   string    `json:"barcode"`
	IsPrimary bool      `json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
}

// ProductLookupResponse adalah respon hasil scan barcode kasir (mengembalikan detail produk dan barcode).
type ProductLookupResponse struct {
	Product        *ProductResponse `json:"product"`
	ScannedBarcode *BarcodeResponse `json:"scanned_barcode"`
}

// --- Serial Unit DTOs ---

// RegisterSerialUnitsRequest adalah struktur JSON untuk mendaftarkan satu atau banyak nomor seri ke suatu produk.
type RegisterSerialUnitsRequest struct {
	LocationID    string   `json:"location_id"`
	SerialNumbers []string `json:"serial_numbers"`
}

// SerialUnitResponse adalah representasi JSON data unit fisik berserial.
type SerialUnitResponse struct {
	ID           string    `json:"id"`
	ProductID    string    `json:"product_id"`
	LocationID   string    `json:"location_id"`
	SerialNumber string    `json:"serial_number"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SerialUnitLookupResponse adalah respon hasil scan nomor seri/IMEI untuk kasir dan layanan garansi.
type SerialUnitLookupResponse struct {
	SerialUnit   *SerialUnitResponse `json:"serial_unit"`
	ProductName  string              `json:"product_name"`
	ProductSKU   string              `json:"product_sku"`
	ProductBrand string              `json:"product_brand"`
	LocationName string              `json:"location_name"`
	LocationCode string              `json:"location_code"`
}

// UpdateSerialStatusRequest adalah struktur JSON untuk memperbarui status unit fisik (misal: terjual atau retur).
type UpdateSerialStatusRequest struct {
	Status string `json:"status"`
}

// --- Price Override DTOs ---

// CreatePriceOverrideRequest adalah struktur JSON untuk mendaftarkan promo harga cabang baru.
type CreatePriceOverrideRequest struct {
	LocationID       string    `json:"location_id"`
	PromotionalPrice int64     `json:"promotional_price"`
	MaxQuantity      *int      `json:"max_quantity,omitempty"`
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	Reason           string    `json:"reason,omitempty"`
}

// PriceOverrideResponse adalah representasi JSON promo harga cabang.
type PriceOverrideResponse struct {
	ID                string    `json:"id"`
	ProductID         string    `json:"product_id"`
	LocationID        string    `json:"location_id"`
	PromotionalPrice  int64     `json:"promotional_price"`
	MaxQuantity       *int      `json:"max_quantity"`
	ClaimedQuantity   int       `json:"claimed_quantity"`
	RemainingQuantity *int      `json:"remaining_quantity"`
	StartDate         time.Time `json:"start_date"`
	EndDate           time.Time `json:"end_date"`
	Reason            string    `json:"reason"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// EffectivePriceResponse adalah representasi harga jual final untuk kasir POS.
type EffectivePriceResponse struct {
	ProductID      string  `json:"product_id"`
	LocationID     string  `json:"location_id"`
	BasePrice      int64   `json:"base_price"`
	EffectivePrice int64   `json:"effective_price"`
	HasDiscount    bool    `json:"has_discount"`
	DiscountAmount int64   `json:"discount_amount"`
	RemainingQuota *int    `json:"remaining_quota,omitempty"`
	PromoID        *string `json:"promo_id,omitempty"`
	PromoReason    *string `json:"promo_reason,omitempty"`
}

// ClaimPromoQuotaRequest adalah request untuk mengklaim sejumlah unit kuota promo (misal saat checkout kasir).
type ClaimPromoQuotaRequest struct {
	Quantity int `json:"quantity"`
}

// --- Stock Transfer DTOs ---

// StockTransferItemRequest adalah baris item barang dalam pengajuan mutasi stok.
type StockTransferItemRequest struct {
	ProductID     string   `json:"product_id"`
	Quantity      int      `json:"quantity"`
	SerialUnitIDs []string `json:"serial_unit_ids,omitempty"`
}

// CreateStockTransferRequest adalah request payload untuk membuat dokumen mutasi stok baru.
type CreateStockTransferRequest struct {
	FromLocationID string                     `json:"from_location_id"`
	ToLocationID   string                     `json:"to_location_id"`
	Notes          string                     `json:"notes,omitempty"`
	Items          []StockTransferItemRequest `json:"items"`
}

// RejectStockTransferRequest adalah request payload untuk menolak permohonan transfer.
type RejectStockTransferRequest struct {
	Reason string `json:"reason"`
}

// StockTransferItemResponse adalah representasi JSON baris item barang dalam dokumen transfer.
type StockTransferItemResponse struct {
	ID               string   `json:"id"`
	ProductID        string   `json:"product_id"`
	Quantity         int      `json:"quantity"`
	ReceivedQuantity int      `json:"received_quantity"`
	SerialUnitIDs    []string `json:"serial_unit_ids"`
	CreatedAt        time.Time `json:"created_at"`
}

// StockTransferResponse adalah representasi JSON dokumen mutasi stok lengkap.
type StockTransferResponse struct {
	ID              string                      `json:"id"`
	TransferNumber  string                      `json:"transfer_number"`
	FromLocationID  string                      `json:"from_location_id"`
	ToLocationID    string                      `json:"to_location_id"`
	Status          string                      `json:"status"`
	Notes           string                      `json:"notes"`
	RejectionReason string                      `json:"rejection_reason,omitempty"`
	RequestedBy     string                      `json:"requested_by"`
	ApprovedBy      *string                     `json:"approved_by,omitempty"`
	ReceivedBy      *string                     `json:"received_by,omitempty"`
	Items           []StockTransferItemResponse `json:"items"`
	CreatedAt       time.Time                   `json:"created_at"`
	UpdatedAt       time.Time                   `json:"updated_at"`
}

// --- Warranty DTOs ---

// CreateWarrantyPolicyRequest adalah request body untuk membuat master kebijakan garansi baru.
type CreateWarrantyPolicyRequest struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	DurationMonths    int    `json:"duration_months"`
	DurationDays      int    `json:"duration_days"`
	Coverage          string `json:"coverage,omitempty"`
	ClaimInstructions string `json:"claim_instructions,omitempty"`
}

// WarrantyPolicyResponse adalah representasi JSON master kebijakan garansi.
type WarrantyPolicyResponse struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Type              string    `json:"type"`
	DurationMonths    int       `json:"duration_months"`
	DurationDays      int       `json:"duration_days"`
	Coverage          string    `json:"coverage"`
	ClaimInstructions string    `json:"claim_instructions"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// AssignProductWarrantyRequest adalah request body untuk menetapkan garansi ke suatu produk.
type AssignProductWarrantyRequest struct {
	WarrantyPolicyID string `json:"warranty_policy_id"`
}

// ProductWarrantyResponse adalah representasi JSON penetapan garansi pada produk.
type ProductWarrantyResponse struct {
	ID               string                  `json:"id"`
	ProductID        string                  `json:"product_id"`
	WarrantyPolicyID string                  `json:"warranty_policy_id"`
	Type             string                  `json:"type"`
	IsActive         bool                    `json:"is_active"`
	Policy           *WarrantyPolicyResponse `json:"policy,omitempty"`
	CreatedAt        time.Time               `json:"created_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
}
