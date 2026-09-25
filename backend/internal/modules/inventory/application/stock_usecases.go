package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/internal/shared/event"
	"github.com/erp-retail/backend/pkg/uid"
)

var (
	ErrLocationInactive = errors.New("lokasi cabang sedang tidak aktif")
)

// AdjustStockCommand membawa data input untuk penyesuaian stok (Stock Opname).
type AdjustStockCommand struct {
	ProductID      string
	LocationID     string
	NewQuantity    int
	Reason         string
	AdjustedBy     string
	AdjustedByName string
}

// AdjustStockUseCase menangani penyesuaian kuantitas fisik stok per lokasi dengan row-level locking.
type AdjustStockUseCase struct {
	stockRepo      domain.StockRepository
	productRepo    domain.ProductRepository
	locationRepo   domain.LocationRepository
	adjustmentRepo domain.StockAdjustmentRepository
	bus            event.Bus
}

func NewAdjustStockUseCase(
	stockRepo domain.StockRepository,
	productRepo domain.ProductRepository,
	locationRepo domain.LocationRepository,
	adjustmentRepo domain.StockAdjustmentRepository,
	bus event.Bus,
) *AdjustStockUseCase {
	return &AdjustStockUseCase{
		stockRepo:      stockRepo,
		productRepo:    productRepo,
		locationRepo:   locationRepo,
		adjustmentRepo: adjustmentRepo,
		bus:            bus,
	}
}

func (uc *AdjustStockUseCase) Execute(ctx context.Context, cmd AdjustStockCommand) (*domain.StockItem, error) {
	if cmd.ProductID == "" {
		return nil, domain.ErrInvalidProductID
	}
	if cmd.LocationID == "" {
		return nil, domain.ErrInvalidLocationID
	}

	// 1. Validasi produk ada di database
	product, err := uc.productRepo.FindByID(ctx, cmd.ProductID)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa produk: %w", err)
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	// 2. Validasi lokasi ada dan aktif
	location, err := uc.locationRepo.FindByID(ctx, cmd.LocationID)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa lokasi: %w", err)
	}
	if location == nil {
		return nil, ErrLocationNotFound
	}
	if !location.IsActive {
		return nil, ErrLocationInactive
	}

	// 3. Mutasi stok secara atomik dengan SELECT ... FOR UPDATE
	var previousQty int
	item, err := uc.stockRepo.AtomicMutate(ctx, cmd.ProductID, cmd.LocationID, func(item *domain.StockItem) error {
		previousQty = item.Quantity
		return item.AdjustQuantity(cmd.NewQuantity)
	})
	if err != nil {
		return nil, err
	}

	// 4. Rekam jejak riwayat Stock Opname (Audit Trail Ledger)
	adjID := uid.New()
	reason := cmd.Reason
	if reason == "" {
		reason = "Penyesuaian Fisik Stok (Stock Opname)"
	}
	adjustedByName := cmd.AdjustedByName
	if adjustedByName == "" {
		adjustedByName = "Staf Toko"
	}

	if uc.adjustmentRepo != nil {
		adjustment, err := domain.NewStockAdjustment(
			adjID,
			cmd.ProductID,
			cmd.LocationID,
			previousQty,
			cmd.NewQuantity,
			reason,
			cmd.AdjustedBy,
			adjustedByName,
		)
		if err == nil {
			_ = uc.adjustmentRepo.Save(ctx, adjustment)
		}
	}

	// 5. Publish event ke Event Bus agar Audit Log sistem dan modul lain mencatatnya
	if uc.bus != nil {
		uc.bus.Publish(event.EventStockAdjusted, event.StockAdjustedPayload{
			AdjustmentID:   adjID,
			ProductID:      cmd.ProductID,
			ProductName:    product.Name,
			LocationID:     cmd.LocationID,
			LocationName:   location.Name,
			PreviousQty:    previousQty,
			NewQty:         cmd.NewQuantity,
			Difference:     cmd.NewQuantity - previousQty,
			Reason:         reason,
			AdjustedBy:     cmd.AdjustedBy,
			AdjustedByName: adjustedByName,
		})
	}

	return item, nil
}

// ListStockAdjustmentsUseCase menangani pembacaan riwayat catatan stock opname.
type ListStockAdjustmentsUseCase struct {
	adjustmentRepo domain.StockAdjustmentRepository
}

func NewListStockAdjustmentsUseCase(adjustmentRepo domain.StockAdjustmentRepository) *ListStockAdjustmentsUseCase {
	return &ListStockAdjustmentsUseCase{adjustmentRepo: adjustmentRepo}
}

