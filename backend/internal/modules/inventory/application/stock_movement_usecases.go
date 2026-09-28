package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/internal/shared/event"
	"github.com/erp-retail/backend/pkg/uid"
)

var (
	ErrInsufficientStockForMovement = errors.New("stok barang di gudang tidak mencukupi untuk dikeluarkan")
	ErrSerialUnitNotAvailable       = errors.New("satu atau lebih nomor seri yang dipilih tidak berstatus 'tersedia' di lokasi ini")
)

// StockMovementItemInput membawa payload per baris barang yang dimutasi.
type StockMovementItemInput struct {
	ProductID     string
	Quantity      int
	Notes         string
	SerialNumbers []string
}

// CreateStockMovementCommand membawa data formulir transaksi Barang Masuk atau Barang Keluar.
type CreateStockMovementCommand struct {
	Type            domain.StockMovementType
	MovementDate    time.Time
	LocationID      string
	CategoryReason  string
	ReferenceNumber string
	Notes           string
	ExecutedBy      string
	ExecutedByName  string
	Items           []StockMovementItemInput
}

// CreateStockMovementUseCase mengorkestrasi transaksi Barang Masuk (Stock In) atau Barang Keluar (Stock Out) secara atomik.
type CreateStockMovementUseCase struct {
	movementRepo domain.StockMovementRepository
	stockRepo    domain.StockRepository
	productRepo  domain.ProductRepository
	locationRepo domain.LocationRepository
	serialRepo   domain.SerialUnitRepository
	bus          event.Bus
}

// NewCreateStockMovementUseCase membuat instance baru CreateStockMovementUseCase.
func NewCreateStockMovementUseCase(
	movementRepo domain.StockMovementRepository,
	stockRepo domain.StockRepository,
	productRepo domain.ProductRepository,
	locationRepo domain.LocationRepository,
	serialRepo domain.SerialUnitRepository,
	bus event.Bus,
) *CreateStockMovementUseCase {
	return &CreateStockMovementUseCase{
		movementRepo: movementRepo,
		stockRepo:    stockRepo,
		productRepo:  productRepo,
		locationRepo: locationRepo,
		serialRepo:   serialRepo,
		bus:          bus,
	}
}

