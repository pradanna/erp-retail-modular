package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/pkg/uid"
)

// RegisterSerialUnitsCommand adalah parameter input untuk mendaftarkan serial unit baru.
type RegisterSerialUnitsCommand struct {
	ProductID     string
	LocationID    string
	SerialNumbers []string
}

// RegisterSerialUnitsUseCase menangani pendaftaran nomor seri baru saat barang datang di gudang/toko.
type RegisterSerialUnitsUseCase struct {
	serialRepo   domain.SerialUnitRepository
	productRepo  domain.ProductRepository
	locationRepo domain.LocationRepository
}

// NewRegisterSerialUnitsUseCase membuat instance baru RegisterSerialUnitsUseCase.
func NewRegisterSerialUnitsUseCase(
	serialRepo domain.SerialUnitRepository,
	productRepo domain.ProductRepository,
	locationRepo domain.LocationRepository,
) *RegisterSerialUnitsUseCase {
	return &RegisterSerialUnitsUseCase{
		serialRepo:   serialRepo,
		productRepo:  productRepo,
		locationRepo: locationRepo,
	}
}

// Execute menjalankan logika bisnis pendaftaran serial number/IMEI.
func (uc *RegisterSerialUnitsUseCase) Execute(ctx context.Context, cmd RegisterSerialUnitsCommand) ([]*domain.SerialUnit, error) {
	if len(cmd.SerialNumbers) == 0 {
		return nil, errors.New("daftar nomor seri tidak boleh kosong")
	}

	// 1. Validasi produk ada dan mengaktifkan flag_serial_tracking
	product, err := uc.productRepo.FindByID(ctx, cmd.ProductID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}
	if !product.FlagSerialTracking {
		return nil, domain.ErrProductNotTrackedBySerial
	}

	// 2. Validasi lokasi ada dan aktif
	location, err := uc.locationRepo.FindByID(ctx, cmd.LocationID)
	if err != nil {
		return nil, err
	}
	if location == nil {
		return nil, ErrLocationNotFound
	}
	if !location.IsActive {
		return nil, errors.New("lokasi cabang/gudang tidak aktif")
	}

	// 3. Validasi keunikan nomor seri di dalam daftar input itu sendiri
	seen := make(map[string]bool)
	var units []*domain.SerialUnit

	for _, rawSN := range cmd.SerialNumbers {
		cleanSN := strings.TrimSpace(rawSN)
		if cleanSN == "" {
			continue
		}
		if seen[cleanSN] {
			return nil, fmt.Errorf("%w: duplikat pada input '%s'", domain.ErrDuplicateSerialNumber, cleanSN)
		}
		seen[cleanSN] = true

		// Cek apakah nomor seri ini sudah pernah terdaftar di database
		existing, err := uc.serialRepo.FindBySerialNumber(ctx, cleanSN)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("%w: '%s'", domain.ErrDuplicateSerialNumber, cleanSN)
		} else if err != nil && !errors.Is(err, domain.ErrSerialNotFound) {
			return nil, fmt.Errorf("gagal verifikasi keunikan serial number: %w", err)
		}

		unit, err := domain.NewSerialUnit(uid.New(), cmd.ProductID, cmd.LocationID, cleanSN)
		if err != nil {
			return nil, err
		}
		units = append(units, unit)
	}

	if len(units) == 0 {
		return nil, errors.New("tidak ada nomor seri valid yang diproses")
	}

	// 4. Batch persist ke database
	if err := uc.serialRepo.BatchSave(ctx, units); err != nil {
		return nil, err
	}

	return units, nil
}

// SerialUnitDetail menampung data gabungan serial unit beserta info nama produk dan lokasi untuk scanner kasir.
type SerialUnitDetail struct {
	Unit         *domain.SerialUnit
	ProductName  string
	ProductSKU   string
	ProductBrand string
	LocationName string
	LocationCode string
}

// LookupSerialNumberUseCase menangani fast lookup unit fisik berdasarkan scan barcode S/N atau IMEI.
type LookupSerialNumberUseCase struct {
	serialRepo   domain.SerialUnitRepository
	productRepo  domain.ProductRepository
	locationRepo domain.LocationRepository
}

// NewLookupSerialNumberUseCase membuat instance baru LookupSerialNumberUseCase.
func NewLookupSerialNumberUseCase(
	serialRepo domain.SerialUnitRepository,
	productRepo domain.ProductRepository,
	locationRepo domain.LocationRepository,
) *LookupSerialNumberUseCase {
	return &LookupSerialNumberUseCase{
		serialRepo:   serialRepo,
		productRepo:  productRepo,
		locationRepo: locationRepo,
	}
}

