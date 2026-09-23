package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/pkg/uid"
)

var (
	ErrWarrantyPolicyNotFound = errors.New("kebijakan garansi tidak ditemukan")
)

// CreateWarrantyPolicyCommand parameter input untuk pembuatan master garansi baru.
type CreateWarrantyPolicyCommand struct {
	Name              string
	Type              domain.WarrantyType
	DurationMonths    int
	DurationDays      int
	Coverage          string
	ClaimInstructions string
}

// CreateWarrantyPolicyUseCase use case untuk membuat master kebijakan garansi.
type CreateWarrantyPolicyUseCase struct {
	repo domain.WarrantyPolicyRepository
}

// NewCreateWarrantyPolicyUseCase membuat instance baru CreateWarrantyPolicyUseCase.
func NewCreateWarrantyPolicyUseCase(repo domain.WarrantyPolicyRepository) *CreateWarrantyPolicyUseCase {
	return &CreateWarrantyPolicyUseCase{repo: repo}
}

// Execute mengeksekusi pembuatan master kebijakan garansi baru.
func (uc *CreateWarrantyPolicyUseCase) Execute(ctx context.Context, cmd CreateWarrantyPolicyCommand) (*domain.WarrantyPolicy, error) {
	id := uid.New()
	policy, err := domain.NewWarrantyPolicy(
		id,
		cmd.Name,
		cmd.Type,
		cmd.DurationMonths,
		cmd.DurationDays,
		cmd.Coverage,
		cmd.ClaimInstructions,
	)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, policy); err != nil {
		return nil, fmt.Errorf("gagal menyimpan kebijakan garansi: %w", err)
	}
	return policy, nil
}

// ListWarrantyPoliciesQuery parameter filter pencarian master kebijakan garansi.
type ListWarrantyPoliciesQuery struct {
	Type         *domain.WarrantyType
	IsActiveOnly bool
}

// ListWarrantyPoliciesUseCase use case untuk melihat daftar master kebijakan garansi.
type ListWarrantyPoliciesUseCase struct {
	repo domain.WarrantyPolicyRepository
}

// NewListWarrantyPoliciesUseCase membuat instance baru ListWarrantyPoliciesUseCase.
func NewListWarrantyPoliciesUseCase(repo domain.WarrantyPolicyRepository) *ListWarrantyPoliciesUseCase {
	return &ListWarrantyPoliciesUseCase{repo: repo}
}

// Execute mengambil daftar master kebijakan garansi sesuai filter.
func (uc *ListWarrantyPoliciesUseCase) Execute(ctx context.Context, query ListWarrantyPoliciesQuery) ([]*domain.WarrantyPolicy, error) {
	return uc.repo.List(ctx, query.Type, query.IsActiveOnly)
}

// AssignProductWarrantyCommand parameter input untuk menugaskan garansi ke produk.
type AssignProductWarrantyCommand struct {
	ProductID        string
	WarrantyPolicyID string
}

// AssignProductWarrantyUseCase use case untuk menetapkan garansi ke produk.
// Invariant Kunci: Maksimal 1 garansi aktif per 'Type' ('toko' atau 'pabrik').
// Garansi lama dengan tipe yang sama akan dinonaktifkan secara atomik.
type AssignProductWarrantyUseCase struct {
	productRepo         domain.ProductRepository
	policyRepo          domain.WarrantyPolicyRepository
	productWarrantyRepo domain.ProductWarrantyRepository
}

// NewAssignProductWarrantyUseCase membuat instance baru AssignProductWarrantyUseCase.
func NewAssignProductWarrantyUseCase(
	productRepo domain.ProductRepository,
	policyRepo domain.WarrantyPolicyRepository,
	productWarrantyRepo domain.ProductWarrantyRepository,
) *AssignProductWarrantyUseCase {
	return &AssignProductWarrantyUseCase{
		productRepo:         productRepo,
		policyRepo:          policyRepo,
		productWarrantyRepo: productWarrantyRepo,
	}
}

// Execute mengeksekusi validasi bisnis dan penugasan garansi ke produk.
func (uc *AssignProductWarrantyUseCase) Execute(ctx context.Context, cmd AssignProductWarrantyCommand) (*domain.ProductWarranty, error) {
	// 1. Validasi produk ada
	product, err := uc.productRepo.FindByID(ctx, cmd.ProductID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	// 2. Validasi master kebijakan garansi ada dan aktif
	policy, err := uc.policyRepo.FindByID(ctx, cmd.WarrantyPolicyID)
	if err != nil {
		if errors.Is(err, domain.ErrPolicyNotFound) {
			return nil, ErrWarrantyPolicyNotFound
		}
		return nil, err
	}
	if !policy.IsActive {
		return nil, errors.New("kebijakan garansi yang dipilih sedang tidak aktif")
	}

	// 3. Buat entity ProductWarranty baru
	id := uid.New()
	pw, err := domain.NewProductWarranty(id, product.ID, policy.ID, policy.Type)
	if err != nil {
		return nil, err
	}

	// 4. Simpan secara atomik (repo otomatis menonaktifkan garansi aktif sebelumnya dengan tipe sama)
	if err := uc.productWarrantyRepo.AssignWarranty(ctx, pw); err != nil {
		return nil, fmt.Errorf("gagal menugaskan garansi ke produk: %w", err)
	}

	pw.Policy = policy
	return pw, nil
}

// GetProductActiveWarrantiesUseCase use case untuk mengambil garansi aktif dari sebuah produk.
type GetProductActiveWarrantiesUseCase struct {
	productRepo         domain.ProductRepository
	productWarrantyRepo domain.ProductWarrantyRepository
}

// NewGetProductActiveWarrantiesUseCase membuat instance baru GetProductActiveWarrantiesUseCase.
func NewGetProductActiveWarrantiesUseCase(
	productRepo domain.ProductRepository,
	productWarrantyRepo domain.ProductWarrantyRepository,
) *GetProductActiveWarrantiesUseCase {
	return &GetProductActiveWarrantiesUseCase{
		productRepo:         productRepo,
		productWarrantyRepo: productWarrantyRepo,
	}
}

// Execute mengambil daftar garansi aktif untuk suatu produk.
func (uc *GetProductActiveWarrantiesUseCase) Execute(ctx context.Context, productID string) ([]*domain.ProductWarranty, error) {
	product, err := uc.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	return uc.productWarrantyRepo.FindActiveByProduct(ctx, productID)
}

// DeactivateProductWarrantyUseCase use case untuk menonaktifkan garansi suatu produk secara manual.
type DeactivateProductWarrantyUseCase struct {
	productWarrantyRepo domain.ProductWarrantyRepository
}

// NewDeactivateProductWarrantyUseCase membuat instance baru DeactivateProductWarrantyUseCase.
func NewDeactivateProductWarrantyUseCase(repo domain.ProductWarrantyRepository) *DeactivateProductWarrantyUseCase {
	return &DeactivateProductWarrantyUseCase{productWarrantyRepo: repo}
}

// Execute menonaktifkan penugasan garansi produk.
func (uc *DeactivateProductWarrantyUseCase) Execute(ctx context.Context, productWarrantyID string) (*domain.ProductWarranty, error) {
	pw, err := uc.productWarrantyRepo.FindByID(ctx, productWarrantyID)
	if err != nil {
		return nil, err
	}

	pw.Deactivate()
	if err := uc.productWarrantyRepo.Update(ctx, pw); err != nil {
		return nil, fmt.Errorf("gagal menonaktifkan garansi produk: %w", err)
	}
	return pw, nil
}
