package interfaces

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/erp-retail/backend/internal/modules/inventory/application"
	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/pkg/uid"
)

// CategoryHandler menangani seluruh endpoint HTTP untuk master kategori.
type CategoryHandler struct {
	createCategory *application.CreateCategoryUseCase
	listCategories *application.ListCategoriesUseCase
	updateCategory *application.UpdateCategoryUseCase
	deleteCategory *application.DeleteCategoryUseCase
	logger         *slog.Logger
}

// NewCategoryHandler membuat instance CategoryHandler dengan dependensi use case.
func NewCategoryHandler(
	createCategory *application.CreateCategoryUseCase,
	listCategories *application.ListCategoriesUseCase,
	updateCategory *application.UpdateCategoryUseCase,
	deleteCategory *application.DeleteCategoryUseCase,
	logger *slog.Logger,
) *CategoryHandler {
	return &CategoryHandler{
		createCategory: createCategory,
		listCategories: listCategories,
		updateCategory: updateCategory,
		deleteCategory: deleteCategory,
		logger:         logger,
	}
}

// Create menangani POST /api/v1/inventory/categories
func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "format JSON request tidak valid: " + err.Error()})
		return
	}

	cmd := application.CreateCategoryCommand{
		Name:     req.Name,
		ParentID: req.ParentID,
		ImageURL: req.ImageURL,
	}

	id, err := h.createCategory.Execute(r.Context(), cmd)
	if err != nil {
		h.logger.Error("gagal membuat kategori", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"id":      id,
		"message": "Kategori berhasil dibuat",
	})
}

// List menangani GET /api/v1/inventory/categories
func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.listCategories.Execute(r.Context())
	if err != nil {
		h.logger.Error("gagal mengambil daftar kategori", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal mengambil data kategori"})
		return
	}

	var data []*CategoryResponse
	for _, c := range categories {
		data = append(data, toCategoryResponse(c))
	}
	if data == nil {
		data = []*CategoryResponse{}
	}

	writeJSON(w, http.StatusOK, data)
}

// Update menangani PUT /api/v1/inventory/categories/{id}
func (h *CategoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "ID kategori wajib diisi"})
		return
	}

	var req UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "format JSON request tidak valid: " + err.Error()})
		return
	}

	cmd := application.UpdateCategoryCommand{
		ID:       id,
		Name:     req.Name,
		ParentID: req.ParentID,
		ImageURL: req.ImageURL,
	}

	if err := h.updateCategory.Execute(r.Context(), cmd); err != nil {
		if errors.Is(err, application.ErrCategoryNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "kategori tidak ditemukan"})
			return
		}
		h.logger.Error("gagal memperbarui kategori", "id", id, "error", err.Error())
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Kategori berhasil diperbarui",
	})
}

// Delete menangani DELETE /api/v1/inventory/categories/{id}
func (h *CategoryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "ID kategori wajib diisi"})
		return
	}

	if err := h.deleteCategory.Execute(r.Context(), id); err != nil {
		if errors.Is(err, application.ErrCategoryNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "kategori tidak ditemukan"})
			return
		}
		h.logger.Error("gagal menghapus kategori", "id", id, "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal menghapus kategori"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Kategori berhasil dihapus",
	})
}

// UploadImage menangani upload berkas gambar kategori (multipart/form-data).
func (h *CategoryHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	// Batasi ukuran request body multipart hingga 10MB
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "gagal memproses data formulir upload: " + err.Error()})
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		file, header, err = r.FormFile("file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "berkas gambar tidak ditemukan (gunakan form field 'image' atau 'file')"})
			return
		}
	}
	defer file.Close()

	// Batas ukuran maksimal gambar kategori: 5 MB
	if header.Size > 5*1024*1024 {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "ukuran berkas gambar melebihi batas maksimal 5 MB"})
		return
	}

	// Validasi ekstensi yang diizinkan
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "format berkas tidak didukung (gunakan JPG, PNG, atau WebP)"})
		return
	}

	uploadDir := "./uploads/categories"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		h.logger.Error("gagal membuat folder uploads/categories", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal menyiapkan folder penyimpanan server"})
		return
	}

	imageID := uid.New()
	targetFileName := fmt.Sprintf("%s%s", imageID, ext)
	targetFilePath := filepath.Join(uploadDir, targetFileName)

	outFile, err := os.Create(targetFilePath)
	if err != nil {
		h.logger.Error("gagal membuat berkas fisik kategori", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal menyimpan berkas di server"})
		return
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, file); err != nil {
		_ = os.Remove(targetFilePath)
		h.logger.Error("gagal menyalin isi berkas gambar kategori", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal menulis data gambar"})
		return
	}

	fileURL := fmt.Sprintf("/uploads/categories/%s", targetFileName)
	h.logger.Info("gambar kategori berhasil diunggah", "filename", targetFileName, "url", fileURL)

	writeJSON(w, http.StatusCreated, map[string]string{
		"url": fileURL,
	})
}

func toCategoryResponse(c *domain.Category) *CategoryResponse {
	return &CategoryResponse{
		ID:        c.ID,
		Name:      c.Name,
		ParentID:  c.ParentID,
		ImageURL:  c.ImageURL,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
