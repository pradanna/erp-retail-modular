package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/pkg/uid"
)

// CreatePriceOverrideCommand adalah parameter input untuk mendaftarkan promo harga cabang baru.
type CreatePriceOverrideCommand struct {
	ProductID        string
	LocationID       string
	PromotionalPrice int64
	MaxQuantity      *int
	StartDate        time.Time
	EndDate          time.Time
	Reason           string
}

// CreatePriceOverrideUseCase menangani pendaftaran harga promo cabang baru.
type CreatePriceOverrideUseCase struct {
	repo         domain.PriceOverrideRepository
	productRepo  domain.ProductRepository
	locationRepo domain.LocationRepository
}

// NewCreatePriceOverrideUseCase membuat instance baru CreatePriceOverrideUseCase.
func NewCreatePriceOverrideUseCase(
	repo domain.PriceOverrideRepository,
	productRepo domain.ProductRepository,
	locationRepo domain.LocationRepository,
) *CreatePriceOverrideUseCase {
	return &CreatePriceOverrideUseCase{
		repo:         repo,
		productRepo:  productRepo,
		locationRepo: locationRepo,
	}
}

// Execute mengeksekusi validasi bisnis dan menyimpan promo harga baru.
func (uc *CreatePriceOverrideUseCase) Execute(ctx context.Context, cmd CreatePriceOverrideCommand) (*domain.PriceOverride, error) {
	// 1. Validasi produk ada dan aktif
	product, err := uc.productRepo.FindByID(ctx, cmd.ProductID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}
	if product.Status != domain.ProductStatusActive {
		return nil, errors.New("tidak dapat membuat promo untuk produk yang tidak aktif")
	}

	// 2. Validasi lokasi cabang ada dan aktif
	location, err := uc.locationRepo.FindByID(ctx, cmd.LocationID)
	if err != nil {
		return nil, err
	}
	if location == nil {
		return nil, ErrLocationNotFound
	}
	if !location.IsActive {
		return nil, errors.New("tidak dapat membuat promo untuk cabang/lokasi yang tidak aktif")
	}

	// 3. Validasi invariant ketat: Tidak boleh ada promo aktif lain yang rentang tanggalnya bertabrakan
	hasOverlap, err := uc.repo.HasOverlappingPromo(ctx, cmd.ProductID, cmd.LocationID, cmd.StartDate, cmd.EndDate, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal verifikasi tabrakan promo: %w", err)
	}
	if hasOverlap {
		return nil, domain.ErrOverlappingPromo
	}

	// 4. Buat entitas domain
	overrideID := uid.New()
	po, err := domain.NewPriceOverride(
		overrideID, cmd.ProductID, cmd.LocationID, cmd.PromotionalPrice,
		cmd.StartDate, cmd.EndDate, cmd.Reason, cmd.MaxQuantity,
	)
	if err != nil {
		return nil, err
	}

	// 5. Simpan ke database
	if err := uc.repo.Save(ctx, po); err != nil {
		return nil, err
	}

	return po, nil
}

