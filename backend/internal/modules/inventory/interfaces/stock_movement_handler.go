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

// StockMovementHandler menangani request HTTP untuk transaksi Barang Masuk, Barang Keluar, dan Laporan Kartu Stok.
type StockMovementHandler struct {
	createMovementUC *application.CreateStockMovementUseCase
	getDetailUC      *application.GetStockMovementDetailUseCase
	listMovementsUC  *application.ListStockMovementsUseCase
	getStockCardUC   *application.GetStockCardReportUseCase
	getValuationUC   *application.GetStockValuationReportUseCase
	logger           *slog.Logger
}

func NewStockMovementHandler(
	createMovementUC *application.CreateStockMovementUseCase,
	getDetailUC *application.GetStockMovementDetailUseCase,
	listMovementsUC *application.ListStockMovementsUseCase,
	getStockCardUC *application.GetStockCardReportUseCase,
	getValuationUC *application.GetStockValuationReportUseCase,
	logger *slog.Logger,
) *StockMovementHandler {
	return &StockMovementHandler{
		createMovementUC: createMovementUC,
		getDetailUC:      getDetailUC,
		listMovementsUC:  listMovementsUC,
		getStockCardUC:   getStockCardUC,
		getValuationUC:   getValuationUC,
		logger:           logger,
	}
}

// CreateStockIn menangani pencatatan transaksi Barang Masuk (Inbound / Stock In).
// Endpoint: POST /api/v1/inventory/movements/in
func (h *StockMovementHandler) CreateStockIn(w http.ResponseWriter, r *http.Request) {
	h.handleCreateMovement(w, r, domain.MovementTypeIn)
}

// CreateStockOut menangani pencatatan transaksi Barang Keluar (Outbound / Stock Out).
// Endpoint: POST /api/v1/inventory/movements/out
func (h *StockMovementHandler) CreateStockOut(w http.ResponseWriter, r *http.Request) {
	h.handleCreateMovement(w, r, domain.MovementTypeOut)
}

func (h *StockMovementHandler) handleCreateMovement(w http.ResponseWriter, r *http.Request, movType domain.StockMovementType) {
	var req CreateStockMovementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "body request tidak valid: " + err.Error()})
		return
	}

	executedBy := "system"
	executedByName := "Staf Toko"
	if claims := auth.GetClaims(r); claims != nil {
		executedBy = claims.UserID
		if claims.Subject != "" {
			executedByName = claims.Subject
		} else if claims.Role != "" {
			executedByName = "Admin (" + claims.Role + ")"
		}

		// Validasi isolasi cabang: admin gudang hanya boleh catat barang masuk/keluar di gudangnya
		if (claims.Role == "warehouse" || claims.Role == "cashier") && claims.Location != "" {
			if req.LocationID != claims.Location {
				writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "akses ditolak: Anda hanya diizinkan mencatat mutasi barang di gudang yang ditugaskan kepada Anda"})
				return
			}
		}
	}

	var itemsInput []application.StockMovementItemInput
	for _, it := range req.Items {
		itemsInput = append(itemsInput, application.StockMovementItemInput{
			ProductID:     it.ProductID,
			Quantity:      it.Quantity,
			Notes:         it.Notes,
			SerialNumbers: it.SerialNumbers,
		})
	}

	movDate := time.Now().UTC()
	if req.MovementDate != "" {
		if parsed, err := time.Parse("2006-01-02", req.MovementDate); err == nil {
			movDate = parsed
		}
	}

	cmd := application.CreateStockMovementCommand{
		Type:            movType,
		MovementDate:    movDate,
		LocationID:      req.LocationID,
		CategoryReason:  req.CategoryReason,
		ReferenceNumber: req.ReferenceNumber,
		Notes:           req.Notes,
		ExecutedBy:      executedBy,
		ExecutedByName:  executedByName,
		Items:           itemsInput,
	}

	movement, err := h.createMovementUC.Execute(r.Context(), cmd)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrLocationNotFound):
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: err.Error()})
		case errors.Is(err, application.ErrLocationInactive),
			errors.Is(err, domain.ErrInvalidMovementType),
			errors.Is(err, domain.ErrInvalidLocationID),
			errors.Is(err, domain.ErrEmptyMovementItems),
			errors.Is(err, domain.ErrInvalidMovementItemQty),
			errors.Is(err, domain.ErrMovementSerialMismatch),
			errors.Is(err, domain.ErrMovementItemDuplicate),
			errors.Is(err, application.ErrInsufficientStockForMovement):
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		default:
			h.logger.Error("gagal membuat pergerakan stok", "error", err, "type", movType)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server: " + err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusCreated, toStockMovementResponse(movement))
}