// Execute mengeksekusi validasi bisnis, mutasi saldo persediaan, dan pencatatan dokumen.
func (uc *CreateStockMovementUseCase) Execute(ctx context.Context, cmd CreateStockMovementCommand) (*domain.StockMovement, error) {
	if !cmd.Type.IsValid() {
		return nil, domain.ErrInvalidMovementType
	}
	if cmd.LocationID == "" {
		return nil, domain.ErrInvalidLocationID
	}
	if len(cmd.Items) == 0 {
		return nil, domain.ErrEmptyMovementItems
	}

	// 1. Validasi Lokasi Cabang/Gudang
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

	// 2. Buat item domain dan validasi keberadaan produk serta kecukupan stok
	var domainItems []*domain.StockMovementItem
	for _, itInput := range cmd.Items {
		product, err := uc.productRepo.FindByID(ctx, itInput.ProductID)
		if err != nil {
			return nil, fmt.Errorf("gagal memeriksa produk: %w", err)
		}
		if product == nil {
			return nil, fmt.Errorf("produk ID %s tidak ditemukan", itInput.ProductID)
		}

		// Jika produk mengaktifkan nomor seri dan user menyertakan serial numbers, pastikan jumlahnya cocok
		if product.FlagSerialTracking && len(itInput.SerialNumbers) > 0 {
			if len(itInput.SerialNumbers) != itInput.Quantity {
				return nil, fmt.Errorf("produk '%s' membutuhkan %d nomor seri, tetapi hanya %d yang disertakan",
					product.Name, itInput.Quantity, len(itInput.SerialNumbers))
			}
		}

		itemEntity, err := domain.NewStockMovementItem(
			uid.New(),
			itInput.ProductID,
			itInput.Quantity,
			itInput.Notes,
			itInput.SerialNumbers,
		)
		if err != nil {
			return nil, err
		}
		itemEntity.ProductName = product.Name
		itemEntity.ProductSKU = product.SKU
		domainItems = append(domainItems, itemEntity)
	}

	// 3. Generate Nomor Dokumen Transaksi Unik (IN-YYYYMM-XXXX / OUT-YYYYMM-XXXX)
	docNumber, err := uc.movementRepo.GenerateMovementNumber(ctx, cmd.Type)
	if err != nil {
		return nil, fmt.Errorf("gagal generate nomor dokumen transaksi: %w", err)
	}

	movementID := uid.New()
	movement, err := domain.NewStockMovement(
		movementID,
		docNumber,
		cmd.Type,
		cmd.MovementDate,
		cmd.LocationID,
		cmd.CategoryReason,
		cmd.ReferenceNumber,
		cmd.Notes,
		cmd.ExecutedBy,
		cmd.ExecutedByName,
		domainItems,
	)
	if err != nil {
		return nil, err
	}
	movement.LocationName = location.Name

	// 4. Mutasi Stok Fisik di inv_stocks secara Atomik per Produk
	for _, it := range domainItems {
		if cmd.Type == domain.MovementTypeIn {
			// BARANG MASUK: Tambah kuantitas fisik
			_, err = uc.stockRepo.AtomicMutate(ctx, it.ProductID, cmd.LocationID, func(stockItem *domain.StockItem) error {
				stockItem.Quantity += it.Quantity
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("gagal menambahkan stok produk '%s': %w", it.ProductName, err)
			}
		} else {
			// BARANG KELUAR: Periksa kuantitas dan kurangi stok fisik
			_, err = uc.stockRepo.AtomicMutate(ctx, it.ProductID, cmd.LocationID, func(stockItem *domain.StockItem) error {
				if stockItem.AvailableQuantity() < it.Quantity {
					return fmt.Errorf("%w: produk '%s' sisa tersedia %d unit, hendak dikeluarkan %d unit",
						ErrInsufficientStockForMovement, it.ProductName, stockItem.AvailableQuantity(), it.Quantity)
				}
				stockItem.Quantity -= it.Quantity
				return nil
			})
			if err != nil {
				return nil, err
			}

			// Jika barang keluar memiliki serial number yang dipilih, ubah status serial menjadi 'terjual'/'retur'
			if len(it.SerialNumbers) > 0 && uc.serialRepo != nil {
				for _, sn := range it.SerialNumbers {
					unit, errFind := uc.serialRepo.FindBySerialNumber(ctx, sn)
					if errFind == nil && unit != nil {
						if cmd.CategoryReason == "rusak_afkir" || cmd.CategoryReason == "retur_supplier" {
							unit.Status = domain.SerialStatusReturned
						} else {
							unit.Status = domain.SerialStatusSold
						}
						_ = uc.serialRepo.Update(ctx, unit)
					}
				}
			}
		}
	}

	// 5. Simpan Dokumen Transaksi ke Database
	if err := uc.movementRepo.Save(ctx, movement); err != nil {
		return nil, fmt.Errorf("gagal menyimpan dokumen transaksi pergerakan stok: %w", err)
	}

	// 6. Publikasikan Event ke Event Bus (Audit Log)
	if uc.bus != nil {
		uc.bus.Publish(event.EventStockMoved, event.StockMovedPayload{
			MovementID:      movement.ID,
			MovementNumber:  movement.MovementNumber,
			Type:            string(movement.Type),
			LocationID:      movement.LocationID,
			LocationName:    location.Name,
			CategoryReason:  movement.CategoryReason,
			ReferenceNumber: movement.ReferenceNumber,
			TotalItems:      len(domainItems),
			ExecutedBy:      cmd.ExecutedBy,
			ExecutedByName:  cmd.ExecutedByName,
		})
	}

	return movement, nil
}

// GetStockMovementDetailUseCase mengambil dokumen pergerakan stok lengkap berdasarkan ID.
type GetStockMovementDetailUseCase struct {
	movementRepo domain.StockMovementRepository
}

func NewGetStockMovementDetailUseCase(movementRepo domain.StockMovementRepository) *GetStockMovementDetailUseCase {
	return &GetStockMovementDetailUseCase{movementRepo: movementRepo}
}

func (uc *GetStockMovementDetailUseCase) Execute(ctx context.Context, id string) (*domain.StockMovement, error) {
	if id == "" {
		return nil, domain.ErrInvalidMovementID
	}
	return uc.movementRepo.FindByID(ctx, id)
}

// ListStockMovementsUseCase mengambil daftar transaksi Barang Masuk dan Keluar.
type ListStockMovementsUseCase struct {
	movementRepo domain.StockMovementRepository
}

func NewListStockMovementsUseCase(movementRepo domain.StockMovementRepository) *ListStockMovementsUseCase {
	return &ListStockMovementsUseCase{movementRepo: movementRepo}
}

func (uc *ListStockMovementsUseCase) Execute(ctx context.Context, filter domain.StockMovementFilter) ([]*domain.StockMovement, int, error) {
	return uc.movementRepo.List(ctx, filter)
}

// GetStockCardReportUseCase menghasilkan buku besar kartu stok per produk di cabang.
type GetStockCardReportUseCase struct {
	movementRepo domain.StockMovementRepository
}

func NewGetStockCardReportUseCase(movementRepo domain.StockMovementRepository) *GetStockCardReportUseCase {
	return &GetStockCardReportUseCase{movementRepo: movementRepo}
}

func (uc *GetStockCardReportUseCase) Execute(
	ctx context.Context,
	productID, locationID string,
	startDate, endDate *time.Time,
) (*domain.StockCardReport, error) {
	if productID == "" {
		return nil, domain.ErrInvalidProductID
	}
	if locationID == "" {
		return nil, domain.ErrInvalidLocationID
	}
	return uc.movementRepo.GetStockCardReport(ctx, productID, locationID, startDate, endDate)
}

// GetStockValuationReportUseCase menghasilkan laporan ringkasan valuasi persediaan di cabang.
type GetStockValuationReportUseCase struct {
	movementRepo domain.StockMovementRepository
}

func NewGetStockValuationReportUseCase(movementRepo domain.StockMovementRepository) *GetStockValuationReportUseCase {
	return &GetStockValuationReportUseCase{movementRepo: movementRepo}
}

func (uc *GetStockValuationReportUseCase) Execute(ctx context.Context, locationID *string) ([]*domain.StockValuationItem, error) {
	return uc.movementRepo.GetValuationReport(ctx, locationID)
}
