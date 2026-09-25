package interfaces

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory/application"
	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

// PriceOverrideHandler menangani request HTTP untuk manajemen harga promo dan perhitungan harga kasir POS.
type PriceOverrideHandler struct {
	createUC            *application.CreatePriceOverrideUseCase
	getEffectivePriceUC *application.GetEffectivePriceUseCase
	listUC              *application.ListPriceOverridesUseCase
	deactivateUC        *application.DeactivatePriceOverrideUseCase
	claimQuotaUC        *application.ClaimPromoQuotaUseCase
	logger              *slog.Logger
}

// NewPriceOverrideHandler membuat instance baru PriceOverrideHandler.
func NewPriceOverrideHandler(
	createUC *application.CreatePriceOverrideUseCase,
	getEffectivePriceUC *application.GetEffectivePriceUseCase,
	listUC *application.ListPriceOverridesUseCase,
	deactivateUC *application.DeactivatePriceOverrideUseCase,
	claimQuotaUC *application.ClaimPromoQuotaUseCase,
	logger *slog.Logger,
) *PriceOverrideHandler {
	return &PriceOverrideHandler{
		createUC:            createUC,
		getEffectivePriceUC: getEffectivePriceUC,
		listUC:              listUC,
		deactivateUC:        deactivateUC,
		claimQuotaUC:        claimQuotaUC,
		logger:              logger,
	}
}

// Create mendaftarkan promo harga khusus cabang baru.
// Endpoint: POST /api/v1/inventory/products/{id}/price-overrides
func (h *PriceOverrideHandler) Create(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID produk wajib diisi"})
		return
	}

	var req CreatePriceOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "body request tidak valid: " + err.Error()})
		return
	}

	cmd := application.CreatePriceOverrideCommand{
		ProductID:        productID,
		LocationID:       req.LocationID,
		PromotionalPrice: req.PromotionalPrice,
		MaxQuantity:      req.MaxQuantity,
		StartDate:        req.StartDate,
		EndDate:          req.EndDate,
		Reason:           req.Reason,
	}

	po, err := h.createUC.Execute(r.Context(), cmd)
	if err != nil {
		if errors.Is(err, domain.ErrOverlappingPromo) {
			writeJSON(w, http.StatusConflict, ErrorResponse{Error: err.Error()})
			return
		}
		if errors.Is(err, domain.ErrInvalidPromoPrice) || errors.Is(err, domain.ErrInvalidPromoDates) || errors.Is(err, domain.ErrInvalidPromoQuota) {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}
		if errors.Is(err, application.ErrProductNotFound) || errors.Is(err, application.ErrLocationNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}

		h.logger.Error("gagal membuat price override", "error", err, "product_id", productID)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, mapPriceOverrideToResponse(po))
}

// ListAll mengambil seluruh daftar promo harga khusus dengan filter (produk, cabang, status aktif).
// Endpoint: GET /api/v1/inventory/price-overrides
func (h *PriceOverrideHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	var productID *string
	if pid := r.URL.Query().Get("product_id"); pid != "" {
		productID = &pid
	}
	var locationID *string
	if lid := r.URL.Query().Get("location_id"); lid != "" {
		locationID = &lid
	}
	var isActive *bool
	if act := r.URL.Query().Get("is_active"); act != "" {
		val := act == "true" || act == "1"
		isActive = &val
	}

	query := application.ListAllPriceOverridesQuery{
		ProductID:  productID,
		LocationID: locationID,
		IsActive:   isActive,
	}

	details, err := h.listUC.ExecuteAll(r.Context(), query)
	if err != nil {
		h.logger.Error("gagal mengambil seluruh price overrides", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server"})
		return
	}

	res := make([]PriceOverrideResponse, 0, len(details))
	for _, d := range details {
		rItem := mapPriceOverrideToResponse(d.Override)
		rItem.ProductName = d.ProductName
		rItem.ProductSKU = d.ProductSKU
		rItem.ProductBrand = d.ProductBrand
		rItem.BasePrice = d.BasePrice
		rItem.LocationName = d.LocationName
		rItem.LocationCode = d.LocationCode
		res = append(res, rItem)
	}

	writeJSON(w, http.StatusOK, res)
}

// List mengambil riwayat dan daftar promo untuk suatu produk.
// Endpoint: GET /api/v1/inventory/products/{id}/price-overrides
func (h *PriceOverrideHandler) List(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID produk wajib diisi"})
		return
	}

	var locationID *string
	if loc := r.URL.Query().Get("location_id"); loc != "" {
		locationID = &loc
	}

	list, err := h.listUC.Execute(r.Context(), productID, locationID)
	if err != nil {
		h.logger.Error("gagal mengambil daftar price override", "error", err, "product_id", productID)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server"})
		return
	}

	res := make([]PriceOverrideResponse, 0, len(list))
	for _, po := range list {
		res = append(res, mapPriceOverrideToResponse(po))
	}

	writeJSON(w, http.StatusOK, res)
}

