package interfaces

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/erp-retail/backend/internal/modules/inventory/application"
	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

// WarrantyHandler menangani request HTTP untuk master kebijakan garansi dan penetapan garansi produk.
type WarrantyHandler struct {
	createPolicyUC          *application.CreateWarrantyPolicyUseCase
	listPoliciesUC          *application.ListWarrantyPoliciesUseCase
	assignProductWarrantyUC *application.AssignProductWarrantyUseCase
	getProductWarrantiesUC  *application.GetProductActiveWarrantiesUseCase
	deactivateWarrantyUC    *application.DeactivateProductWarrantyUseCase
	logger                  *slog.Logger
}

// NewWarrantyHandler membuat instance baru WarrantyHandler.
func NewWarrantyHandler(
	createPolicyUC *application.CreateWarrantyPolicyUseCase,
	listPoliciesUC *application.ListWarrantyPoliciesUseCase,
	assignProductWarrantyUC *application.AssignProductWarrantyUseCase,
	getProductWarrantiesUC *application.GetProductActiveWarrantiesUseCase,
	deactivateWarrantyUC *application.DeactivateProductWarrantyUseCase,
	logger *slog.Logger,
) *WarrantyHandler {
	return &WarrantyHandler{
		createPolicyUC:          createPolicyUC,
		listPoliciesUC:          listPoliciesUC,
		assignProductWarrantyUC: assignProductWarrantyUC,
		getProductWarrantiesUC:  getProductWarrantiesUC,
		deactivateWarrantyUC:    deactivateWarrantyUC,
		logger:                  logger,
	}
}

// CreatePolicy membuat master kebijakan garansi baru.
// POST /api/v1/inventory/warranties/policies
func (h *WarrantyHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	var req CreateWarrantyPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "body request tidak valid: " + err.Error()})
		return
	}

	cmd := application.CreateWarrantyPolicyCommand{
		Name:              req.Name,
		Type:              domain.WarrantyType(req.Type),
		DurationMonths:    req.DurationMonths,
		DurationDays:      req.DurationDays,
		Coverage:          req.Coverage,
		ClaimInstructions: req.ClaimInstructions,
	}

	policy, err := h.createPolicyUC.Execute(r.Context(), cmd)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidWarrantyName),
			errors.Is(err, domain.ErrInvalidWarrantyType),
			errors.Is(err, domain.ErrInvalidWarrantyDuration):
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		default:
			h.logger.Error("gagal membuat master kebijakan garansi", "error", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal membuat kebijakan garansi: " + err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusCreated, toWarrantyPolicyResponse(policy))
}

// ListPolicies menampilkan daftar master kebijakan garansi.
// GET /api/v1/inventory/warranties/policies
func (h *WarrantyHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	query := application.ListWarrantyPoliciesQuery{}

	typeParam := r.URL.Query().Get("type")
	if typeParam != "" {
		wType := domain.WarrantyType(typeParam)
		if !wType.IsValid() {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "tipe garansi filter harus 'toko' atau 'pabrik'"})
			return
		}
		query.Type = &wType
	}

	activeOnlyParam := r.URL.Query().Get("active_only")
	if activeOnlyParam == "true" {
		query.IsActiveOnly = true
	}

	policies, err := h.listPoliciesUC.Execute(r.Context(), query)
	if err != nil {
		h.logger.Error("gagal mengambil daftar kebijakan garansi", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal mengambil kebijakan garansi: " + err.Error()})
		return
	}

	resp := make([]WarrantyPolicyResponse, 0, len(policies))
	for _, p := range policies {
		resp = append(resp, toWarrantyPolicyResponse(p))
	}

	writeJSON(w, http.StatusOK, resp)
}

// AssignProductWarranty menetapkan kebijakan garansi ke suatu produk.
// Invariant Kunci: Otomatis menonaktifkan garansi aktif sebelumnya dengan tipe ('toko'/'pabrik') yang sama.
// POST /api/v1/inventory/products/{id}/warranties
func (h *WarrantyHandler) AssignProductWarranty(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID produk wajib diisi"})
		return
	}

	var req AssignProductWarrantyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "body request tidak valid: " + err.Error()})
		return
	}

	cmd := application.AssignProductWarrantyCommand{
		ProductID:        productID,
		WarrantyPolicyID: req.WarrantyPolicyID,
	}

	pw, err := h.assignProductWarrantyUC.Execute(r.Context(), cmd)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrProductNotFound),
			errors.Is(err, application.ErrWarrantyPolicyNotFound),
			errors.Is(err, domain.ErrPolicyNotFound):
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: err.Error()})
		case errors.Is(err, domain.ErrInvalidProductID),
			errors.Is(err, domain.ErrInvalidPolicyID),
			errors.Is(err, domain.ErrInvalidWarrantyType):
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		default:
			h.logger.Error("gagal menugaskan garansi ke produk", "error", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal menugaskan garansi: " + err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusCreated, toProductWarrantyResponse(pw))
}

// GetProductWarranties mengambil garansi aktif yang berlaku pada suatu produk.
// GET /api/v1/inventory/products/{id}/warranties
func (h *WarrantyHandler) GetProductWarranties(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID produk wajib diisi"})
		return
	}

	warranties, err := h.getProductWarrantiesUC.Execute(r.Context(), productID)
	if err != nil {
		if errors.Is(err, application.ErrProductNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		h.logger.Error("gagal mengambil garansi produk", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal mengambil garansi produk: " + err.Error()})
		return
	}

	resp := make([]ProductWarrantyResponse, 0, len(warranties))
	for _, pw := range warranties {
		resp = append(resp, toProductWarrantyResponse(pw))
	}

	writeJSON(w, http.StatusOK, resp)
}

// DeactivateProductWarranty menonaktifkan penugasan garansi produk secara manual.
// POST /api/v1/inventory/warranties/products/{id}/deactivate
func (h *WarrantyHandler) DeactivateProductWarranty(w http.ResponseWriter, r *http.Request) {
	productWarrantyID := r.PathValue("id")
	if productWarrantyID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID garansi produk wajib diisi"})
		return
	}

	pw, err := h.deactivateWarrantyUC.Execute(r.Context(), productWarrantyID)
	if err != nil {
		if errors.Is(err, domain.ErrProductWarrantyNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		h.logger.Error("gagal menonaktifkan garansi produk", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal menonaktifkan garansi: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, toProductWarrantyResponse(pw))
}

func toWarrantyPolicyResponse(p *domain.WarrantyPolicy) WarrantyPolicyResponse {
	return WarrantyPolicyResponse{
		ID:                p.ID,
		Name:              p.Name,
		Type:              string(p.Type),
		DurationMonths:    p.DurationMonths,
		DurationDays:      p.DurationDays,
		Coverage:          p.Coverage,
		ClaimInstructions: p.ClaimInstructions,
		IsActive:          p.IsActive,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}

func toProductWarrantyResponse(pw *domain.ProductWarranty) ProductWarrantyResponse {
	resp := ProductWarrantyResponse{
		ID:               pw.ID,
		ProductID:        pw.ProductID,
		WarrantyPolicyID: pw.WarrantyPolicyID,
		Type:             string(pw.Type),
		IsActive:         pw.IsActive,
		CreatedAt:        pw.CreatedAt,
		UpdatedAt:        pw.UpdatedAt,
	}
	if pw.Policy != nil {
		polResp := toWarrantyPolicyResponse(pw.Policy)
		resp.Policy = &polResp
	}
	return resp
}