func (uc *ListStockAdjustmentsUseCase) Execute(ctx context.Context, filter domain.StockAdjustmentFilter) ([]*domain.StockAdjustment, int, error) {
	return uc.adjustmentRepo.List(ctx, filter)
}

// GetStockUseCase menangani query stok spesifik untuk 1 produk di 1 cabang.
type GetStockUseCase struct {
	stockRepo domain.StockRepository
}

func NewGetStockUseCase(stockRepo domain.StockRepository) *GetStockUseCase {
	return &GetStockUseCase{stockRepo: stockRepo}
}

func (uc *GetStockUseCase) Execute(ctx context.Context, productID, locationID string) (*domain.StockItem, error) {
	if productID == "" {
		return nil, domain.ErrInvalidProductID
	}
	if locationID == "" {
		return nil, domain.ErrInvalidLocationID
	}

	item, err := uc.stockRepo.FindByProductAndLocation(ctx, productID, locationID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data stok: %w", err)
	}

	// Jika produk belum pernah dicatat stoknya di lokasi ini, kembalikan objek stok bernilai 0
	if item == nil {
		return domain.NewStockItem(uid.New(), productID, locationID, 0, 0)
	}

	return item, nil
}

// ListStockByLocationUseCase menangani pengambilan seluruh daftar stok pada satu lokasi cabang/gudang.
type ListStockByLocationUseCase struct {
	stockRepo    domain.StockRepository
	locationRepo domain.LocationRepository
}

func NewListStockByLocationUseCase(
	stockRepo domain.StockRepository,
	locationRepo domain.LocationRepository,
) *ListStockByLocationUseCase {
	return &ListStockByLocationUseCase{
		stockRepo:    stockRepo,
		locationRepo: locationRepo,
	}
}

func (uc *ListStockByLocationUseCase) Execute(ctx context.Context, locationID string) ([]*domain.StockItem, error) {
	if locationID == "" {
		return nil, domain.ErrInvalidLocationID
	}

	// Validasi lokasi ada
	location, err := uc.locationRepo.FindByID(ctx, locationID)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa lokasi: %w", err)
	}
	if location == nil {
		return nil, ErrLocationNotFound
	}

	return uc.stockRepo.ListByLocation(ctx, locationID)
}

// ListLowStockAlertsUseCase menangani pencarian stok yang menipis (kuantitas <= min_stock).
type ListLowStockAlertsUseCase struct {
	stockRepo domain.StockRepository
}

func NewListLowStockAlertsUseCase(stockRepo domain.StockRepository) *ListLowStockAlertsUseCase {
	return &ListLowStockAlertsUseCase{stockRepo: stockRepo}
}

func (uc *ListLowStockAlertsUseCase) Execute(ctx context.Context, locationID *string) ([]*domain.StockItem, error) {
	return uc.stockRepo.ListLowStockAlerts(ctx, locationID)
}

// UpdateMinStockCommand membawa data untuk mengubah ambang batas minimum stok.
type UpdateMinStockCommand struct {
	ProductID string
	LocationID string
	MinStock  int
}

// UpdateMinStockUseCase menangani pembaruan ambang batas minimum stok.
type UpdateMinStockUseCase struct {
	stockRepo    domain.StockRepository
	productRepo  domain.ProductRepository
	locationRepo domain.LocationRepository
}

func NewUpdateMinStockUseCase(
	stockRepo domain.StockRepository,
	productRepo domain.ProductRepository,
	locationRepo domain.LocationRepository,
) *UpdateMinStockUseCase {
	return &UpdateMinStockUseCase{
		stockRepo:    stockRepo,
		productRepo:  productRepo,
		locationRepo: locationRepo,
	}
}

func (uc *UpdateMinStockUseCase) Execute(ctx context.Context, cmd UpdateMinStockCommand) (*domain.StockItem, error) {
	if cmd.ProductID == "" {
		return nil, domain.ErrInvalidProductID
	}
	if cmd.LocationID == "" {
		return nil, domain.ErrInvalidLocationID
	}

	// 1. Validasi produk ada di database
	product, err := uc.productRepo.FindByID(ctx, cmd.ProductID)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa produk: %w", err)
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	// 2. Validasi lokasi ada di database
	location, err := uc.locationRepo.FindByID(ctx, cmd.LocationID)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa lokasi: %w", err)
	}
	if location == nil {
		return nil, ErrLocationNotFound
	}

	// 3. Update min_stock secara atomik
	item, err := uc.stockRepo.AtomicMutate(ctx, cmd.ProductID, cmd.LocationID, func(item *domain.StockItem) error {
		return item.UpdateMinStock(cmd.MinStock)
	})
	if err != nil {
		return nil, err
	}

	return item, nil
}
