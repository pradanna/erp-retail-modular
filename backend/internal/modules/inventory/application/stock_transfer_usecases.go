package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/pkg/uid"
)

// CreateStockTransferCommand adalah parameter input untuk membuat permohonan mutasi stok.
type CreateStockTransferCommand struct {
	FromLocationID string
	ToLocationID   string
	Notes          string
	RequestedBy    string
	Items          []domain.StockTransferItemInput
}

// CreateStockTransferUseCase menangani alur pembuatan permohonan mutasi stok dan reservasi stok cabang asal.
type CreateStockTransferUseCase struct {
	transferRepo domain.StockTransferRepository
	stockRepo    domain.StockRepository
	locationRepo domain.LocationRepository
	productRepo  domain.ProductRepository
	serialRepo   domain.SerialUnitRepository
}

// NewCreateStockTransferUseCase membuat instance baru CreateStockTransferUseCase.
func NewCreateStockTransferUseCase(
	transferRepo domain.StockTransferRepository,
	stockRepo domain.StockRepository,
	locationRepo domain.LocationRepository,
	productRepo domain.ProductRepository,
	serialRepo domain.SerialUnitRepository,
) *CreateStockTransferUseCase {
	return &CreateStockTransferUseCase{
		transferRepo: transferRepo,
		stockRepo:    stockRepo,
		locationRepo: locationRepo,
		productRepo:  productRepo,
		serialRepo:   serialRepo,
	}
}

// Execute memvalidasi cabang asal/tujuan, ketersediaan stok fisik, mencadangkan stok (reserve), dan membuat dokumen transfer.
func (uc *CreateStockTransferUseCase) Execute(ctx context.Context, cmd CreateStockTransferCommand) (*domain.StockTransfer, error) {
	if cmd.FromLocationID == cmd.ToLocationID {
		return nil, domain.ErrSameLocationTransfer
	}

	// 1. Validasi lokasi asal & tujuan
	fromLoc, err := uc.locationRepo.FindByID(ctx, cmd.FromLocationID)
	if err != nil {
		return nil, err
	}
	if fromLoc == nil || !fromLoc.IsActive {
		return nil, fmt.Errorf("lokasi cabang asal tidak valid atau tidak aktif: %w", ErrLocationNotFound)
	}

	toLoc, err := uc.locationRepo.FindByID(ctx, cmd.ToLocationID)
	if err != nil {
		return nil, err
	}
	if toLoc == nil || !toLoc.IsActive {
		return nil, fmt.Errorf("lokasi cabang tujuan tidak valid atau tidak aktif: %w", ErrLocationNotFound)
	}

	if len(cmd.Items) == 0 {
		return nil, domain.ErrEmptyTransferItems
	}

	// 2. Validasi setiap produk dan ketersediaan stok
	for _, it := range cmd.Items {
		prod, err := uc.productRepo.FindByID(ctx, it.ProductID)
		if err != nil {
			return nil, err
		}
		if prod == nil {
			return nil, fmt.Errorf("produk ID '%s' tidak ditemukan: %w", it.ProductID, ErrProductNotFound)
		}
		if it.Quantity <= 0 {
			return nil, domain.ErrInvalidTransferQuantity
		}

		// 2b. Validasi aturan nomor seri (FlagSerialTracking)
		if prod.FlagSerialTracking {
			// Produk bernomor seri WAJIB menyertakan serial unit IDs sebanyak kuantitas yang dimutasi
			if len(it.SerialUnitIDs) == 0 {
				return nil, fmt.Errorf("produk '%s' memiliki pelacakan serial aktif: wajib menyertakan %d nomor seri/IMEI",
					prod.Name, it.Quantity)
			}
			if len(it.SerialUnitIDs) != it.Quantity {
				return nil, fmt.Errorf("jumlah nomor seri untuk '%s' tidak sesuai (dibutuhkan %d unit, disediakan %d)",
					prod.Name, it.Quantity, len(it.SerialUnitIDs))
			}

			seenSerialsInItem := make(map[string]bool)
			for _, sID := range it.SerialUnitIDs {
				if seenSerialsInItem[sID] {
					return nil, fmt.Errorf("nomor seri ID '%s' dicantumkan ganda pada produk '%s'", sID, prod.Name)
				}
				seenSerialsInItem[sID] = true

				unit, err := uc.serialRepo.FindByID(ctx, sID)
				if err != nil {
					return nil, err
				}
				if unit == nil {
					return nil, fmt.Errorf("unit fisik serial ID '%s' tidak ditemukan", sID)
				}
				if unit.LocationID != cmd.FromLocationID {
					return nil, fmt.Errorf("unit fisik '%s' tidak berada di cabang asal (lokasi unit: %s)", unit.SerialNumber, unit.LocationID)
				}
				if unit.Status != domain.SerialStatusAvailable {
					return nil, fmt.Errorf("unit fisik '%s' tidak berstatus tersedia (status saat ini: %s)",
						unit.SerialNumber, unit.Status)
				}
			}
		} else {
			// Produk non-serial tidak boleh menyertakan nomor seri
			if len(it.SerialUnitIDs) > 0 {
				return nil, fmt.Errorf("produk '%s' adalah barang non-serial (tidak memerlukan nomor seri)", prod.Name)
			}
		}
	}

	// 3. Cadangkan stok di cabang asal secara atomik (SELECT ... FOR UPDATE)
	for _, it := range cmd.Items {
		_, err := uc.stockRepo.AtomicMutate(ctx, it.ProductID, cmd.FromLocationID, func(item *domain.StockItem) error {
			return item.Reserve(it.Quantity)
		})
		if err != nil {
			return nil, fmt.Errorf("gagal mencadangkan stok produk: %w", err)
		}
	}

	// 4. Buat nomor surat jalan urut dan entitas StockTransfer
	trfNumber, err := uc.transferRepo.GenerateTransferNumber(ctx)
	if err != nil {
		return nil, err
	}

	transferID := uid.New()
	trf, err := domain.NewStockTransfer(
		transferID, trfNumber, cmd.FromLocationID, cmd.ToLocationID,
		cmd.Notes, cmd.RequestedBy, cmd.Items,
	)
	if err != nil {
		return nil, err
	}

	// 5. Simpan dokumen transfer
	if err := uc.transferRepo.Save(ctx, trf); err != nil {
		return nil, fmt.Errorf("gagal menyimpan dokumen transfer: %w", err)
	}

	return trf, nil
}