// EffectivePriceResult adalah hasil kalkulasi harga riil saat kasir melakukan scan/transaksi.
type EffectivePriceResult struct {
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

// GetEffectivePriceUseCase menangani penentuan harga jual riil suatu barang di cabang tertentu saat ini.
type GetEffectivePriceUseCase struct {
	repo         domain.PriceOverrideRepository
	productRepo  domain.ProductRepository
	locationRepo domain.LocationRepository
}

// NewGetEffectivePriceUseCase membuat instance baru GetEffectivePriceUseCase.
func NewGetEffectivePriceUseCase(
	repo domain.PriceOverrideRepository,
	productRepo domain.ProductRepository,
	locationRepo domain.LocationRepository,
) *GetEffectivePriceUseCase {
	return &GetEffectivePriceUseCase{
		repo:         repo,
		productRepo:  productRepo,
		locationRepo: locationRepo,
	}
}

// Execute menghitung harga jual final di cabang (mengambil promo aktif jika ada dan kuota masih ada, atau fallback ke harga dasar produk).
func (uc *GetEffectivePriceUseCase) Execute(ctx context.Context, productID, locationID string, at time.Time) (*EffectivePriceResult, error) {
	product, err := uc.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	res := &EffectivePriceResult{
		ProductID:      productID,
		LocationID:     locationID,
		BasePrice:      product.SellingPrice,
		EffectivePrice: product.SellingPrice,
		HasDiscount:    false,
		DiscountAmount: 0,
	}

	// Cari apakah ada promo aktif saat ini di cabang tersebut (kuota belum habis)
	promo, err := uc.repo.FindActive(ctx, productID, locationID, at)
	if err != nil && !errors.Is(err, domain.ErrPromoNotFound) {
		return nil, fmt.Errorf("gagal memeriksa promo cabang: %w", err)
	}

	if promo != nil {
		res.EffectivePrice = promo.PromotionalPrice
		res.HasDiscount = true
		res.DiscountAmount = product.SellingPrice - promo.PromotionalPrice
		if res.DiscountAmount < 0 {
			res.DiscountAmount = 0
		}
		res.RemainingQuota = promo.RemainingQuota()
		res.PromoID = &promo.ID
		res.PromoReason = &promo.Reason
	}

	return res, nil
}

// ListPriceOverridesUseCase mengambil riwayat dan daftar promo untuk suatu produk.
type ListPriceOverridesUseCase struct {
	repo domain.PriceOverrideRepository
}

// NewListPriceOverridesUseCase membuat instance baru ListPriceOverridesUseCase.
func NewListPriceOverridesUseCase(repo domain.PriceOverrideRepository) *ListPriceOverridesUseCase {
	return &ListPriceOverridesUseCase{repo: repo}
}

// Execute mengambil daftar price overrides milik produk tertentu.
func (uc *ListPriceOverridesUseCase) Execute(ctx context.Context, productID string, locationID *string) ([]*domain.PriceOverride, error) {
	if productID == "" {
		return nil, domain.ErrInvalidProductID
	}
	return uc.repo.ListByProduct(ctx, productID, locationID)
}

// DeactivatePriceOverrideUseCase mematikan promo secara manual sebelum tanggal berakhirnya.
type DeactivatePriceOverrideUseCase struct {
	repo domain.PriceOverrideRepository
}

// NewDeactivatePriceOverrideUseCase membuat instance baru DeactivatePriceOverrideUseCase.
func NewDeactivatePriceOverrideUseCase(repo domain.PriceOverrideRepository) *DeactivatePriceOverrideUseCase {
	return &DeactivatePriceOverrideUseCase{repo: repo}
}

// Execute menonaktifkan promo berdasarkan ID.
func (uc *DeactivatePriceOverrideUseCase) Execute(ctx context.Context, id string) (*domain.PriceOverride, error) {
	if id == "" {
		return nil, domain.ErrInvalidOverrideID
	}

	po, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	po.Deactivate()

	if err := uc.repo.Update(ctx, po); err != nil {
		return nil, err
	}

	return po, nil
}

// ClaimPromoQuotaUseCase menangani pemotongan kuota promo saat terjadi transaksi kasir.
type ClaimPromoQuotaUseCase struct {
	repo domain.PriceOverrideRepository
}

// NewClaimPromoQuotaUseCase membuat instance baru ClaimPromoQuotaUseCase.
func NewClaimPromoQuotaUseCase(repo domain.PriceOverrideRepository) *ClaimPromoQuotaUseCase {
	return &ClaimPromoQuotaUseCase{repo: repo}
}

// Execute memotong sejumlah unit dari kuota promo yang sedang berjalan.
func (uc *ClaimPromoQuotaUseCase) Execute(ctx context.Context, promoID string, qty int) error {
	if promoID == "" {
		return domain.ErrInvalidOverrideID
	}
	if qty <= 0 {
		return errors.New("jumlah klaim kuota harus lebih dari 0")
	}
	return uc.repo.IncrementClaimedQuantity(ctx, promoID, qty)
}
