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
	"github.com/erp-retail/backend/internal/shared/auth"
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
	productImageHandler  *inventoryHTTP.ProductImageHandler
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
	stockAdjustmentRepo := infrastructure.NewStockAdjustmentRepository(db)

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
	adjustStockUC := application.NewAdjustStockUseCase(stockRepo, productRepo, locationRepo, stockAdjustmentRepo, bus)
	listStockAdjustmentsUC := application.NewListStockAdjustmentsUseCase(stockAdjustmentRepo)
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
	listSerialUC := application.NewListSerialUnitsUseCase(serialRepo, productRepo, locationRepo)
	updateSerialStatusUC := application.NewUpdateSerialStatusUseCase(serialRepo)

	// Layer 2: Application — use cases Price Override
	createOverrideUC := application.NewCreatePriceOverrideUseCase(priceOverrideRepo, productRepo, locationRepo)
	getEffectivePriceUC := application.NewGetEffectivePriceUseCase(priceOverrideRepo, productRepo, locationRepo)
	listOverridesUC := application.NewListPriceOverridesUseCase(priceOverrideRepo, productRepo, locationRepo)
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
		listStockAdjustmentsUC,
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

	productImageRepo := infrastructure.NewProductImageRepository(db)
	uploadProductImageUC := application.NewUploadProductImageUseCase(productRepo, productImageRepo)
	listProductImagesUC := application.NewListProductImagesUseCase(productImageRepo)
	deleteProductImageUC := application.NewDeleteProductImageUseCase(productImageRepo)
	setPrimaryProductImageUC := application.NewSetPrimaryProductImageUseCase(productImageRepo)
	productImageHandler := inventoryHTTP.NewProductImageHandler(
		uploadProductImageUC,
		listProductImagesUC,
		deleteProductImageUC,
		setPrimaryProductImageUC,
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
		productImageHandler:  productImageHandler,
	}
}

