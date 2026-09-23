package interfaces

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/erp-retail/backend/internal/modules/inventory/application"
	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

// StockHandler menangani request HTTP untuk manajemen stok barang per cabang/gudang.
type StockHandler struct {
	adjustStockUC    *application.AdjustStockUseCase
	getStockUC       *application.GetStockUseCase
	listStockUC      *application.ListStockByLocationUseCase
	listAlertsUC     *application.ListLowStockAlertsUseCase
	updateMinStockUC *application.UpdateMinStockUseCase
	logger           *slog.Logger
}

func NewStockHandler(
	adjustStockUC *application.AdjustStockUseCase,
	getStockUC *application.GetStockUseCase,
	listStockUC *application.ListStockByLocationUseCase,
	listAlertsUC *application.ListLowStockAlertsUseCase,
	updateMinStockUC *application.UpdateMinStockUseCase,
	logger *slog.Logger,
) *StockHandler {
	return &StockHandler{
		adjustStockUC:    adjustStockUC,
		getStockUC:       getStockUC,
		listStockUC:      listStockUC,
		listAlertsUC:     listAlertsUC,
		updateMinStockUC: updateMinStockUC,
		logger:           logger,
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

	cmd := application.AdjustStockCommand{
		ProductID:   req.ProductID,
		LocationID:  req.LocationID,
		NewQuantity: req.NewQuantity,
		Reason:      req.Reason,
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