// ApproveStockTransferUseCase menangani persetujuan dokumen mutasi oleh Superadmin/Owner.
type ApproveStockTransferUseCase struct {
	transferRepo domain.StockTransferRepository
}

func NewApproveStockTransferUseCase(transferRepo domain.StockTransferRepository) *ApproveStockTransferUseCase {
	return &ApproveStockTransferUseCase{transferRepo: transferRepo}
}

func (uc *ApproveStockTransferUseCase) Execute(ctx context.Context, transferID, approvedBy string) (*domain.StockTransfer, error) {
	trf, err := uc.transferRepo.FindByID(ctx, transferID)
	if err != nil {
		return nil, err
	}

	if err := trf.Approve(approvedBy); err != nil {
		return nil, err
	}

	if err := uc.transferRepo.Update(ctx, trf); err != nil {
		return nil, err
	}

	return trf, nil
}

// RejectStockTransferUseCase menangani penolakan mutasi stok dan mengembalikan stok yang dicadangkan.
type RejectStockTransferUseCase struct {
	transferRepo domain.StockTransferRepository
	stockRepo    domain.StockRepository
}

func NewRejectStockTransferUseCase(
	transferRepo domain.StockTransferRepository,
	stockRepo domain.StockRepository,
) *RejectStockTransferUseCase {
	return &RejectStockTransferUseCase{
		transferRepo: transferRepo,
		stockRepo:    stockRepo,
	}
}

func (uc *RejectStockTransferUseCase) Execute(ctx context.Context, transferID, rejectedBy, reason string) (*domain.StockTransfer, error) {
	trf, err := uc.transferRepo.FindByID(ctx, transferID)
	if err != nil {
		return nil, err
	}

	if err := trf.Reject(rejectedBy, reason); err != nil {
		return nil, err
	}

	// Lepaskan cadangan stok di cabang asal secara atomik
	for _, it := range trf.Items {
		_, _ = uc.stockRepo.AtomicMutate(ctx, it.ProductID, trf.FromLocationID, func(item *domain.StockItem) error {
			return item.ReleaseReservation(it.Quantity)
		})
	}

	if err := uc.transferRepo.Update(ctx, trf); err != nil {
		return nil, err
	}

	return trf, nil
}

// ShipStockTransferUseCase menangani pengiriman barang (truk berangkat) dan memotong stok fisik cabang asal.
type ShipStockTransferUseCase struct {
	transferRepo domain.StockTransferRepository
	stockRepo    domain.StockRepository
}

func NewShipStockTransferUseCase(
	transferRepo domain.StockTransferRepository,
	stockRepo domain.StockRepository,
) *ShipStockTransferUseCase {
	return &ShipStockTransferUseCase{
		transferRepo: transferRepo,
		stockRepo:    stockRepo,
	}
}

