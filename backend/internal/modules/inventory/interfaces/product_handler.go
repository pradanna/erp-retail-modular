package interfaces

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/erp-retail/backend/internal/modules/inventory/application"
	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/internal/shared/auth"
)

// ProductHandler menangani semua HTTP request yang berkaitan dengan endpoint produk.
//
// MENGAPA HANDLER DI LAYER INTERFACES?
// Layer interfaces bertugas menerjemahkan "bahasa HTTP" ke "bahasa domain":
// - Baca JSON request → convert ke Command/Query
// - Jalankan use case
// - Convert hasil use case → JSON response
//
// Handler TIDAK BOLEH berisi business logic (validasi bisnis, rule kalkulasi, dsb).
// Handler hanya berisi: decode → call use case → encode response.
//
// Bandingkan dengan use case yang tidak tahu tentang HTTP sama sekali.
type ProductHandler struct {
	createProduct *application.CreateProductUseCase
	getProduct    *application.GetProductUseCase
	listProducts  *application.ListProductsUseCase
	updateProduct *application.UpdateProductUseCase
	setStatus     *application.SetProductStatusUseCase
	permService   auth.PermissionService
	logger        *slog.Logger
}

// SetPermissionService menyuntikkan PermissionService untuk evaluasi otorisasi granular PBAC.
func (h *ProductHandler) SetPermissionService(ps auth.PermissionService) {
	h.permService = ps
}

// NewProductHandler membuat handler baru dengan dependency yang sudah disiapkan.
func NewProductHandler(
	createProduct *application.CreateProductUseCase,
	getProduct *application.GetProductUseCase,
	listProducts *application.ListProductsUseCase,
	updateProduct *application.UpdateProductUseCase,
	setStatus *application.SetProductStatusUseCase,
	logger *slog.Logger,
) *ProductHandler {
	return &ProductHandler{
		createProduct: createProduct,
		getProduct:    getProduct,
		listProducts:  listProducts,
		updateProduct: updateProduct,
		setStatus:     setStatus,
		logger:        logger,
	}
}

// Create menangani POST /api/v1/inventory/products
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	// 1. Decode JSON request body ke DTO
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "format request tidak valid: " + err.Error()})
		return
	}

	// 2. Convert DTO → Application Command
	cmd := application.CreateProductCommand{
		SKU:           req.SKU,
		CategoryID:    req.CategoryID,
		Name:          req.Name,
		Brand:         req.Brand,
		Description:   req.Description,
		Unit:          req.Unit,
		PurchasePrice: req.PurchasePrice,
		SellingPrice:       req.SellingPrice,
		IsPPN:              req.IsPPN,
		FlagSerialTracking: req.FlagSerialTracking,
		WeightGram:         req.WeightGram,
		AtributVarian:      req.AtributVarian,
	}

	// 3. Jalankan use case
	productID, err := h.createProduct.Execute(r.Context(), cmd)
	if err != nil {
		h.logger.Error("gagal membuat produk", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	// 4. Kembalikan response sukses
	writeJSON(w, http.StatusCreated, CreateProductResponse{
		ID:      productID,
		Message: "Produk berhasil dibuat",
	})
}

// Update menangani PUT /api/v1/inventory/products/{id}
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "ID produk wajib diisi"})
		return
	}

	var req UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "format request tidak valid: " + err.Error()})
		return
	}

	cmd := application.UpdateProductCommand{
		ID:            id,
		Name:          req.Name,
		Brand:         req.Brand,
		Description:   req.Description,
		Unit:          req.Unit,
		PurchasePrice: req.PurchasePrice,
		SellingPrice:       req.SellingPrice,
		IsPPN:              req.IsPPN,
		FlagSerialTracking: req.FlagSerialTracking,
		WeightGram:         req.WeightGram,
		AtributVarian:      req.AtributVarian,
	}

	if err := h.updateProduct.Execute(r.Context(), cmd); err != nil {
		if errors.Is(err, application.ErrProductNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "produk tidak ditemukan"})
			return
		}
		h.logger.Error("gagal memperbarui produk", "id", id, "error", err.Error())
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Produk berhasil diperbarui",
	})
}

