package application

import (
	"context"
	"fmt"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/pkg/uid"
)

// CreateProductCommand adalah input data untuk use case CreateProduct.
// Pola "Command" (dari CQRS) memisahkan intent dari eksekusi:
// "Saya ingin membuat produk dengan data ini" — bukan prosedur langsung.
type CreateProductCommand struct {
	SKU           string
	CategoryID    string
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

// CreateProductUseCase adalah use case untuk membuat produk baru.
//
// MENGAPA USE CASE DI LAYER APPLICATION?
// Layer application bertugas mengorkestrasi — ia tidak berisi aturan bisnis (itu di domain),
// dan tidak berisi detail teknis (itu di infrastructure).
// Application hanya menjawab: "Langkah apa yang harus dijalankan untuk melakukan X?"
//
// Perhatikan: CreateProductUseCase hanya bergantung pada INTERFACE domain.ProductRepository,
// bukan pada implementasi konkret SQL. Ini adalah Dependency Injection via interface.
type CreateProductUseCase struct {
	productRepo domain.ProductRepository
}

// NewCreateProductUseCase adalah constructor use case.
// Constructor ini menerima interface (bukan concrete type) — sehingga bisa di-inject
// dengan mock saat testing, atau implementasi real SQL saat production.
func NewCreateProductUseCase(repo domain.ProductRepository) *CreateProductUseCase {
	return &CreateProductUseCase{productRepo: repo}
}

// Execute menjalankan use case: validasi → buat entity → simpan ke repository.
//
// Alurnya:
// 1. Cek apakah SKU sudah dipakai (business rule: SKU harus unik)
// 2. Buat Product entity baru via domain constructor (validasi domain invariant)
// 3. Set field tambahan yang tidak masuk ke constructor
// 4. Simpan ke repository via interface (domain tidak tahu SQL, use case juga tidak)
// 5. Kembalikan ID produk yang baru dibuat
func (uc *CreateProductUseCase) Execute(ctx context.Context, cmd CreateProductCommand) (string, error) {
	// 1. Periksa duplikat SKU — business rule: satu SKU per toko
	existing, err := uc.productRepo.FindBySKU(ctx, cmd.SKU)
	if err != nil {
		return "", fmt.Errorf("gagal memeriksa SKU: %w", err)
	}
	if existing != nil {
		return "", fmt.Errorf("SKU '%s' sudah digunakan oleh produk lain", cmd.SKU)
	}

	// 2. Generate UUIDv7 untuk ID produk baru
	productID := uid.New()

	// 3. Buat entity Product via domain constructor (validasi di dalam constructor)
	product, err := domain.NewProduct(
		productID,
		cmd.SKU,
		cmd.CategoryID,
		cmd.Name,
		cmd.Brand,
		cmd.Unit,
		cmd.PurchasePrice,
		cmd.SellingPrice,
	)
	if err != nil {
		// Error ini berasal dari domain validation — bukan technical error
		return "", fmt.Errorf("data produk tidak valid: %w", err)
	}

	// 4. Set field opsional tambahan
	product.Description = cmd.Description
	product.IsPPN = cmd.IsPPN
	product.FlagSerialTracking = cmd.FlagSerialTracking
	product.WeightGram = cmd.WeightGram
	if cmd.AtributVarian != nil {
		product.AtributVarian = cmd.AtributVarian
	}

	// 5. Simpan ke database via repository interface
	if err := uc.productRepo.Save(ctx, product); err != nil {
		return "", fmt.Errorf("gagal menyimpan produk: %w", err)
	}

	return productID, nil
}
