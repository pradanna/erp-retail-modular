package inventory

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory/application"
	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/internal/modules/inventory/infrastructure"
	inventoryHTTP "github.com/erp-retail/backend/internal/modules/inventory/interfaces"
	"github.com/erp-retail/backend/internal/shared/event"
)

// InventoryService adalah interface publik (facade) yang diekspos modul Inventory
// ke modul lain yang membutuhkan data inventory secara sinkron.
//
// PENTING: Modul Sales, Purchasing, dll. hanya boleh berinteraksi dengan Inventory
// melalui interface ini — bukan langsung import package domain/infrastructure Inventory.
// Ini menjaga Bounded Context tetap terisolasi.
type InventoryService interface {
	// GetProductByID mengambil informasi produk berdasarkan ID.
	// Dipakai oleh modul Sales untuk validasi produk saat membuat transaksi.
	GetProductByID(id string) (*ProductInfo, error)

	// GetLocationByID mengambil informasi lokasi cabang/gudang berdasarkan ID.
	// Dipakai oleh modul Sales, Purchasing, dan Ecommerce untuk validasi lokasi.
	GetLocationByID(id string) (*LocationInfo, error)

	// GetStock mengambil informasi ketersediaan stok produk pada cabang/gudang tertentu.
	// Dipakai oleh modul Sales untuk validasi stok sebelum checkout.
	GetStock(productID, locationID string) (*StockInfo, error)

	// GetProductByBarcode mencari data produk berdasarkan kode barcode pabrik yang di-scan.
	// Dipakai oleh modul Sales saat kasir menembak barcode dengan scanner.
	GetProductByBarcode(barcode string) (*ProductInfo, error)

	// GetSerialUnitBySN mencari data unit fisik berdasarkan serial number / IMEI.
	// Dipakai oleh modul Sales (kasir POS) dan Aftersales (klaim garansi).
	GetSerialUnitBySN(sn string) (*SerialUnitInfo, error)

	// GetEffectivePrice menghitung harga jual riil saat ini (memperhitungkan promo aktif cabang).
	// Dipakai oleh modul Sales saat kasir membuat nota transaksi penjualan.
	GetEffectivePrice(productID, locationID string) (*EffectivePriceInfo, error)

	// ClaimPromoQuota mengklaim / memotong sejumlah kuota promo saat terjadi checkout penjualan.
	// Dipakai oleh modul Sales saat kasir memproses pembayaran.
	ClaimPromoQuota(promoID string, qty int) error

	// GetProductWarranties mengambil garansi toko dan garansi resmi pabrik yang aktif pada produk.
	// Dipakai oleh modul Sales saat mencetak nota / kartu garansi ke pembeli.
	GetProductWarranties(productID string) ([]*ProductWarrantyInfo, error)
}

// ProductInfo adalah DTO minimal yang diekspos ke modul lain.
// Sengaja lebih sedikit field dari domain entity penuh —
// modul lain tidak perlu tahu semua detail internal Inventory.
type ProductInfo struct {
	ID           string
	Name         string
	SellingPrice int64
	IsActive     bool
}

// LocationInfo adalah DTO minimal lokasi yang diekspos ke modul lain.
type LocationInfo struct {
	ID       string
	Code     string
	Name     string
	Type     string
	IsActive bool
}

// StockInfo adalah DTO minimal ketersediaan stok yang diekspos ke modul lain.
type StockInfo struct {
	ProductID         string
	LocationID        string
	Quantity          int
	ReservedQuantity  int
	AvailableQuantity int
}

// SerialUnitInfo adalah DTO minimal unit fisik berserial yang diekspos ke modul lain.
type SerialUnitInfo struct {
	ID           string
	ProductID    string
	LocationID   string
	SerialNumber string
	Status       string
}

// EffectivePriceInfo adalah DTO minimal kalkulasi harga riil yang diekspos ke modul Sales.
type EffectivePriceInfo struct {
	ProductID      string
	LocationID     string
	BasePrice      int64
	EffectivePrice int64
	HasDiscount    bool
	DiscountAmount int64
	RemainingQuota *int
	PromoReason    *string
}