// Register mendaftarkan semua route HTTP modul Inventory ke router utama.
// Fungsi ini dipanggil HANYA oleh main.go, dan HANYA jika modul Inventory aktif di lisensi.
func (m *Module) Register(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
	permService auth.PermissionService,
	userResolver auth.UserResolver,
) {
	// Suntikkan dependency ke handler yang membutuhkan
	m.productHandler.SetPermissionService(permService)
	m.stockTransferHandler.SetUserResolver(userResolver)

	// require merangkai autentikasi JWT dan otorisasi granular PBAC
	require := func(permission string, handler http.HandlerFunc) http.Handler {
		return authMiddleware(auth.RequirePermission(permission, permService)(handler))
	}

	// --- Routes Produk ---
	mux.Handle("POST /api/v1/inventory/products", require("inventory.products.create", m.productHandler.Create))
	mux.Handle("GET /api/v1/inventory/products", require("inventory.products.view", m.productHandler.List))
	mux.Handle("GET /api/v1/inventory/products/{id}", require("inventory.products.view", m.productHandler.GetByID))
	mux.Handle("PUT /api/v1/inventory/products/{id}", require("inventory.products.edit", m.productHandler.Update))
	mux.Handle("PATCH /api/v1/inventory/products/{id}/status", require("inventory.products.status", m.productHandler.SetStatus))

	// --- Routes Foto Produk ---
	mux.Handle("POST /api/v1/inventory/products/{id}/images", require("inventory.products.edit", m.productImageHandler.Upload))
	mux.Handle("GET /api/v1/inventory/products/{id}/images", require("inventory.products.view", m.productImageHandler.ListByProduct))
	mux.Handle("DELETE /api/v1/inventory/products/{id}/images/{image_id}", require("inventory.products.edit", m.productImageHandler.Delete))
	mux.Handle("PATCH /api/v1/inventory/products/{id}/images/{image_id}/primary", require("inventory.products.edit", m.productImageHandler.SetPrimary))

	// --- Routes Kategori ---
	mux.Handle("POST /api/v1/inventory/categories/upload-image", require("inventory.categories.create", m.categoryHandler.UploadImage))
	mux.Handle("POST /api/v1/inventory/categories", require("inventory.categories.create", m.categoryHandler.Create))
	mux.Handle("GET /api/v1/inventory/categories", require("inventory.categories.view", m.categoryHandler.List))
	mux.Handle("PUT /api/v1/inventory/categories/{id}", require("inventory.categories.edit", m.categoryHandler.Update))
	mux.Handle("DELETE /api/v1/inventory/categories/{id}", require("inventory.categories.delete", m.categoryHandler.Delete))

	// --- Routes Lokasi / Cabang ---
	mux.Handle("POST /api/v1/inventory/locations", require("inventory.locations.create", m.locationHandler.Create))
	mux.Handle("GET /api/v1/inventory/locations", require("inventory.locations.view", m.locationHandler.List))
	mux.Handle("GET /api/v1/inventory/locations/{id}", require("inventory.locations.view", m.locationHandler.GetByID))
	mux.Handle("PUT /api/v1/inventory/locations/{id}", require("inventory.locations.edit", m.locationHandler.Update))
	mux.Handle("PATCH /api/v1/inventory/locations/{id}/status", require("inventory.locations.status", m.locationHandler.SetStatus))
	mux.Handle("DELETE /api/v1/inventory/locations/{id}", require("inventory.locations.delete", m.locationHandler.Delete))

	// --- Routes Stok per Cabang / Lokasi ---
	mux.Handle("POST /api/v1/inventory/stocks/adjust", require("inventory.stocks.adjust", m.stockHandler.Adjust))
	mux.Handle("GET /api/v1/inventory/stocks/adjustments", require("inventory.stocks.view", m.stockHandler.ListAdjustments))
	mux.Handle("PUT /api/v1/inventory/stocks/min-stock", require("inventory.stocks.min_stock", m.stockHandler.UpdateMinStock))
	mux.Handle("GET /api/v1/inventory/stocks", require("inventory.stocks.view", m.stockHandler.Get))
	mux.Handle("GET /api/v1/inventory/stocks/alerts", require("inventory.stocks.view", m.stockHandler.ListAlerts))

	// --- Routes Barcode Produk ---
	mux.Handle("POST /api/v1/inventory/products/{id}/barcodes", require("inventory.barcodes.manage", m.barcodeHandler.Add))
	mux.Handle("GET /api/v1/inventory/products/{id}/barcodes", require("inventory.barcodes.view", m.barcodeHandler.ListByProduct))
	mux.Handle("DELETE /api/v1/inventory/products/{id}/barcodes/{barcode_id}", require("inventory.barcodes.manage", m.barcodeHandler.Delete))
	mux.Handle("GET /api/v1/inventory/barcodes/lookup", require("inventory.barcodes.view", m.barcodeHandler.Lookup))

	// --- Routes Serial Number & IMEI Tracking ---
	mux.Handle("GET /api/v1/inventory/serials", require("inventory.serials.view", m.serialUnitHandler.List))
	mux.Handle("POST /api/v1/inventory/products/{id}/serials", require("inventory.serials.register", m.serialUnitHandler.Register))
	mux.Handle("GET /api/v1/inventory/products/{id}/serials", require("inventory.serials.view", m.serialUnitHandler.ListByProduct))
	mux.Handle("GET /api/v1/inventory/serials/lookup", require("inventory.serials.view", m.serialUnitHandler.Lookup))
	mux.Handle("PATCH /api/v1/inventory/serials/{id}/status", require("inventory.serials.status", m.serialUnitHandler.UpdateStatus))

	// --- Routes Price Override (Promo / Harga Khusus Cabang) ---
	mux.Handle("GET /api/v1/inventory/price-overrides", require("inventory.prices.view", m.priceOverrideHandler.ListAll))
	mux.Handle("POST /api/v1/inventory/products/{id}/price-overrides", require("inventory.prices.create", m.priceOverrideHandler.Create))
	mux.Handle("GET /api/v1/inventory/products/{id}/price-overrides", require("inventory.prices.view", m.priceOverrideHandler.List))
	mux.Handle("GET /api/v1/inventory/price-overrides/effective-price", require("inventory.prices.view", m.priceOverrideHandler.GetEffectivePrice))
	mux.Handle("PATCH /api/v1/inventory/price-overrides/{id}/deactivate", require("inventory.prices.deactivate", m.priceOverrideHandler.Deactivate))
	mux.Handle("POST /api/v1/inventory/price-overrides/{id}/claim", require("inventory.prices.claim", m.priceOverrideHandler.ClaimQuota))

	// --- Routes Stock Transfer (Mutasi Stok Antar Cabang) ---
	mux.Handle("POST /api/v1/inventory/transfers", require("inventory.transfers.create", m.stockTransferHandler.Create))
	mux.Handle("GET /api/v1/inventory/transfers", require("inventory.transfers.view", m.stockTransferHandler.List))
	mux.Handle("GET /api/v1/inventory/transfers/{id}", require("inventory.transfers.view", m.stockTransferHandler.Get))
	mux.Handle("POST /api/v1/inventory/transfers/{id}/approve", require("inventory.transfers.approve", m.stockTransferHandler.Approve))
	mux.Handle("POST /api/v1/inventory/transfers/{id}/reject", require("inventory.transfers.approve", m.stockTransferHandler.Reject))
	mux.Handle("POST /api/v1/inventory/transfers/{id}/ship", require("inventory.transfers.ship", m.stockTransferHandler.Ship))
	mux.Handle("POST /api/v1/inventory/transfers/{id}/receive", require("inventory.transfers.receive", m.stockTransferHandler.Receive))

	// --- Routes Warranty (Garansi Toko & Pabrik) ---
	mux.Handle("POST /api/v1/inventory/warranties/policies", require("inventory.warranties.manage", m.warrantyHandler.CreatePolicy))
	mux.Handle("GET /api/v1/inventory/warranties/policies", require("inventory.warranties.view", m.warrantyHandler.ListPolicies))
	mux.Handle("POST /api/v1/inventory/products/{id}/warranties", require("inventory.warranties.manage", m.warrantyHandler.AssignProductWarranty))
	mux.Handle("GET /api/v1/inventory/products/{id}/warranties", require("inventory.warranties.view", m.warrantyHandler.GetProductWarranties))
	mux.Handle("POST /api/v1/inventory/warranties/products/{id}/deactivate", require("inventory.warranties.manage", m.warrantyHandler.DeactivateProductWarranty))
}

