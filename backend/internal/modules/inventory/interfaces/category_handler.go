package interfaces

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/erp-retail/backend/internal/modules/inventory/application"
	"github.com/erp-retail/backend/internal/modules/inventory/domain"
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