// ProductWarrantyInfo adalah DTO minimal informasi garansi produk yang diekspos ke modul lain.
type ProductWarrantyInfo struct {
	ID                string
	ProductID         string
	WarrantyPolicyID  string
	Type              string
	PolicyName        string
	DurationMonths    int
	DurationDays      int
	Coverage          string
	ClaimInstructions string
	IsActive          bool
}

// inventoryServiceImpl mengimplementasikan InventoryService untuk dipanggil modul lain.
type inventoryServiceImpl struct {
	repo         domain.ProductRepository
	locationRepo domain.LocationRepository
	stockRepo    domain.StockRepository
	barcodeRepo  domain.BarcodeRepository
	serialRepo   domain.SerialUnitRepository
	priceRepo    domain.PriceOverrideRepository
	warrantyRepo domain.ProductWarrantyRepository
}

func (s *inventoryServiceImpl) GetProductByID(id string) (*ProductInfo, error) {
	ctx := context.Background()
	p, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	return &ProductInfo{
		ID:           p.ID,
		Name:         p.Name,
		SellingPrice: p.SellingPrice,
		IsActive:     p.Status == domain.ProductStatusActive,
	}, nil
}

func (s *inventoryServiceImpl) GetLocationByID(id string) (*LocationInfo, error) {
	ctx := context.Background()
	l, err := s.locationRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if l == nil {
		return nil, nil
	}
	return &LocationInfo{
		ID:       l.ID,
		Code:     l.Code,
		Name:     l.Name,
		Type:     string(l.Type),
		IsActive: l.IsActive,
	}, nil
}

func (s *inventoryServiceImpl) GetStock(productID, locationID string) (*StockInfo, error) {
	ctx := context.Background()
	item, err := s.stockRepo.FindByProductAndLocation(ctx, productID, locationID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return &StockInfo{
			ProductID:         productID,
			LocationID:        locationID,
			Quantity:          0,
			ReservedQuantity:  0,
			AvailableQuantity: 0,
		}, nil
	}
	return &StockInfo{
		ProductID:         item.ProductID,
		LocationID:        item.LocationID,
		Quantity:          item.Quantity,
		ReservedQuantity:  item.ReservedQuantity,
		AvailableQuantity: item.AvailableQuantity(),
	}, nil
}

func (s *inventoryServiceImpl) GetProductByBarcode(barcode string) (*ProductInfo, error) {
	ctx := context.Background()
	p, _, err := s.barcodeRepo.FindProductByBarcode(ctx, barcode)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, nil
	}
	return &ProductInfo{
		ID:           p.ID,
		Name:         p.Name,
		SellingPrice: p.SellingPrice,
		IsActive:     p.Status == domain.ProductStatusActive,
	}, nil
}

func (s *inventoryServiceImpl) GetSerialUnitBySN(sn string) (*SerialUnitInfo, error) {
	ctx := context.Background()
	u, err := s.serialRepo.FindBySerialNumber(ctx, sn)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, nil
	}
	return &SerialUnitInfo{
		ID:           u.ID,
		ProductID:    u.ProductID,
		LocationID:   u.LocationID,
		SerialNumber: u.SerialNumber,
		Status:       string(u.Status),
	}, nil
}

func (s *inventoryServiceImpl) GetEffectivePrice(productID, locationID string) (*EffectivePriceInfo, error) {
	ctx := context.Background()
	now := time.Now().UTC()

	product, err := s.repo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, nil
	}

	res := &EffectivePriceInfo{
		ProductID:      productID,
		LocationID:     locationID,
		BasePrice:      product.SellingPrice,
		EffectivePrice: product.SellingPrice,
		HasDiscount:    false,
		DiscountAmount: 0,
	}

	promo, err := s.priceRepo.FindActive(ctx, productID, locationID, now)
	if err != nil && !errors.Is(err, domain.ErrPromoNotFound) {
		return nil, err
	}

	if promo != nil {
		res.EffectivePrice = promo.PromotionalPrice
		res.HasDiscount = true
		res.DiscountAmount = product.SellingPrice - promo.PromotionalPrice
		if res.DiscountAmount < 0 {
			res.DiscountAmount = 0
		}
		res.RemainingQuota = promo.RemainingQuota()
		res.PromoReason = &promo.Reason
	}

	return res, nil
}