// SetStatus menangani PATCH /api/v1/inventory/products/{id}/status
func (h *ProductHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "ID produk wajib diisi"})
		return
	}

	var req SetProductStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "format request tidak valid: " + err.Error()})
		return
	}

	cmd := application.SetProductStatusCommand{
		ID:     id,
		Status: domain.ProductStatus(req.Status),
	}

	if err := h.setStatus.Execute(r.Context(), cmd); err != nil {
		if errors.Is(err, application.ErrProductNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "produk tidak ditemukan"})
			return
		}
		h.logger.Error("gagal memperbarui status produk", "id", id, "error", err.Error())
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Status produk berhasil diperbarui",
	})
}

// GetByID menangani GET /api/v1/inventory/products/{id}
func (h *ProductHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Go 1.22+ mendukung PathValue langsung dari r *http.Request
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "ID produk wajib diisi"})
		return
	}

	product, err := h.getProduct.Execute(r.Context(), id)
	if err != nil {
		if errors.Is(err, application.ErrProductNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "produk tidak ditemukan"})
			return
		}
		h.logger.Error("gagal mengambil detail produk", "id", id, "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal"})
		return
	}

	canViewCost := true
	if claims := auth.GetClaims(r); claims != nil && h.permService != nil {
		canViewCost = h.permService.HasPermission(claims.Role, "inventory.products.view_cost")
	}

	writeJSON(w, http.StatusOK, toProductResponse(product, canViewCost))
}

// List menangani GET /api/v1/inventory/products
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	query := application.ListProductsQuery{
		Page:  parseIntParam(q.Get("page"), 1),
		Limit: parseIntParam(q.Get("limit"), 20),
	}

	if s := q.Get("status"); s != "" {
		status := domain.ProductStatus(s)
		query.Status = &status
	}
	if cat := q.Get("category_id"); cat != "" {
		query.CategoryID = &cat
	}
	if search := q.Get("search"); search != "" {
		query.Search = &search
	}

	res, err := h.listProducts.Execute(r.Context(), query)
	if err != nil {
		h.logger.Error("gagal mengambil daftar produk", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal mengambil daftar produk"})
		return
	}

	canViewCost := true
	if claims := auth.GetClaims(r); claims != nil && h.permService != nil {
		canViewCost = h.permService.HasPermission(claims.Role, "inventory.products.view_cost")
	}

	var data []*ProductResponse
	for _, p := range res.Products {
		data = append(data, toProductResponse(p, canViewCost))
	}
	if data == nil {
		data = []*ProductResponse{}
	}

	writeJSON(w, http.StatusOK, ListProductResponse{
		Data:       data,
		Total:      res.Total,
		Page:       res.Page,
		Limit:      res.Limit,
		TotalPages: res.TotalPages,
	})
}

// ---- Helper functions ----

func toProductResponse(p *domain.Product, canViewCost bool) *ProductResponse {
	var purchasePrice *int64
	if canViewCost {
		val := p.PurchasePrice
		purchasePrice = &val
	}

	return &ProductResponse{
		ID:                 p.ID,
		SKU:                p.SKU,
		CategoryID:         p.CategoryID,
		Name:               p.Name,
		Brand:              p.Brand,
		Description:        p.Description,
		Unit:               p.Unit,
		PurchasePrice:      purchasePrice,
		SellingPrice:       p.SellingPrice,
		Status:             string(p.Status),
		IsPPN:              p.IsPPN,
		FlagSerialTracking: p.FlagSerialTracking,
		WeightGram:         p.WeightGram,
		AtributVarian:      p.AtributVarian,
		PrimaryImageURL:    p.PrimaryImageURL,
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}
