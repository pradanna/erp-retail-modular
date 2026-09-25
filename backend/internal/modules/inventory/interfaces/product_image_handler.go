package interfaces

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/erp-retail/backend/internal/modules/inventory/application"
	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

type ProductImageHandler struct {
	uploadUC     *application.UploadProductImageUseCase
	listUC       *application.ListProductImagesUseCase
	deleteUC     *application.DeleteProductImageUseCase
	setPrimaryUC *application.SetPrimaryProductImageUseCase
	logger       *slog.Logger
}

func NewProductImageHandler(
	uploadUC *application.UploadProductImageUseCase,
	listUC *application.ListProductImagesUseCase,
	deleteUC *application.DeleteProductImageUseCase,
	setPrimaryUC *application.SetPrimaryProductImageUseCase,
	logger *slog.Logger,
) *ProductImageHandler {
	return &ProductImageHandler{
		uploadUC:     uploadUC,
		listUC:       listUC,
		deleteUC:     deleteUC,
		setPrimaryUC: setPrimaryUC,
		logger:       logger,
	}
}

// Upload menangani unggahan berkas foto produk (multipart/form-data).
func (h *ProductImageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "id produk wajib diisi"})
		return
	}

	// Batasi ukuran request body multipart hingga 10MB
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "gagal memproses data formulir upload: " + err.Error()})
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		// Coba fallback dengan field 'file'
		file, header, err = r.FormFile("file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "berkas gambar tidak ditemukan (gunakan key 'image' atau 'file')"})
			return
		}
	}
	defer file.Close()

	isPrimary := r.FormValue("is_primary") == "true"

	img, err := h.uploadUC.Execute(
		r.Context(),
		productID,
		header.Filename,
		file,
		header.Size,
		isPrimary,
	)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "produk tidak ditemukan"})
			return
		}
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	h.logger.Info("foto produk berhasil diunggah", "product_id", productID, "image_id", img.ID, "url", img.URL)
	writeJSON(w, http.StatusCreated, ProductImageResponse{
		ID:        img.ID,
		ProductID: img.ProductID,
		URL:       img.URL,
		IsPrimary: img.IsPrimary,
		SortOrder: img.SortOrder,
		CreatedAt: img.CreatedAt,
	})
}

// ListByProduct menyajikan seluruh daftar foto milik produk.
func (h *ProductImageHandler) ListByProduct(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "id produk wajib diisi"})
		return
	}

	images, err := h.listUC.Execute(r.Context(), productID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal mengambil daftar foto produk: " + err.Error()})
		return
	}

	res := make([]ProductImageResponse, 0, len(images))
	for _, img := range images {
		res = append(res, ProductImageResponse{
			ID:        img.ID,
			ProductID: img.ProductID,
			URL:       img.URL,
			IsPrimary: img.IsPrimary,
			SortOrder: img.SortOrder,
			CreatedAt: img.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, res)
}

// Delete menghapus foto produk dan berkas fisiknya.
func (h *ProductImageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	imageID := r.PathValue("image_id")

	if productID == "" || imageID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "id produk dan id foto wajib diisi"})
		return
	}

	if err := h.deleteUC.Execute(r.Context(), productID, imageID); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	h.logger.Info("foto produk berhasil dihapus", "product_id", productID, "image_id", imageID)
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "foto produk berhasil dihapus",
	})
}

// SetPrimary menetapkan foto tertentu sebagai foto utama produk.
func (h *ProductImageHandler) SetPrimary(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	imageID := r.PathValue("image_id")

	if productID == "" || imageID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "id produk dan id foto wajib diisi"})
		return
	}

	if err := h.setPrimaryUC.Execute(r.Context(), productID, imageID); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	h.logger.Info("foto utama produk berhasil ditetapkan", "product_id", productID, "image_id", imageID)
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "foto utama berhasil diperbarui",
	})
}