func (s *inventoryServiceImpl) ClaimPromoQuota(promoID string, qty int) error {
	ctx := context.Background()
	return s.priceRepo.IncrementClaimedQuantity(ctx, promoID, qty)
}

func (s *inventoryServiceImpl) GetProductWarranties(productID string) ([]*ProductWarrantyInfo, error) {
	ctx := context.Background()
	warranties, err := s.warrantyRepo.FindActiveByProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	var res []*ProductWarrantyInfo
	for _, w := range warranties {
		info := &ProductWarrantyInfo{
			ID:               w.ID,
			ProductID:        w.ProductID,
			WarrantyPolicyID: w.WarrantyPolicyID,
			Type:             string(w.Type),
			IsActive:         w.IsActive,
		}
		if w.Policy != nil {
			info.PolicyName = w.Policy.Name
			info.DurationMonths = w.Policy.DurationMonths
			info.DurationDays = w.Policy.DurationDays
			info.Coverage = w.Policy.Coverage
			info.ClaimInstructions = w.Policy.ClaimInstructions
		}
		res = append(res, info)
	}
	return res, nil
}

// Module adalah struktur yang menyimpan semua komponen modul Inventory
// yang sudah di-wire (disiapkan koneksinya).
type Module struct {
	service              InventoryService
	productHandler       *inventoryHTTP.ProductHandler
	categoryHandler      *inventoryHTTP.CategoryHandler
	locationHandler      *inventoryHTTP.LocationHandler
	stockHandler         *inventoryHTTP.StockHandler
	barcodeHandler       *inventoryHTTP.BarcodeHandler
	serialUnitHandler    *inventoryHTTP.SerialUnitHandler
	priceOverrideHandler *inventoryHTTP.PriceOverrideHandler
	stockTransferHandler *inventoryHTTP.StockTransferHandler
	warrantyHandler      *inventoryHTTP.WarrantyHandler
}

// Service mengembalikan public interface InventoryService untuk digunakan modul lain.
func (m *Module) Service() InventoryService {
	return m.service
}