// Execute mencari unit fisik dan melengkapinya dengan info produk dan cabang.
func (uc *LookupSerialNumberUseCase) Execute(ctx context.Context, sn string) (*SerialUnitDetail, error) {
	cleanSN := strings.TrimSpace(sn)
	if cleanSN == "" {
		return nil, domain.ErrInvalidSerialNumber
	}

	unit, err := uc.serialRepo.FindBySerialNumber(ctx, cleanSN)
	if err != nil {
		return nil, err
	}

	detail := &SerialUnitDetail{Unit: unit}

	if product, err := uc.productRepo.FindByID(ctx, unit.ProductID); err == nil && product != nil {
		detail.ProductName = product.Name
		detail.ProductSKU = product.SKU
		detail.ProductBrand = product.Brand
	}

	if location, err := uc.locationRepo.FindByID(ctx, unit.LocationID); err == nil && location != nil {
		detail.LocationName = location.Name
		detail.LocationCode = location.Code
	}

	return detail, nil
}

// ListSerialUnitsQuery adalah parameter pencarian daftar serial unit.
type ListSerialUnitsQuery struct {
	ProductID  *string
	LocationID *string
	Status     *domain.SerialStatus
	Search     *string
}

// ListSerialUnitsUseCase menangani pengambilan daftar serial unit fisik dengan filter dan metadata produk & lokasi.
type ListSerialUnitsUseCase struct {
	serialRepo   domain.SerialUnitRepository
	productRepo  domain.ProductRepository
	locationRepo domain.LocationRepository
}

// NewListSerialUnitsUseCase membuat instance baru ListSerialUnitsUseCase.
func NewListSerialUnitsUseCase(
	serialRepo domain.SerialUnitRepository,
	productRepo domain.ProductRepository,
	locationRepo domain.LocationRepository,
) *ListSerialUnitsUseCase {
	return &ListSerialUnitsUseCase{
		serialRepo:   serialRepo,
		productRepo:  productRepo,
		locationRepo: locationRepo,
	}
}

// Execute mengambil daftar unit fisik beserta metadata produk dan cabang.
func (uc *ListSerialUnitsUseCase) Execute(ctx context.Context, q ListSerialUnitsQuery) ([]*SerialUnitDetail, error) {
	units, err := uc.serialRepo.List(ctx, q.ProductID, q.LocationID, q.Status, q.Search)
	if err != nil {
		return nil, err
	}

	prodCache := make(map[string]*domain.Product)
	locCache := make(map[string]*domain.Location)
	details := make([]*SerialUnitDetail, 0, len(units))

	for _, u := range units {
		d := &SerialUnitDetail{Unit: u}

		if p, ok := prodCache[u.ProductID]; ok {
			if p != nil {
				d.ProductName = p.Name
				d.ProductSKU = p.SKU
				d.ProductBrand = p.Brand
			}
		} else {
			if p, err := uc.productRepo.FindByID(ctx, u.ProductID); err == nil && p != nil {
				prodCache[u.ProductID] = p
				d.ProductName = p.Name
				d.ProductSKU = p.SKU
				d.ProductBrand = p.Brand
			} else {
				prodCache[u.ProductID] = nil
			}
		}

		if l, ok := locCache[u.LocationID]; ok {
			if l != nil {
				d.LocationName = l.Name
				d.LocationCode = l.Code
			}
		} else {
			if l, err := uc.locationRepo.FindByID(ctx, u.LocationID); err == nil && l != nil {
				locCache[u.LocationID] = l
				d.LocationName = l.Name
				d.LocationCode = l.Code
			} else {
				locCache[u.LocationID] = nil
			}
		}

		details = append(details, d)
	}

	return details, nil
}

// UpdateSerialStatusCommand adalah parameter untuk memperbarui status serial unit.
type UpdateSerialStatusCommand struct {
	ID        string
	NewStatus domain.SerialStatus
}

// UpdateSerialStatusUseCase menangani transisi state machine unit fisik.
type UpdateSerialStatusUseCase struct {
	serialRepo domain.SerialUnitRepository
}

// NewUpdateSerialStatusUseCase membuat instance baru UpdateSerialStatusUseCase.
func NewUpdateSerialStatusUseCase(serialRepo domain.SerialUnitRepository) *UpdateSerialStatusUseCase {
	return &UpdateSerialStatusUseCase{serialRepo: serialRepo}
}

// Execute mengeksekusi perubahan status unit sesuai aturan bisnis domain.
func (uc *UpdateSerialStatusUseCase) Execute(ctx context.Context, cmd UpdateSerialStatusCommand) (*domain.SerialUnit, error) {
	unit, err := uc.serialRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}

	normStatus := domain.NormalizeSerialStatus(string(cmd.NewStatus))
	switch normStatus {
	case domain.SerialStatusAvailable:
		if err := unit.MarkAsAvailable(); err != nil {
			return nil, err
		}
	case domain.SerialStatusSold:
		if err := unit.MarkAsSold(); err != nil {
			return nil, err
		}
	case domain.SerialStatusReturned:
		if err := unit.MarkAsReturned(); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("transisi status ke '%s' tidak didukung secara langsung", cmd.NewStatus)
	}

	if err := uc.serialRepo.Update(ctx, unit); err != nil {
		return nil, err
	}

	return unit, nil
}
