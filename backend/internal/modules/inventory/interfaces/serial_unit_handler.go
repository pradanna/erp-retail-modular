package interfaces

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/erp-retail/backend/internal/modules/inventory/application"
	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

// SerialUnitHandler menangani request HTTP untuk registrasi, lookup, dan mutasi unit fisik berserial.
type SerialUnitHandler struct {
	registerUC   *application.RegisterSerialUnitsUseCase
	lookupUC     *application.LookupSerialNumberUseCase
	listUC       *application.ListSerialUnitsUseCase
	updateStatusUC *application.UpdateSerialStatusUseCase
	logger       *slog.Logger
}

// NewSerialUnitHandler membuat instance baru SerialUnitHandler.
func NewSerialUnitHandler(
	registerUC *application.RegisterSerialUnitsUseCase,
	lookupUC *application.LookupSerialNumberUseCase,
	listUC *application.ListSerialUnitsUseCase,
	updateStatusUC *application.UpdateSerialStatusUseCase,
	logger *slog.Logger,
) *SerialUnitHandler {
	return &SerialUnitHandler{
		registerUC:     registerUC,
		lookupUC:       lookupUC,
		listUC:         listUC,
		updateStatusUC: updateStatusUC,
		logger:         logger,
	}
}

// Register mendaftarkan nomor seri/IMEI baru untuk suatu produk di lokasi tertentu.
// Endpoint: POST /api/v1/inventory/products/{id}/serials
func (h *SerialUnitHandler) Register(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID produk wajib diisi"})
		return
	}

	var req RegisterSerialUnitsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "body request tidak valid: " + err.Error()})
		return
	}

	cmd := application.RegisterSerialUnitsCommand{
		ProductID:     productID,
		LocationID:    req.LocationID,
		SerialNumbers: req.SerialNumbers,
	}

	units, err := h.registerUC.Execute(r.Context(), cmd)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotTrackedBySerial) {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}
		if errors.Is(err, domain.ErrDuplicateSerialNumber) {
			writeJSON(w, http.StatusConflict, ErrorResponse{Error: err.Error()})
			return
		}
		if errors.Is(err, domain.ErrInvalidSerialNumber) || errors.Is(err, domain.ErrSerialHasInvalidChar) {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}
		if errors.Is(err, application.ErrProductNotFound) || errors.Is(err, application.ErrLocationNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}

		h.logger.Error("gagal mendaftarkan serial units", "error", err, "product_id", productID)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server: " + err.Error()})
		return
	}

	var responseList []SerialUnitResponse
	for _, u := range units {
		responseList = append(responseList, mapSerialUnitToResponse(u))
	}

	writeJSON(w, http.StatusCreated, responseList)
}

// List mengambil daftar seluruh unit fisik berserial dengan filter (produk, cabang, status, pencarian serial).
// Endpoint: GET /api/v1/inventory/serials
func (h *SerialUnitHandler) List(w http.ResponseWriter, r *http.Request) {
	var productID *string
	if pid := r.URL.Query().Get("product_id"); pid != "" {
		productID = &pid
	}
	var locationID *string
	if lid := r.URL.Query().Get("location_id"); lid != "" {
		locationID = &lid
	}
	var statusFilter *domain.SerialStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := domain.SerialStatus(s)
		if st.IsValid() {
			statusFilter = &st
		}
	}
	var search *string
	if q := r.URL.Query().Get("search"); q != "" {
		search = &q
	}

	query := application.ListSerialUnitsQuery{
		ProductID:  productID,
		LocationID: locationID,
		Status:     statusFilter,
		Search:     search,
	}

	details, err := h.listUC.Execute(r.Context(), query)
	if err != nil {
		h.logger.Error("gagal mengambil daftar serial unit", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server"})
		return
	}

	responseList := make([]SerialUnitResponse, 0, len(details))
	for _, d := range details {
		responseList = append(responseList, mapDetailToResponse(d))
	}

	writeJSON(w, http.StatusOK, responseList)
}