// New membuat dan meng-wire (menghubungkan) semua komponen modul Inventory.
// Di sinilah Dependency Injection (DI) manual dilakukan:
//   infrastructure (repo) → application (use case) → interfaces (handler)
func New(db *sql.DB, bus event.Bus, logger *slog.Logger) *Module {
	// Layer 1: Infrastructure — implementasi repository
	productRepo := infrastructure.NewProductRepository(db)
	categoryRepo := infrastructure.NewCategoryRepository(db)
	locationRepo := infrastructure.NewLocationRepository(db)
	stockRepo := infrastructure.NewStockRepository(db)
	barcodeRepo := infrastructure.NewBarcodeRepository(db)
	serialRepo := infrastructure.NewSerialUnitRepository(db)
	priceOverrideRepo := infrastructure.NewPriceOverrideRepository(db)
	transferRepo := infrastructure.NewStockTransferRepository(db)
	warrantyPolicyRepo := infrastructure.NewWarrantyPolicyRepository(db)
	productWarrantyRepo := infrastructure.NewProductWarrantyRepository(db)

	// Layer 2: Application — use cases Produk
	createProductUC := application.NewCreateProductUseCase(productRepo)
	getProductUC := application.NewGetProductUseCase(productRepo)
	listProductsUC := application.NewListProductsUseCase(productRepo)
	updateProductUC := application.NewUpdateProductUseCase(productRepo)
	setStatusUC := application.NewSetProductStatusUseCase(productRepo)

	// Layer 2: Application — use cases Kategori
	createCategoryUC := application.NewCreateCategoryUseCase(categoryRepo)
	listCategoriesUC := application.NewListCategoriesUseCase(categoryRepo)
	updateCategoryUC := application.NewUpdateCategoryUseCase(categoryRepo)
	deleteCategoryUC := application.NewDeleteCategoryUseCase(categoryRepo)

	// Layer 2: Application — use cases Lokasi
	createLocationUC := application.NewCreateLocationUseCase(locationRepo)
	listLocationsUC := application.NewListLocationsUseCase(locationRepo)
	getLocationUC := application.NewGetLocationUseCase(locationRepo)
	updateLocationUC := application.NewUpdateLocationUseCase(locationRepo)
	setLocationStatusUC := application.NewSetLocationStatusUseCase(locationRepo)
	deleteLocationUC := application.NewDeleteLocationUseCase(locationRepo)

	// Layer 2: Application — use cases Stok
	adjustStockUC := application.NewAdjustStockUseCase(stockRepo, productRepo, locationRepo)
	getStockUC := application.NewGetStockUseCase(stockRepo)
	listStockUC := application.NewListStockByLocationUseCase(stockRepo, locationRepo)
	listAlertsUC := application.NewListLowStockAlertsUseCase(stockRepo)
	updateMinStockUC := application.NewUpdateMinStockUseCase(stockRepo, productRepo, locationRepo)

	// Layer 2: Application — use cases Barcode
	addBarcodeUC := application.NewAddBarcodeUseCase(barcodeRepo, productRepo)
	deleteBarcodeUC := application.NewDeleteBarcodeUseCase(barcodeRepo)
	listBarcodesUC := application.NewListBarcodesByProductUseCase(barcodeRepo, productRepo)
	lookupBarcodeUC := application.NewLookupProductByBarcodeUseCase(barcodeRepo)

	// Layer 2: Application — use cases Serial Unit
	registerSerialUC := application.NewRegisterSerialUnitsUseCase(serialRepo, productRepo, locationRepo)
	lookupSerialUC := application.NewLookupSerialNumberUseCase(serialRepo, productRepo, locationRepo)
	listSerialUC := application.NewListSerialUnitsUseCase(serialRepo)
	updateSerialStatusUC := application.NewUpdateSerialStatusUseCase(serialRepo)

	// Layer 2: Application — use cases Price Override
	createOverrideUC := application.NewCreatePriceOverrideUseCase(priceOverrideRepo, productRepo, locationRepo)
	getEffectivePriceUC := application.NewGetEffectivePriceUseCase(priceOverrideRepo, productRepo, locationRepo)
	listOverridesUC := application.NewListPriceOverridesUseCase(priceOverrideRepo)
	deactivateOverrideUC := application.NewDeactivatePriceOverrideUseCase(priceOverrideRepo)
	claimQuotaUC := application.NewClaimPromoQuotaUseCase(priceOverrideRepo)

	// Layer 2: Application — use cases Stock Transfer (Mutasi Stok)
	createTransferUC := application.NewCreateStockTransferUseCase(transferRepo, stockRepo, locationRepo, productRepo, serialRepo)
	approveTransferUC := application.NewApproveStockTransferUseCase(transferRepo)
	rejectTransferUC := application.NewRejectStockTransferUseCase(transferRepo, stockRepo)
	shipTransferUC := application.NewShipStockTransferUseCase(transferRepo, stockRepo)
	receiveTransferUC := application.NewReceiveStockTransferUseCase(transferRepo, stockRepo, serialRepo)
	getTransferUC := application.NewGetStockTransferUseCase(transferRepo)
	listTransferUC := application.NewListStockTransfersUseCase(transferRepo)

	// Layer 2: Application — use cases Garansi (Warranty)
	createWarrantyPolicyUC := application.NewCreateWarrantyPolicyUseCase(warrantyPolicyRepo)
	listWarrantyPoliciesUC := application.NewListWarrantyPoliciesUseCase(warrantyPolicyRepo)
	assignProductWarrantyUC := application.NewAssignProductWarrantyUseCase(productRepo, warrantyPolicyRepo, productWarrantyRepo)
	getProductWarrantiesUC := application.NewGetProductActiveWarrantiesUseCase(productRepo, productWarrantyRepo)
	deactivateWarrantyUC := application.NewDeactivateProductWarrantyUseCase(productWarrantyRepo)

	// Layer 3: Interfaces — handlers
	productHandler := inventoryHTTP.NewProductHandler(
		createProductUC,
		getProductUC,
		listProductsUC,
		updateProductUC,
		setStatusUC,
		logger,
	)

	categoryHandler := inventoryHTTP.NewCategoryHandler(
		createCategoryUC,
		listCategoriesUC,
		updateCategoryUC,
		deleteCategoryUC,
		logger,
	)

	locationHandler := inventoryHTTP.NewLocationHandler(
		createLocationUC,
		listLocationsUC,
		getLocationUC,
		updateLocationUC,
		setLocationStatusUC,
		deleteLocationUC,
		logger,
	)

	stockHandler := inventoryHTTP.NewStockHandler(
		adjustStockUC,
		getStockUC,
		listStockUC,
		listAlertsUC,
		updateMinStockUC,
		logger,
	)

	barcodeHandler := inventoryHTTP.NewBarcodeHandler(
		addBarcodeUC,
		deleteBarcodeUC,
		listBarcodesUC,
		lookupBarcodeUC,
		logger,
	)

	serialUnitHandler := inventoryHTTP.NewSerialUnitHandler(
		registerSerialUC,
		lookupSerialUC,
		listSerialUC,
		updateSerialStatusUC,
		logger,
	)

	priceOverrideHandler := inventoryHTTP.NewPriceOverrideHandler(
		createOverrideUC,
		getEffectivePriceUC,
		listOverridesUC,
		deactivateOverrideUC,
		claimQuotaUC,
		logger,
	)

	stockTransferHandler := inventoryHTTP.NewStockTransferHandler(
		createTransferUC,
		approveTransferUC,
		rejectTransferUC,
		shipTransferUC,
		receiveTransferUC,
		getTransferUC,
		listTransferUC,
		logger,
	)

	warrantyHandler := inventoryHTTP.NewWarrantyHandler(
		createWarrantyPolicyUC,
		listWarrantyPoliciesUC,
		assignProductWarrantyUC,
		getProductWarrantiesUC,
		deactivateWarrantyUC,
		logger,
	)

	return &Module{
		service: &inventoryServiceImpl{
			repo:         productRepo,
			locationRepo: locationRepo,
			stockRepo:    stockRepo,
			barcodeRepo:  barcodeRepo,
			serialRepo:   serialRepo,
			priceRepo:    priceOverrideRepo,
			warrantyRepo: productWarrantyRepo,
		},
		productHandler:       productHandler,
		categoryHandler:      categoryHandler,
		locationHandler:      locationHandler,
		stockHandler:         stockHandler,
		barcodeHandler:       barcodeHandler,
		serialUnitHandler:    serialUnitHandler,
		priceOverrideHandler: priceOverrideHandler,
		stockTransferHandler: stockTransferHandler,
		warrantyHandler:      warrantyHandler,
	}
}

