package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

// ErrProductNotFound dikembalikan saat produk yang dicari tidak ditemukan.
var ErrProductNotFound = errors.New("produk tidak ditemukan")

// GetProductUseCase menangani pencarian produk tunggal berdasarkan ID.
type GetProductUseCase struct {
	repo domain.ProductRepository
}

// NewGetProductUseCase membuat instance baru GetProductUseCase.
func NewGetProductUseCase(repo domain.ProductRepository) *GetProductUseCase {
	return &GetProductUseCase{repo: repo}
}

// Execute menjalankan use case untuk mengambil produk berdasarkan ID.
func (uc *GetProductUseCase) Execute(ctx context.Context, id string) (*domain.Product, error) {
	if id == "" {
		return nil, errors.New("product ID tidak boleh kosong")
	}

	product, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil produk: %w", err)
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	return product, nil
}

// ListProductsQuery membawa parameter query dari caller ke use case.
type ListProductsQuery struct {
	Status     *domain.ProductStatus
	CategoryID *string
	Search     *string
	Page       int
	Limit      int
}

// ListProductsResult membungkus hasil pencarian daftar produk beserta pagination info.
type ListProductsResult struct {
	Products   []*domain.Product
	Total      int
	Page       int
	Limit      int
	TotalPages int
}

// ListProductsUseCase menangani query daftar produk dengan filter dan pagination.
type ListProductsUseCase struct {
	repo domain.ProductRepository
}

// NewListProductsUseCase membuat instance baru ListProductsUseCase.
func NewListProductsUseCase(repo domain.ProductRepository) *ListProductsUseCase {
	return &ListProductsUseCase{repo: repo}
}

// Execute mengeksekusi pencarian produk dan menghitung halaman pagination.
func (uc *ListProductsUseCase) Execute(ctx context.Context, query ListProductsQuery) (*ListProductsResult, error) {
	// Normalisasi pagination
	page := query.Page
	if page < 1 {
		page = 1
	}
	limit := query.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filter := domain.ProductFilter{
		Status:     query.Status,
		CategoryID: query.CategoryID,
		Search:     query.Search,
		Page:       page,
		Limit:      limit,
	}

	products, total, err := uc.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("gagal query produk: %w", err)
	}

	// Hitung total halaman (ceil total / limit)
	totalPages := 0
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return &ListProductsResult{
		Products:   products,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}
