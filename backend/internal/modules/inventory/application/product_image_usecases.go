package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/pkg/uid"
)

const (
	MaxProductImageSize = 5 * 1024 * 1024 // 5 MB
	UploadBaseDir       = "./uploads/products"
)

// UploadProductImageUseCase menangani penyimpanan berkas fisik dan pencatatan foto produk.
type UploadProductImageUseCase struct {
	productRepo domain.ProductRepository
	imgRepo     domain.ProductImageRepository
}

func NewUploadProductImageUseCase(
	productRepo domain.ProductRepository,
	imgRepo domain.ProductImageRepository,
) *UploadProductImageUseCase {
	return &UploadProductImageUseCase{
		productRepo: productRepo,
		imgRepo:     imgRepo,
	}
}

func (uc *UploadProductImageUseCase) Execute(
	ctx context.Context,
	productID string,
	origFilename string,
	fileReader io.Reader,
	fileSize int64,
	isPrimary bool,
) (*domain.ProductImage, error) {
	if productID == "" {
		return nil, errors.New("product_id wajib diisi")
	}

	// 1. Validasi produk ada
	prod, err := uc.productRepo.FindByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("gagal verifikasi produk: %w", err)
	}
	if prod == nil {
		return nil, domain.ErrProductNotFound
	}

	// 2. Validasi ukuran berkas
	if fileSize > MaxProductImageSize {
		return nil, fmt.Errorf("ukuran gambar melebihi batas maksimal 5 MB")
	}

	// 3. Validasi ekstensi
	ext := strings.ToLower(filepath.Ext(origFilename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return nil, errors.New("format berkas tidak didukung (gunakan JPG, PNG, atau WebP)")
	}

	// 4. Pastikan direktori penyimpanan ada
	if err := os.MkdirAll(UploadBaseDir, 0755); err != nil {
		return nil, fmt.Errorf("gagal menyiapkan folder penyimpanan berkas: %w", err)
	}

	// 5. Simpan berkas fisik dengan UUIDv7
	imageID := uid.New()
	targetFileName := fmt.Sprintf("%s%s", imageID, ext)
	targetFilePath := filepath.Join(UploadBaseDir, targetFileName)

	outFile, err := os.Create(targetFilePath)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat berkas fisik: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, fileReader); err != nil {
		_ = os.Remove(targetFilePath)
		return nil, fmt.Errorf("gagal menyalin isi berkas gambar: %w", err)
	}

	// 6. Cek apakah ini foto pertama (jika ya, otomatis jadikan primary)
	count, err := uc.imgRepo.CountByProductID(ctx, productID)
	if err != nil {
		_ = os.Remove(targetFilePath)
		return nil, err
	}

	effectivePrimary := isPrimary || count == 0
	sortOrder := count + 1
	fileURL := fmt.Sprintf("/uploads/products/%s", targetFileName)

	img, err := domain.NewProductImage(imageID, productID, fileURL, effectivePrimary, sortOrder)
	if err != nil {
		_ = os.Remove(targetFilePath)
		return nil, err
	}

	if err := uc.imgRepo.Save(ctx, img); err != nil {
		_ = os.Remove(targetFilePath)
		return nil, err
	}

	// Jika diminta primary dan sebelumnya sudah ada foto lain, jadikan primary di DB
	if effectivePrimary && count > 0 {
		_ = uc.imgRepo.SetPrimary(ctx, productID, imageID)
	}

	return img, nil
}

// ListProductImagesUseCase mengambil daftar foto produk.
type ListProductImagesUseCase struct {
	imgRepo domain.ProductImageRepository
}

func NewListProductImagesUseCase(imgRepo domain.ProductImageRepository) *ListProductImagesUseCase {
	return &ListProductImagesUseCase{imgRepo: imgRepo}
}

func (uc *ListProductImagesUseCase) Execute(ctx context.Context, productID string) ([]*domain.ProductImage, error) {
	if productID == "" {
		return nil, errors.New("product_id wajib diisi")
	}
	return uc.imgRepo.FindByProductID(ctx, productID)
}

// DeleteProductImageUseCase menghapus foto dari database dan menghapus berkas fisik dari disk.
type DeleteProductImageUseCase struct {
	imgRepo domain.ProductImageRepository
}

func NewDeleteProductImageUseCase(imgRepo domain.ProductImageRepository) *DeleteProductImageUseCase {
	return &DeleteProductImageUseCase{imgRepo: imgRepo}
}

func (uc *DeleteProductImageUseCase) Execute(ctx context.Context, productID, imageID string) error {
	img, err := uc.imgRepo.FindByID(ctx, imageID)
	if err != nil {
		return err
	}
	if img == nil || img.ProductID != productID {
		return errors.New("foto tidak ditemukan pada produk ini")
	}

	wasPrimary := img.IsPrimary

	// 1. Hapus dari database
	if err := uc.imgRepo.Delete(ctx, imageID); err != nil {
		return err
	}

	// 2. Hapus berkas fisik dari disk jika ada
	filePath := filepath.Join(".", filepath.Clean(img.URL))
	_ = os.Remove(filePath)

	// 3. Jika foto yang dihapus adalah foto utama, jadikan foto lain yang tersisa sebagai utama
	if wasPrimary {
		remaining, err := uc.imgRepo.FindByProductID(ctx, productID)
		if err == nil && len(remaining) > 0 {
			_ = uc.imgRepo.SetPrimary(ctx, productID, remaining[0].ID)
		}
	}

	return nil
}

// SetPrimaryProductImageUseCase menetapkan foto tertentu sebagai foto utama.
type SetPrimaryProductImageUseCase struct {
	imgRepo domain.ProductImageRepository
}

func NewSetPrimaryProductImageUseCase(imgRepo domain.ProductImageRepository) *SetPrimaryProductImageUseCase {
	return &SetPrimaryProductImageUseCase{imgRepo: imgRepo}
}

func (uc *SetPrimaryProductImageUseCase) Execute(ctx context.Context, productID, imageID string) error {
	img, err := uc.imgRepo.FindByID(ctx, imageID)
	if err != nil {
		return err
	}
	if img == nil || img.ProductID != productID {
		return errors.New("foto tidak ditemukan pada produk ini")
	}

	return uc.imgRepo.SetPrimary(ctx, productID, imageID)
}
