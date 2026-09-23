package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

// UpdateProductCommand membawa data perubahan produk dari handler ke use case.
type UpdateProductCommand struct {
	ID            string
	Name          string
	Brand         string
	Description   string
	Unit          string
	PurchasePrice      int64
	SellingPrice       int64
	IsPPN              bool
	FlagSerialTracking bool
	WeightGram         int
	AtributVarian      map[string]any
}

// UpdateProductUseCase mengkoordinasikan proses pembaruan data produk.
type UpdateProductUseCase struct {
	repo domain.ProductRepository
}

// NewUpdateProductUseCase membuat instance baru UpdateProductUseCase.
func NewUpdateProductUseCase(repo domain.ProductRepository) *UpdateProductUseCase {
	return &UpdateProductUseCase{repo: repo}
}

// Execute menjalankan use case untuk memperbarui produk.
func (uc *UpdateProductUseCase) Execute(ctx context.Context, cmd UpdateProductCommand) error {
	if cmd.ID == "" {
		return errors.New("ID produk wajib diisi")
	}

	// 1. Ambil produk dari repository
	product, err := uc.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		return fmt.Errorf("gagal mengambil produk: %w", err)
	}
	if product == nil {
		return ErrProductNotFound
	}

	// 2. Terapkan perubahan melalui method Domain Entity
	// Aturan bisnis (invariants) divalidasi langsung di dalam entity domain
	if err := product.UpdateDetails(cmd.Name, cmd.Brand, cmd.Unit, cmd.Description, cmd.WeightGram); err != nil {
		return err
	}

	if err := product.UpdatePrice(cmd.PurchasePrice, cmd.SellingPrice); err != nil {
		return err
	}

	product.SetPPN(cmd.IsPPN)
	product.FlagSerialTracking = cmd.FlagSerialTracking
	product.SetAtributVarian(cmd.AtributVarian)

	// 3. Simpan perubahan ke database
	if err := uc.repo.Update(ctx, product); err != nil {
		return fmt.Errorf("gagal memperbarui produk di database: %w", err)
	}

	return nil
}

// SetProductStatusCommand membawa data perubahan status produk.
type SetProductStatusCommand struct {
	ID     string
	Status domain.ProductStatus
}

// SetProductStatusUseCase menangani pengubahan status produk (aktif/nonaktif/discontinued).
type SetProductStatusUseCase struct {
	repo domain.ProductRepository
}

// NewSetProductStatusUseCase membuat instance baru SetProductStatusUseCase.
func NewSetProductStatusUseCase(repo domain.ProductRepository) *SetProductStatusUseCase {
	return &SetProductStatusUseCase{repo: repo}
}

// Execute menjalankan use case pengubahan status produk.
func (uc *SetProductStatusUseCase) Execute(ctx context.Context, cmd SetProductStatusCommand) error {
	if cmd.ID == "" {
		return errors.New("ID produk wajib diisi")
	}

	product, err := uc.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		return fmt.Errorf("gagal mengambil produk: %w", err)
	}
	if product == nil {
		return ErrProductNotFound
	}

	// Validasi dan ubah status di Domain Entity
	if err := product.ChangeStatus(cmd.Status); err != nil {
		return err
	}

	if err := uc.repo.Update(ctx, product); err != nil {
		return fmt.Errorf("gagal memperbarui status produk di database: %w", err)
	}

	return nil
}
