package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/pkg/uid"
)

// AddBarcodeCommand membawa data untuk mendaftarkan barcode pabrik baru ke suatu produk.
type AddBarcodeCommand struct {
	ProductID string
	Barcode   string
	IsPrimary bool
}

// AddBarcodeUseCase menangani pendaftaran barcode ke produk.
type AddBarcodeUseCase struct {
	barcodeRepo domain.BarcodeRepository
	productRepo domain.ProductRepository
}

func NewAddBarcodeUseCase(
	barcodeRepo domain.BarcodeRepository,
	productRepo domain.ProductRepository,
) *AddBarcodeUseCase {
	return &AddBarcodeUseCase{
		barcodeRepo: barcodeRepo,
		productRepo: productRepo,
	}
}

func (uc *AddBarcodeUseCase) Execute(ctx context.Context, cmd AddBarcodeCommand) (*domain.ProductBarcode, error) {
	if cmd.ProductID == "" {
		return nil, domain.ErrInvalidProductID
	}

	// 1. Validasi produk induk ada di database
	product, err := uc.productRepo.FindByID(ctx, cmd.ProductID)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa produk: %w", err)
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	// 2. Validasi keunikan barcode di seluruh sistem
	existing, err := uc.barcodeRepo.FindByBarcode(ctx, cmd.Barcode)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa keunikan barcode: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: %s (sudah dipakai di produk lain)", domain.ErrDuplicateBarcode, cmd.Barcode)
	}

	// 3. Jika barcode ini dijadikan primary, reset primary lama milik produk ini
	if cmd.IsPrimary {
		if err := uc.barcodeRepo.ResetPrimary(ctx, cmd.ProductID); err != nil {
			return nil, fmt.Errorf("gagal mereset primary barcode lama: %w", err)
		}
	}

	// 4. Buat entitas domain ProductBarcode
	barcodeID := uid.New()
	barcode, err := domain.NewProductBarcode(barcodeID, cmd.ProductID, cmd.Barcode, cmd.IsPrimary)
	if err != nil {
		return nil, err
	}

	// 5. Simpan ke database
	if err := uc.barcodeRepo.Save(ctx, barcode); err != nil {
		return nil, fmt.Errorf("gagal menyimpan barcode: %w", err)
	}

	return barcode, nil
}

// DeleteBarcodeUseCase menangani penghapusan barcode dari produk.
type DeleteBarcodeUseCase struct {
	barcodeRepo domain.BarcodeRepository
}

func NewDeleteBarcodeUseCase(barcodeRepo domain.BarcodeRepository) *DeleteBarcodeUseCase {
	return &DeleteBarcodeUseCase{barcodeRepo: barcodeRepo}
}

func (uc *DeleteBarcodeUseCase) Execute(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("ID barcode wajib diisi")
	}
	return uc.barcodeRepo.Delete(ctx, id)
}

// ListBarcodesByProductUseCase mengambil seluruh barcode milik produk tertentu.
type ListBarcodesByProductUseCase struct {
	barcodeRepo domain.BarcodeRepository
	productRepo domain.ProductRepository
}

func NewListBarcodesByProductUseCase(
	barcodeRepo domain.BarcodeRepository,
	productRepo domain.ProductRepository,
) *ListBarcodesByProductUseCase {
	return &ListBarcodesByProductUseCase{
		barcodeRepo: barcodeRepo,
		productRepo: productRepo,
	}
}

func (uc *ListBarcodesByProductUseCase) Execute(ctx context.Context, productID string) ([]*domain.ProductBarcode, error) {
	if productID == "" {
		return nil, domain.ErrInvalidProductID
	}

	product, err := uc.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa produk: %w", err)
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	return uc.barcodeRepo.ListByProductID(ctx, productID)
}

// LookupProductByBarcodeUseCase menangani scan barcode kasir untuk menemukan produk & harga jual.
type LookupProductByBarcodeUseCase struct {
	barcodeRepo domain.BarcodeRepository
}

func NewLookupProductByBarcodeUseCase(barcodeRepo domain.BarcodeRepository) *LookupProductByBarcodeUseCase {
	return &LookupProductByBarcodeUseCase{barcodeRepo: barcodeRepo}
}

func (uc *LookupProductByBarcodeUseCase) Execute(ctx context.Context, code string) (*domain.Product, *domain.ProductBarcode, error) {
	if code == "" {
		return nil, nil, errors.New("kode barcode wajib diisi")
	}

	prod, barcode, err := uc.barcodeRepo.FindProductByBarcode(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("gagal lookup barcode: %w", err)
	}
	if prod == nil {
		return nil, nil, ErrProductNotFound
	}

	return prod, barcode, nil
}
