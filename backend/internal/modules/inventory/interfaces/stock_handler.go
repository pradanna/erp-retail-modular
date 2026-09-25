package interfaces

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/erp-retail/backend/internal/modules/inventory/application"
	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/internal/shared/auth"
)

// StockHandler menangani request HTTP untuk manajemen stok barang per cabang/gudang.
type StockHandler struct {
	adjustStockUC     *application.AdjustStockUseCase
	listAdjustmentsUC *application.ListStockAdjustmentsUseCase
	getStockUC        *application.GetStockUseCase
	listStockUC       *application.ListStockByLocationUseCase
	listAlertsUC      *application.ListLowStockAlertsUseCase
	updateMinStockUC  *application.UpdateMinStockUseCase
	logger            *slog.Logger
}

func NewStockHandler(
	adjustStockUC *application.AdjustStockUseCase,
	listAdjustmentsUC *application.ListStockAdjustmentsUseCase,
	getStockUC *application.GetStockUseCase,
	listStockUC *application.ListStockByLocationUseCase,
	listAlertsUC *application.ListLowStockAlertsUseCase,
	updateMinStockUC *application.UpdateMinStockUseCase,
	logger *slog.Logger,
) *StockHandler {
	return &StockHandler{
		adjustStockUC:     adjustStockUC,
		listAdjustmentsUC: listAdjustmentsUC,
		getStockUC:        getStockUC,
		listStockUC:       listStockUC,
		listAlertsUC:      listAlertsUC,
		updateMinStockUC:  updateMinStockUC,
		logger:            logger,
	}
}

// Adjust menangani penyesuaian stok manual (Stock Opname).
// Endpoint: POST /api/v1/inventory/stocks/adjust
func (h *StockHandler) Adjust(w http.ResponseWriter, r *http.Request) {
	var req AdjustStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "body request tidak valid: " + err.Error()})
		return
	}

	adjustedBy := "system"
	adjustedByName := "Admin Toko"
	if claims := auth.GetClaims(r); claims != nil {
		adjustedBy = claims.UserID
		if claims.Subject != "" {
			adjustedByName = claims.Subject
		} else if claims.Role != "" {
			adjustedByName = "Admin (" + claims.Role + ")"
		}
	}

	cmd := application.AdjustStockCommand{
		ProductID:      req.ProductID,
		LocationID:     req.LocationID,
		NewQuantity:    req.NewQuantity,
		Reason:         req.Reason,
		AdjustedBy:     adjustedBy,
		AdjustedByName: adjustedByName,
	}

	item, err := h.adjustStockUC.Execute(r.Context(), cmd)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrProductNotFound),
			errors.Is(err, application.ErrLocationNotFound):
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: err.Error()})
		case errors.Is(err, application.ErrLocationInactive),
			errors.Is(err, domain.ErrInvalidProductID),
			errors.Is(err, domain.ErrInvalidLocationID),
			errors.Is(err, domain.ErrNegativeQuantity),
			errors.Is(err, domain.ErrReservedExceedsStock):
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		default:
			h.logger.Error("gagal adjust stock", "error", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server"})
		}
		return
	}

	writeJSON(w, http.StatusOK, toStockResponse(item))
}

// UpdateMinStock menangani pembaruan batas minimum stok.
// Endpoint: PUT /api/v1/inventory/stocks/min-stock
func (h *StockHandler) UpdateMinStock(w http.ResponseWriter, r *http.Request) {
	var req UpdateMinStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "body request tidak valid: " + err.Error()})
		return
	}

	cmd := application.UpdateMinStockCommand{
		ProductID:  req.ProductID,
		LocationID: req.LocationID,
		MinStock:   req.MinStock,
	}

	item, err := h.updateMinStockUC.Execute(r.Context(), cmd)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrProductNotFound),
			errors.Is(err, application.ErrLocationNotFound):
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: err.Error()})
		case errors.Is(err, domain.ErrNegativeMinStock),
			errors.Is(err, domain.ErrInvalidProductID),
			errors.Is(err, domain.ErrInvalidLocationID):
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		default:
			h.logger.Error("gagal update min stock", "error", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server"})
		}
		return
	}

	writeJSON(w, http.StatusOK, toStockResponse(item))
}