// GetEffectivePrice menghitung harga jual riil saat ini untuk kasir POS.
// Endpoint: GET /api/v1/inventory/price-overrides/effective-price?product_id=...&location_id=...
func (h *PriceOverrideHandler) GetEffectivePrice(w http.ResponseWriter, r *http.Request) {
	productID := r.URL.Query().Get("product_id")
	locationID := r.URL.Query().Get("location_id")

	if productID == "" || locationID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "query parameter 'product_id' dan 'location_id' wajib diisi"})
		return
	}

	result, err := h.getEffectivePriceUC.Execute(r.Context(), productID, locationID, time.Now().UTC())
	if err != nil {
		if errors.Is(err, application.ErrProductNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "produk tidak ditemukan"})
			return
		}

		h.logger.Error("gagal menghitung effective price", "error", err, "product_id", productID, "location_id", locationID)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server"})
		return
	}

	res := EffectivePriceResponse{
		ProductID:      result.ProductID,
		LocationID:     result.LocationID,
		BasePrice:      result.BasePrice,
		EffectivePrice: result.EffectivePrice,
		HasDiscount:    result.HasDiscount,
		DiscountAmount: result.DiscountAmount,
		RemainingQuota: result.RemainingQuota,
		PromoID:        result.PromoID,
		PromoReason:    result.PromoReason,
	}

	writeJSON(w, http.StatusOK, res)
}

// Deactivate mematikan promo harga cabang secara manual.
// Endpoint: PATCH /api/v1/inventory/price-overrides/{id}/deactivate
func (h *PriceOverrideHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID price override wajib diisi"})
		return
	}

	po, err := h.deactivateUC.Execute(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrPromoNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "promo harga tidak ditemukan"})
			return
		}

		h.logger.Error("gagal menonaktifkan price override", "error", err, "id", id)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, mapPriceOverrideToResponse(po))
}

// ClaimQuota memotong kuota promo saat terjadi transaksi kasir.
// Endpoint: POST /api/v1/inventory/price-overrides/{id}/claim
func (h *PriceOverrideHandler) ClaimQuota(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID price override wajib diisi"})
		return
	}

	var req ClaimPromoQuotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "body request tidak valid: " + err.Error()})
		return
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	if err := h.claimQuotaUC.Execute(r.Context(), id, req.Quantity); err != nil {
		if errors.Is(err, domain.ErrPromoNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "promo harga tidak ditemukan atau tidak aktif"})
			return
		}
		if errors.Is(err, domain.ErrPromoQuotaExhausted) {
			writeJSON(w, http.StatusConflict, ErrorResponse{Error: "kuota promo telah habis"})
			return
		}
		h.logger.Error("gagal klaim kuota promo", "error", err, "id", id)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "kuota promo berhasil diklaim", "claimed_quantity": req.Quantity})
}

// Helper untuk mapping domain.PriceOverride ke PriceOverrideResponse
func mapPriceOverrideToResponse(po *domain.PriceOverride) PriceOverrideResponse {
	return PriceOverrideResponse{
		ID:                po.ID,
		ProductID:         po.ProductID,
		LocationID:        po.LocationID,
		PromotionalPrice:  po.PromotionalPrice,
		MaxQuantity:       po.MaxQuantity,
		ClaimedQuantity:   po.ClaimedQuantity,
		RemainingQuantity: po.RemainingQuota(),
		StartDate:         po.StartDate,
		EndDate:           po.EndDate,
		Reason:            po.Reason,
		IsActive:          po.IsActive,
		CreatedAt:         po.CreatedAt,
		UpdatedAt:         po.UpdatedAt,
	}
}
