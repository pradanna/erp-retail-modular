package interfaces

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/erp-retail/backend/internal/modules/inventory/application"
	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

// LocationHandler menangani seluruh HTTP request untuk master lokasi cabang/gudang.
type LocationHandler struct {
	createLocation *application.CreateLocationUseCase
	listLocations  *application.ListLocationsUseCase
	getLocation    *application.GetLocationUseCase
	updateLocation *application.UpdateLocationUseCase
	setStatus      *application.SetLocationStatusUseCase
	deleteLocation *application.DeleteLocationUseCase
	logger         *slog.Logger
}

// NewLocationHandler membuat instance LocationHandler dengan dependensi use case terkait.
func NewLocationHandler(
	createLocation *application.CreateLocationUseCase,
	listLocations *application.ListLocationsUseCase,
	getLocation *application.GetLocationUseCase,
	updateLocation *application.UpdateLocationUseCase,
	setStatus *application.SetLocationStatusUseCase,
	deleteLocation *application.DeleteLocationUseCase,
	logger *slog.Logger,
) *LocationHandler {
	return &LocationHandler{
		createLocation: createLocation,
		listLocations:  listLocations,
		getLocation:    getLocation,
		updateLocation: updateLocation,
		setStatus:      setStatus,
		deleteLocation: deleteLocation,
		logger:         logger,
	}
}

// Create menangani POST /api/v1/inventory/locations
func (h *LocationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "format JSON request tidak valid: " + err.Error()})
		return
	}

	cmd := application.CreateLocationCommand{
		Code:    req.Code,
		Name:    req.Name,
		Type:    domain.LocationType(req.Type),
		Address: req.Address,
	}

	id, err := h.createLocation.Execute(r.Context(), cmd)
	if err != nil {
		if errors.Is(err, application.ErrDuplicateCode) {
			writeJSON(w, http.StatusConflict, ErrorResponse{Error: err.Error()})
			return
		}
		h.logger.Error("gagal membuat lokasi", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"id":      id,
		"message": "Lokasi cabang berhasil dibuat",
	})
}

// List menangani GET /api/v1/inventory/locations
func (h *LocationHandler) List(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active_only") == "true"

	locations, err := h.listLocations.Execute(r.Context(), activeOnly)
	if err != nil {
		h.logger.Error("gagal mengambil daftar lokasi", "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal mengambil data lokasi"})
		return
	}

	var data []*LocationResponse
	for _, l := range locations {
		data = append(data, toLocationResponse(l))
	}
	if data == nil {
		data = []*LocationResponse{}
	}

	writeJSON(w, http.StatusOK, data)
}

// GetByID menangani GET /api/v1/inventory/locations/{id}
func (h *LocationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "ID lokasi wajib diisi"})
		return
	}

	loc, err := h.getLocation.Execute(r.Context(), id)
	if err != nil {
		if errors.Is(err, application.ErrLocationNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "lokasi tidak ditemukan"})
			return
		}
		h.logger.Error("gagal mengambil detail lokasi", "id", id, "error", err.Error())
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal mengambil detail lokasi"})
		return
	}

	writeJSON(w, http.StatusOK, toLocationResponse(loc))
}

// Update menangani PUT /api/v1/inventory/locations/{id}
func (h *LocationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "ID lokasi wajib diisi"})
		return
	}

	var req UpdateLocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "format JSON request tidak valid: " + err.Error()})
		return
	}

	cmd := application.UpdateLocationCommand{
		ID:      id,
		Code:    req.Code,
		Name:    req.Name,
		Type:    domain.LocationType(req.Type),
		Address: req.Address,
	}

	if err := h.updateLocation.Execute(r.Context(), cmd); err != nil {
		if errors.Is(err, application.ErrLocationNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "lokasi tidak ditemukan"})
			return
		}
		if errors.Is(err, application.ErrDuplicateCode) {
			writeJSON(w, http.StatusConflict, ErrorResponse{Error: err.Error()})
			return
		}
		h.logger.Error("gagal memperbarui lokasi", "id", id, "error", err.Error())
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Informasi lokasi berhasil diperbarui",
	})
}

// SetStatus menangani PATCH /api/v1/inventory/locations/{id}/status
func (h *LocationHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "ID lokasi wajib diisi"})
		return
	}

	var req SetLocationStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "format JSON request tidak valid: " + err.Error()})
		return
	}

	if err := h.setStatus.Execute(r.Context(), id, req.IsActive); err != nil {
		if errors.Is(err, application.ErrLocationNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "lokasi tidak ditemukan"})
			return
		}
		h.logger.Error("gagal mengubah status lokasi", "id", id, "error", err.Error())
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	msg := "Lokasi dinonaktifkan (tutup operasional)"
	if req.IsActive {
		msg = "Lokasi berhasil diaktifkan kembali"
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": msg,
	})
}

// Delete menangani DELETE /api/v1/inventory/locations/{id}
func (h *LocationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "ID lokasi wajib diisi"})
		return
	}

	if err := h.deleteLocation.Execute(r.Context(), id); err != nil {
		if errors.Is(err, application.ErrLocationNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "lokasi tidak ditemukan"})
			return
		}
		h.logger.Error("gagal menghapus lokasi", "id", id, "error", err.Error())
		writeJSON(w, http.StatusConflict, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "Lokasi berhasil dihapus",
	})
}

// Helper mapper domain -> DTO response
func toLocationResponse(l *domain.Location) *LocationResponse {
	return &LocationResponse{
		ID:        l.ID,
		Code:      l.Code,
		Name:      l.Name,
		Type:      string(l.Type),
		Address:   l.Address,
		IsActive:  l.IsActive,
		CreatedAt: l.CreatedAt,
		UpdatedAt: l.UpdatedAt,
	}
}