// Get menangani query stok spesifik (product_id & location_id) atau daftar stok di satu cabang (location_id).
// Endpoint: GET /api/v1/inventory/stocks?product_id=...&location_id=...
//           GET /api/v1/inventory/stocks?location_id=...
func (h *StockHandler) Get(w http.ResponseWriter, r *http.Request) {
	productID := r.URL.Query().Get("product_id")
	locationID := r.URL.Query().Get("location_id")

	if locationID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter query 'location_id' wajib diisi"})
		return
	}

	// 1. Jika product_id disediakan, ambil stok spesifik 1 produk
	if productID != "" {
		item, err := h.getStockUC.Execute(r.Context(), productID, locationID)
		if err != nil {
			h.logger.Error("gagal ambil stok", "error", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal mengambil data stok"})
			return
		}
		writeJSON(w, http.StatusOK, toStockResponse(item))
		return
	}

	// 2. Jika hanya location_id disediakan, ambil daftar stok di lokasi tersebut
	items, err := h.listStockUC.Execute(r.Context(), locationID)
	if err != nil {
		if errors.Is(err, application.ErrLocationNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		h.logger.Error("gagal ambil daftar stok cabang", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal mengambil daftar stok cabang"})
		return
	}

	resp := make([]*StockResponse, 0, len(items))
	for _, it := range items {
		resp = append(resp, toStockResponse(it))
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListAlerts mengambil daftar barang yang stoknya menipis (kuantitas <= min_stock).
// Endpoint: GET /api/v1/inventory/stocks/alerts?location_id=...
func (h *StockHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	locParam := r.URL.Query().Get("location_id")
	var locPtr *string
	if locParam != "" {
		locPtr = &locParam
	}

	items, err := h.listAlertsUC.Execute(r.Context(), locPtr)
	if err != nil {
		h.logger.Error("gagal list low stock alerts", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal mengambil daftar alert stok"})
		return
	}

	resp := make([]*StockResponse, 0, len(items))
	for _, it := range items {
		resp = append(resp, toStockResponse(it))
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListAdjustments menangani pembacaan riwayat catatan penyesuaian stok (Stock Opname).
// Endpoint: GET /api/v1/inventory/stocks/adjustments
func (h *StockHandler) ListAdjustments(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filter := domain.StockAdjustmentFilter{
		LocationID: q.Get("location_id"),
		ProductID:  q.Get("product_id"),
		Page:       page,
		Limit:      limit,
	}

	items, total, err := h.listAdjustmentsUC.Execute(r.Context(), filter)
	if err != nil {
		h.logger.Error("gagal list stock adjustments", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal mengambil riwayat penyesuaian stok: " + err.Error()})
		return
	}

	res := make([]StockAdjustmentResponse, len(items))
	for i, it := range items {
		res[i] = StockAdjustmentResponse{
			ID:               it.ID,
			ProductID:        it.ProductID,
			ProductName:      it.ProductName,
			ProductSKU:       it.ProductSKU,
			LocationID:       it.LocationID,
			LocationName:     it.LocationName,
			PreviousQuantity: it.PreviousQuantity,
			NewQuantity:      it.NewQuantity,
			Difference:       it.Difference,
			Reason:           it.Reason,
			AdjustedBy:       it.AdjustedBy,
			AdjustedByName:   it.AdjustedByName,
			CreatedAt:        it.CreatedAt.Format(time.RFC3339),
		}
	}

	totalPages := (total + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": res,
		"meta": map[string]any{
			"page":        page,
			"limit":       limit,
			"total_items": total,
			"total_pages": totalPages,
		},
	})
}

func toStockResponse(item *domain.StockItem) *StockResponse {
	return &StockResponse{
		ID:                item.ID,
		ProductID:         item.ProductID,
		LocationID:        item.LocationID,
		Quantity:          item.Quantity,
		ReservedQuantity:  item.ReservedQuantity,
		AvailableQuantity: item.AvailableQuantity(),
		MinStock:          item.MinStock,
		IsLowStock:        item.IsLowStock(),
		UpdatedAt:         item.UpdatedAt,
	}
}