// Register mendaftarkan semua route HTTP modul Inventory ke router utama.
// Fungsi ini dipanggil HANYA oleh main.go, dan HANYA jika modul Inventory aktif di lisensi.
func (m *Module) Register(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	// --- Routes Produk ---

	// Route: POST /api/v1/inventory/products (Buat produk baru)
	mux.Handle("POST /api/v1/inventory/products",
		authMiddleware(http.HandlerFunc(m.productHandler.Create)),
	)

	// Route: GET /api/v1/inventory/products (Daftar produk dengan paging/filter/search)
	mux.Handle("GET /api/v1/inventory/products",
		authMiddleware(http.HandlerFunc(m.productHandler.List)),
	)

	// Route: GET /api/v1/inventory/products/{id} (Detail 1 produk)
	mux.Handle("GET /api/v1/inventory/products/{id}",
		authMiddleware(http.HandlerFunc(m.productHandler.GetByID)),
	)

	// Route: PUT /api/v1/inventory/products/{id} (Perbarui data produk)
	mux.Handle("PUT /api/v1/inventory/products/{id}",
		authMiddleware(http.HandlerFunc(m.productHandler.Update)),
	)

	// Route: PATCH /api/v1/inventory/products/{id}/status (Ubah status produk)
	mux.Handle("PATCH /api/v1/inventory/products/{id}/status",
		authMiddleware(http.HandlerFunc(m.productHandler.SetStatus)),
	)

	// --- Routes Kategori ---

	// Route: POST /api/v1/inventory/categories (Buat kategori baru)
	mux.Handle("POST /api/v1/inventory/categories",
		authMiddleware(http.HandlerFunc(m.categoryHandler.Create)),
	)

	// Route: GET /api/v1/inventory/categories (Daftar seluruh kategori)
	mux.Handle("GET /api/v1/inventory/categories",
		authMiddleware(http.HandlerFunc(m.categoryHandler.List)),
	)

	// Route: PUT /api/v1/inventory/categories/{id} (Perbarui kategori)
	mux.Handle("PUT /api/v1/inventory/categories/{id}",
		authMiddleware(http.HandlerFunc(m.categoryHandler.Update)),
	)

	// Route: DELETE /api/v1/inventory/categories/{id} (Hapus kategori)
	mux.Handle("DELETE /api/v1/inventory/categories/{id}",
		authMiddleware(http.HandlerFunc(m.categoryHandler.Delete)),
	)

	// --- Routes Lokasi / Cabang ---

	// Route: POST /api/v1/inventory/locations (Buat lokasi baru)
	mux.Handle("POST /api/v1/inventory/locations",
		authMiddleware(http.HandlerFunc(m.locationHandler.Create)),
	)

	// Route: GET /api/v1/inventory/locations (Daftar seluruh lokasi)
	mux.Handle("GET /api/v1/inventory/locations",
		authMiddleware(http.HandlerFunc(m.locationHandler.List)),
	)

	// Route: GET /api/v1/inventory/locations/{id} (Detail 1 lokasi)
	mux.Handle("GET /api/v1/inventory/locations/{id}",
		authMiddleware(http.HandlerFunc(m.locationHandler.GetByID)),
	)

	// Route: PUT /api/v1/inventory/locations/{id} (Perbarui informasi lokasi)
	mux.Handle("PUT /api/v1/inventory/locations/{id}",
		authMiddleware(http.HandlerFunc(m.locationHandler.Update)),
	)

	// Route: PATCH /api/v1/inventory/locations/{id}/status (Ubah status aktif lokasi)
	mux.Handle("PATCH /api/v1/inventory/locations/{id}/status",
		authMiddleware(http.HandlerFunc(m.locationHandler.SetStatus)),
	)

	// Route: DELETE /api/v1/inventory/locations/{id} (Hapus lokasi)
	mux.Handle("DELETE /api/v1/inventory/locations/{id}",
		authMiddleware(http.HandlerFunc(m.locationHandler.Delete)),
	)

	// --- Routes Stok per Cabang / Lokasi ---

	// Route: POST /api/v1/inventory/stocks/adjust (Penyesuaian stok manual / Stock Opname)
	mux.Handle("POST /api/v1/inventory/stocks/adjust",
		authMiddleware(http.HandlerFunc(m.stockHandler.Adjust)),
	)

	// Route: PUT /api/v1/inventory/stocks/min-stock (Perbarui batas minimum peringatan stok)
	mux.Handle("PUT /api/v1/inventory/stocks/min-stock",
		authMiddleware(http.HandlerFunc(m.stockHandler.UpdateMinStock)),
	)

	// Route: GET /api/v1/inventory/stocks (Ambil stok spesifik atau daftar stok cabang)
	mux.Handle("GET /api/v1/inventory/stocks",
		authMiddleware(http.HandlerFunc(m.stockHandler.Get)),
	)

	// Route: GET /api/v1/inventory/stocks/alerts (Daftar peringatan stok menipis / low stock)
	mux.Handle("GET /api/v1/inventory/stocks/alerts",
		authMiddleware(http.HandlerFunc(m.stockHandler.ListAlerts)),
	)

	// --- Routes Barcode Produk ---

	// Route: POST /api/v1/inventory/products/{id}/barcodes (Daftarkan barcode pabrik ke produk)
	mux.Handle("POST /api/v1/inventory/products/{id}/barcodes",
		authMiddleware(http.HandlerFunc(m.barcodeHandler.Add)),
	)

	// Route: GET /api/v1/inventory/products/{id}/barcodes (Daftar barcode milik produk)
	mux.Handle("GET /api/v1/inventory/products/{id}/barcodes",
		authMiddleware(http.HandlerFunc(m.barcodeHandler.ListByProduct)),
	)

	// Route: DELETE /api/v1/inventory/products/{id}/barcodes/{barcode_id} (Hapus barcode)
	mux.Handle("DELETE /api/v1/inventory/products/{id}/barcodes/{barcode_id}",
		authMiddleware(http.HandlerFunc(m.barcodeHandler.Delete)),
	)

	// Route: GET /api/v1/inventory/barcodes/lookup (Scan barcode kasir / cari produk)
	mux.Handle("GET /api/v1/inventory/barcodes/lookup",
		authMiddleware(http.HandlerFunc(m.barcodeHandler.Lookup)),
	)

	// --- Routes Serial Number & IMEI Tracking ---

	// Route: POST /api/v1/inventory/products/{id}/serials (Daftarkan serial number / IMEI unit fisik)
	mux.Handle("POST /api/v1/inventory/products/{id}/serials",
		authMiddleware(http.HandlerFunc(m.serialUnitHandler.Register)),
	)

	// Route: GET /api/v1/inventory/products/{id}/serials (Daftar serial unit milik suatu produk)
	mux.Handle("GET /api/v1/inventory/products/{id}/serials",
		authMiddleware(http.HandlerFunc(m.serialUnitHandler.ListByProduct)),
	)

	// Route: GET /api/v1/inventory/serials/lookup (Scan barcode serial number kasir & klaim garansi)
	mux.Handle("GET /api/v1/inventory/serials/lookup",
		authMiddleware(http.HandlerFunc(m.serialUnitHandler.Lookup)),
	)

	// Route: PATCH /api/v1/inventory/serials/{id}/status (Update status unit: terjual, retur)
	mux.Handle("PATCH /api/v1/inventory/serials/{id}/status",
		authMiddleware(http.HandlerFunc(m.serialUnitHandler.UpdateStatus)),
	)

	// --- Routes Price Override (Promo / Harga Khusus Cabang) ---

	// Route: POST /api/v1/inventory/products/{id}/price-overrides (Daftarkan promo harga cabang)
	mux.Handle("POST /api/v1/inventory/products/{id}/price-overrides",
		authMiddleware(http.HandlerFunc(m.priceOverrideHandler.Create)),
	)

	// Route: GET /api/v1/inventory/products/{id}/price-overrides (Daftar promo harga cabang produk)
	mux.Handle("GET /api/v1/inventory/products/{id}/price-overrides",
		authMiddleware(http.HandlerFunc(m.priceOverrideHandler.List)),
	)

	// Route: GET /api/v1/inventory/price-overrides/effective-price (Hitung harga efektif kasir POS)
	mux.Handle("GET /api/v1/inventory/price-overrides/effective-price",
		authMiddleware(http.HandlerFunc(m.priceOverrideHandler.GetEffectivePrice)),
	)

	// Route: PATCH /api/v1/inventory/price-overrides/{id}/deactivate (Nonaktifkan promo harga cabang)
	mux.Handle("PATCH /api/v1/inventory/price-overrides/{id}/deactivate",
		authMiddleware(http.HandlerFunc(m.priceOverrideHandler.Deactivate)),
	)

	// Route: POST /api/v1/inventory/price-overrides/{id}/claim (Klaim kuota promo saat transaksi kasir)
	mux.Handle("POST /api/v1/inventory/price-overrides/{id}/claim",
		authMiddleware(http.HandlerFunc(m.priceOverrideHandler.ClaimQuota)),
	)

	// --- Routes Stock Transfer (Mutasi Stok Antar Cabang) ---

	// Route: POST /api/v1/inventory/transfers (Buat permohonan mutasi stok baru)
	mux.Handle("POST /api/v1/inventory/transfers",
		authMiddleware(http.HandlerFunc(m.stockTransferHandler.Create)),
	)

	// Route: GET /api/v1/inventory/transfers (Daftar riwayat transfer stok)
	mux.Handle("GET /api/v1/inventory/transfers",
		authMiddleware(http.HandlerFunc(m.stockTransferHandler.List)),
	)

	// Route: GET /api/v1/inventory/transfers/{id} (Detail dokumen transfer dan item barangnya)
	mux.Handle("GET /api/v1/inventory/transfers/{id}",
		authMiddleware(http.HandlerFunc(m.stockTransferHandler.Get)),
	)

	// Route: POST /api/v1/inventory/transfers/{id}/approve (Setujui permohonan transfer)
	mux.Handle("POST /api/v1/inventory/transfers/{id}/approve",
		authMiddleware(http.HandlerFunc(m.stockTransferHandler.Approve)),
	)

	// Route: POST /api/v1/inventory/transfers/{id}/reject (Tolak permohonan transfer)
	mux.Handle("POST /api/v1/inventory/transfers/{id}/reject",
		authMiddleware(http.HandlerFunc(m.stockTransferHandler.Reject)),
	)

	// Route: POST /api/v1/inventory/transfers/{id}/ship (Konfirmasi pengiriman truk berangkat)
	mux.Handle("POST /api/v1/inventory/transfers/{id}/ship",
		authMiddleware(http.HandlerFunc(m.stockTransferHandler.Ship)),
	)

	// Route: POST /api/v1/inventory/transfers/{id}/receive (Konfirmasi penerimaan di cabang tujuan)
	mux.Handle("POST /api/v1/inventory/transfers/{id}/receive",
		authMiddleware(http.HandlerFunc(m.stockTransferHandler.Receive)),
	)

	// --- Routes Warranty (Garansi Toko & Pabrik) ---

	// Route: POST /api/v1/inventory/warranties/policies (Buat master kebijakan garansi)
	mux.Handle("POST /api/v1/inventory/warranties/policies",
		authMiddleware(http.HandlerFunc(m.warrantyHandler.CreatePolicy)),
	)

	// Route: GET /api/v1/inventory/warranties/policies (Daftar master kebijakan garansi)
	mux.Handle("GET /api/v1/inventory/warranties/policies",
		authMiddleware(http.HandlerFunc(m.warrantyHandler.ListPolicies)),
	)

	// Route: POST /api/v1/inventory/products/{id}/warranties (Tetapkan garansi ke produk)
	mux.Handle("POST /api/v1/inventory/products/{id}/warranties",
		authMiddleware(http.HandlerFunc(m.warrantyHandler.AssignProductWarranty)),
	)

	// Route: GET /api/v1/inventory/products/{id}/warranties (Lihat garansi aktif produk)
	mux.Handle("GET /api/v1/inventory/products/{id}/warranties",
		authMiddleware(http.HandlerFunc(m.warrantyHandler.GetProductWarranties)),
	)

	// Route: POST /api/v1/inventory/warranties/products/{id}/deactivate (Nonaktifkan garansi produk)
	mux.Handle("POST /api/v1/inventory/warranties/products/{id}/deactivate",
		authMiddleware(http.HandlerFunc(m.warrantyHandler.DeactivateProductWarranty)),
	)
}