// ListByProduct mengambil seluruh unit fisik berserial milik suatu produk.
// Endpoint: GET /api/v1/inventory/products/{id}/serials
func (h *SerialUnitHandler) ListByProduct(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID produk wajib diisi"})
		return
	}

	var statusFilter *domain.SerialStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := domain.SerialStatus(s)
		if st.IsValid() {
			statusFilter = &st
		}
	}

	query := application.ListSerialUnitsQuery{
		ProductID: &productID,
		Status:    statusFilter,
	}

	details, err := h.listUC.Execute(r.Context(), query)
	if err != nil {
		h.logger.Error("gagal mengambil daftar serial unit produk", "error", err, "product_id", productID)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server"})
		return
	}

	responseList := make([]SerialUnitResponse, 0, len(details))
	for _, d := range details {
		responseList = append(responseList, mapDetailToResponse(d))
	}

	writeJSON(w, http.StatusOK, responseList)
}

// Lookup mencari unit fisik berdasarkan barcode scan S/N atau IMEI.
// Endpoint: GET /api/v1/inventory/serials/lookup?sn=...
func (h *SerialUnitHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	sn := r.URL.Query().Get("sn")
	if sn == "" {
		sn = r.URL.Query().Get("serial")
	}
	if sn == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter query 'sn' atau 'serial' (serial number) wajib diisi"})
		return
	}

	detail, err := h.lookupUC.Execute(r.Context(), sn)
	if err != nil {
		if errors.Is(err, domain.ErrSerialNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "unit fisik dengan nomor seri tersebut tidak ditemukan"})
			return
		}
		if errors.Is(err, domain.ErrInvalidSerialNumber) {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}

		h.logger.Error("gagal lookup serial number", "error", err, "sn", sn)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server"})
		return
	}

	res := SerialUnitLookupResponse{
		SerialUnit:   ptr(mapSerialUnitToResponse(detail.Unit)),
		ProductName:  detail.ProductName,
		ProductSKU:   detail.ProductSKU,
		ProductBrand: detail.ProductBrand,
		LocationName: detail.LocationName,
		LocationCode: detail.LocationCode,
	}

	writeJSON(w, http.StatusOK, res)
}

// UpdateStatus memperbarui status siklus hidup unit (misal: terjual atau retur).
// Endpoint: PATCH /api/v1/inventory/serials/{id}/status
func (h *SerialUnitHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	serialID := r.PathValue("id")
	if serialID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID serial wajib diisi"})
		return
	}

	var req UpdateSerialStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "body request tidak valid: " + err.Error()})
		return
	}

	cmd := application.UpdateSerialStatusCommand{
		ID:        serialID,
		NewStatus: domain.SerialStatus(req.Status),
	}

	unit, err := h.updateStatusUC.Execute(r.Context(), cmd)
	if err != nil {
		if errors.Is(err, domain.ErrSerialNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "serial unit tidak ditemukan"})
			return
		}
		if errors.Is(err, domain.ErrSerialAlreadySold) || errors.Is(err, domain.ErrSerialNotSold) || errors.Is(err, domain.ErrSerialNotAvailable) {
			writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse{Error: err.Error()})
			return
		}

		h.logger.Error("gagal update status serial unit", "error", err, "serial_id", serialID)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, mapSerialUnitToResponse(unit))
}

// Helper untuk mapping domain.SerialUnit ke SerialUnitResponse
func mapSerialUnitToResponse(u *domain.SerialUnit) SerialUnitResponse {
	return SerialUnitResponse{
		ID:           u.ID,
		ProductID:    u.ProductID,
		LocationID:   u.LocationID,
		SerialNumber: u.SerialNumber,
		Status:       string(u.Status),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

// Helper untuk mapping application.SerialUnitDetail ke SerialUnitResponse
func mapDetailToResponse(d *application.SerialUnitDetail) SerialUnitResponse {
	resp := mapSerialUnitToResponse(d.Unit)
	resp.ProductName = d.ProductName
	resp.ProductSKU = d.ProductSKU
	resp.ProductBrand = d.ProductBrand
	resp.LocationName = d.LocationName
	resp.LocationCode = d.LocationCode
	return resp
}

// Helper untuk membuat pointer struct
func ptr[T any](v T) *T {
	return &v
}