// GetDetail mengambil detail lengkap pergerakan stok beserta item dan serial number.
// Endpoint: GET /api/v1/inventory/movements/{id}
func (h *StockMovementHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID dokumen wajib diisi"})
		return
	}

	m, err := h.getDetailUC.Execute(r.Context(), id)
	if err != nil {
		h.logger.Error("gagal mengambil detail pergerakan stok", "id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server"})
		return
	}
	if m == nil {
		writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "dokumen pergerakan stok tidak ditemukan"})
		return
	}

	writeJSON(w, http.StatusOK, toStockMovementResponse(m))
}

// List mengambil daftar riwayat transaksi Barang Masuk dan Barang Keluar.
// Endpoint: GET /api/v1/inventory/movements
func (h *StockMovementHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	locationID := q.Get("location_id")
	typeStr := q.Get("type")

	page := 1
	if p, err := strconv.Atoi(q.Get("page")); err == nil && p > 0 {
		page = p
	}
	limit := 10
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	var movType *domain.StockMovementType
	if typeStr == "in" || typeStr == "out" {
		t := domain.StockMovementType(typeStr)
		movType = &t
	}

	var startDate, endDate *time.Time
	if s := q.Get("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			startDate = &t
		}
	}
	if e := q.Get("end_date"); e != "" {
		if t, err := time.Parse("2006-01-02", e); err == nil {
			endDate = &t
		}
	}

	if claims := auth.GetClaims(r); claims != nil {
		if (claims.Role == "warehouse" || claims.Role == "cashier") && claims.Location != "" {
			locationID = claims.Location
		}
	}

	filter := domain.StockMovementFilter{
		LocationID: locationID,
		Type:       movType,
		StartDate:  startDate,
		EndDate:    endDate,
		Page:       page,
		Limit:      limit,
	}

	list, total, err := h.listMovementsUC.Execute(r.Context(), filter)
	if err != nil {
		h.logger.Error("gagal list pergerakan stok", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server"})
		return
	}

	itemsResp := make([]StockMovementResponse, 0)
	for _, m := range list {
		itemsResp = append(itemsResp, toStockMovementResponse(m))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data":  itemsResp,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetStockCard mengambil laporan buku besar kartu stok untuk 1 produk di 1 cabang.
// Endpoint: GET /api/v1/inventory/reports/stock-card
func (h *StockMovementHandler) GetStockCard(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	productID := q.Get("product_id")
	locationID := q.Get("location_id")

	// Validasi isolasi cabang: admin gudang hanya boleh melihat kartu stok di gudangnya
	if claims := auth.GetClaims(r); claims != nil {
		if (claims.Role == "warehouse" || claims.Role == "cashier") && claims.Location != "" {
			if locationID != "" && locationID != claims.Location {
				writeJSON(w, http.StatusForbidden, ErrorResponse{Error: "akses ditolak: Anda hanya diizinkan melihat kartu stok di gudang yang ditugaskan kepada Anda"})
				return
			}
			locationID = claims.Location
		}
	}

	if productID == "" || locationID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter product_id dan location_id wajib diisi"})
		return
	}

	var startDate, endDate *time.Time
	if s := q.Get("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			startDate = &t
		}
	}
	if e := q.Get("end_date"); e != "" {
		if t, err := time.Parse("2006-01-02", e); err == nil {
			// Jadikan akhir hari (23:59:59)
			endOfDay := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			endDate = &endOfDay
		}
	}

	report, err := h.getStockCardUC.Execute(r.Context(), productID, locationID, startDate, endDate)
	if err != nil {
		h.logger.Error("gagal generate laporan kartu stok", "product_id", productID, "location_id", locationID, "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, toStockCardReportResponse(report))
}

// GetValuationReport mengambil ringkasan nilai valuasi persediaan di cabang.
// Endpoint: GET /api/v1/inventory/reports/valuation
func (h *StockMovementHandler) GetValuationReport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var locID *string
	if claims := auth.GetClaims(r); claims != nil {
		if (claims.Role == "warehouse" || claims.Role == "cashier") && claims.Location != "" {
			locID = &claims.Location
		}
	}
	if locID == nil {
		if loc := q.Get("location_id"); loc != "" {
			locID = &loc
		}
	}

	report, err := h.getValuationUC.Execute(r.Context(), locID)
	if err != nil {
		h.logger.Error("gagal generate laporan valuasi", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		return
	}

	resp := make([]StockValuationItemResponse, 0)
	for _, it := range report {
		resp = append(resp, StockValuationItemResponse{
			ProductID:      it.ProductID,
			ProductSKU:     it.ProductSKU,
			ProductName:    it.ProductName,
			CategoryName:   it.CategoryName,
			LocationID:     it.LocationID,
			LocationName:   it.LocationName,
			Quantity:       it.Quantity,
			MinStock:       it.MinStock,
			BasePrice:      it.BasePrice,
			TotalValuation: it.TotalValuation,
			Status:         it.Status,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// --- Mapper Helpers ---

func toStockMovementResponse(m *domain.StockMovement) StockMovementResponse {
	itemsResp := make([]StockMovementItemResponse, 0)
	for _, it := range m.Items {
		itemsResp = append(itemsResp, StockMovementItemResponse{
			ID:            it.ID,
			ProductID:     it.ProductID,
			ProductName:   it.ProductName,
			ProductSKU:    it.ProductSKU,
			Quantity:      it.Quantity,
			Notes:         it.Notes,
			SerialNumbers: it.SerialNumbers,
			CreatedAt:     it.CreatedAt,
		})
	}

	return StockMovementResponse{
		ID:              m.ID,
		MovementNumber:  m.MovementNumber,
		Type:            string(m.Type),
		MovementDate:    m.MovementDate.Format("2006-01-02"),
		LocationID:      m.LocationID,
		LocationName:    m.LocationName,
		CategoryReason:  m.CategoryReason,
		ReferenceNumber: m.ReferenceNumber,
		Notes:           m.Notes,
		ExecutedBy:      m.ExecutedBy,
		ExecutedByName:  m.ExecutedByName,
		Items:           itemsResp,
		CreatedAt:       m.CreatedAt,
	}
}

func toStockCardReportResponse(r *domain.StockCardReport) StockCardReportResponse {
	entriesResp := make([]StockCardEntryResponse, 0)
	for _, e := range r.Entries {
		entriesResp = append(entriesResp, StockCardEntryResponse{
			Date:            e.Date,
			MovementType:    e.MovementType,
			DocumentNumber:  e.DocumentNumber,
			ReferenceNumber: e.ReferenceNumber,
			CategoryReason:  e.CategoryReason,
			InQuantity:      e.InQuantity,
			OutQuantity:     e.OutQuantity,
			Balance:         e.Balance,
			ExecutedByName:  e.ExecutedByName,
			Notes:           e.Notes,
		})
	}

	return StockCardReportResponse{
		ProductID:      r.ProductID,
		ProductName:    r.ProductName,
		ProductSKU:     r.ProductSKU,
		LocationID:     r.LocationID,
		LocationName:   r.LocationName,
		OpeningBalance: r.OpeningBalance,
		TotalIn:        r.TotalIn,
		TotalOut:       r.TotalOut,
		ClosingBalance: r.ClosingBalance,
		Entries:        entriesResp,
	}
}