func (uc *ShipStockTransferUseCase) Execute(ctx context.Context, transferID string) (*domain.StockTransfer, error) {
	trf, err := uc.transferRepo.FindByID(ctx, transferID)
	if err != nil {
		return nil, err
	}

	if err := trf.Ship(); err != nil {
		return nil, err
	}

	// Potong stok riil di cabang asal secara atomik (DeductReserved)
	for _, it := range trf.Items {
		_, err := uc.stockRepo.AtomicMutate(ctx, it.ProductID, trf.FromLocationID, func(item *domain.StockItem) error {
			return item.DeductReserved(it.Quantity)
		})
		if err != nil {
			return nil, fmt.Errorf("gagal memotong stok barang yang dikirim: %w", err)
		}
	}

	if err := uc.transferRepo.Update(ctx, trf); err != nil {
		return nil, err
	}

	return trf, nil
}

// ReceiveStockTransferUseCase menangani konfirmasi penerimaan barang di cabang tujuan,
// menambah stok fisik cabang tujuan, dan memindahkan lokasi unit berserial.
type ReceiveStockTransferUseCase struct {
	transferRepo domain.StockTransferRepository
	stockRepo    domain.StockRepository
	serialRepo   domain.SerialUnitRepository
}

func NewReceiveStockTransferUseCase(
	transferRepo domain.StockTransferRepository,
	stockRepo domain.StockRepository,
	serialRepo domain.SerialUnitRepository,
) *ReceiveStockTransferUseCase {
	return &ReceiveStockTransferUseCase{
		transferRepo: transferRepo,
		stockRepo:    stockRepo,
		serialRepo:   serialRepo,
	}
}

func (uc *ReceiveStockTransferUseCase) Execute(ctx context.Context, transferID, receivedBy string) (*domain.StockTransfer, error) {
	trf, err := uc.transferRepo.FindByID(ctx, transferID)
	if err != nil {
		return nil, err
	}

	if err := trf.Receive(receivedBy); err != nil {
		return nil, err
	}

	// 1. Tambah stok fisik di cabang tujuan secara atomik
	for _, it := range trf.Items {
		_, err := uc.stockRepo.AtomicMutate(ctx, it.ProductID, trf.ToLocationID, func(item *domain.StockItem) error {
			return item.AdjustQuantity(item.Quantity + it.Quantity)
		})
		if err != nil {
			return nil, fmt.Errorf("gagal menambah stok di cabang tujuan: %w", err)
		}

		// 2. Pindahkan lokasi fisik seluruh unit berserial yang ditransfer
		for _, sID := range it.SerialUnitIDs {
			unit, err := uc.serialRepo.FindByID(ctx, sID)
			if err != nil {
				return nil, err
			}
			if unit != nil {
				if err := unit.TransferLocation(trf.ToLocationID); err != nil {
					return nil, fmt.Errorf("gagal memindahkan lokasi unit serial '%s': %w", unit.SerialNumber, err)
				}
				if err := uc.serialRepo.Update(ctx, unit); err != nil {
					return nil, fmt.Errorf("gagal menyimpan update lokasi serial: %w", err)
				}
			}
		}
	}

	// 3. Simpan perubahan status dokumen transfer ke database
	if err := uc.transferRepo.Update(ctx, trf); err != nil {
		return nil, err
	}

	return trf, nil
}

// GetStockTransferUseCase mengambil detail 1 dokumen transfer berdasarkan UUID atau nomor surat jalan.
type GetStockTransferUseCase struct {
	transferRepo domain.StockTransferRepository
}

func NewGetStockTransferUseCase(transferRepo domain.StockTransferRepository) *GetStockTransferUseCase {
	return &GetStockTransferUseCase{transferRepo: transferRepo}
}

func (uc *GetStockTransferUseCase) Execute(ctx context.Context, idOrNumber string) (*domain.StockTransfer, error) {
	if idOrNumber == "" {
		return nil, domain.ErrInvalidTransferID
	}

	// Coba cari by ID dulu, jika tidak ketemu coba by Number
	trf, err := uc.transferRepo.FindByID(ctx, idOrNumber)
	if err == nil {
		return trf, nil
	}
	if errors.Is(err, domain.ErrTransferNotFound) {
		return uc.transferRepo.FindByNumber(ctx, idOrNumber)
	}
	return nil, err
}

// ListStockTransfersUseCase mengambil daftar dokumen mutasi stok berdasarkan filter.
type ListStockTransfersUseCase struct {
	transferRepo domain.StockTransferRepository
}

func NewListStockTransfersUseCase(transferRepo domain.StockTransferRepository) *ListStockTransfersUseCase {
	return &ListStockTransfersUseCase{transferRepo: transferRepo}
}

func (uc *ListStockTransfersUseCase) Execute(ctx context.Context, filter domain.StockTransferFilter) ([]*domain.StockTransfer, error) {
	return uc.transferRepo.List(ctx, filter)
}
