package interfaces

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/erp-retail/backend/internal/modules/inventory/application"
	"github.com/erp-retail/backend/internal/modules/inventory/domain"
	"github.com/erp-retail/backend/internal/shared/auth"
)

// StockTransferHandler menangani request HTTP untuk seluruh alur mutasi stok antar cabang.
type StockTransferHandler struct {
	createUC  *application.CreateStockTransferUseCase
	approveUC *application.ApproveStockTransferUseCase
	rejectUC  *application.RejectStockTransferUseCase
	shipUC    *application.ShipStockTransferUseCase
	receiveUC    *application.ReceiveStockTransferUseCase
	getUC        *application.GetStockTransferUseCase
	listUC       *application.ListStockTransfersUseCase
	userResolver UserResolver
	logger       *slog.Logger
}

// UserResolver menyediakan kapabilitas resolusi identitas pengguna menjadi nama lengkap.
type UserResolver interface {
	ResolveUserName(ctx context.Context, idOrUsername string) string
	ResolveUserNames(ctx context.Context, idsOrUsernames []string) map[string]string
}

// NewStockTransferHandler membuat instance baru StockTransferHandler.
func NewStockTransferHandler(
	createUC *application.CreateStockTransferUseCase,
	approveUC *application.ApproveStockTransferUseCase,
	rejectUC *application.RejectStockTransferUseCase,
	shipUC *application.ShipStockTransferUseCase,
	receiveUC *application.ReceiveStockTransferUseCase,
	getUC *application.GetStockTransferUseCase,
	listUC *application.ListStockTransfersUseCase,
	logger *slog.Logger,
) *StockTransferHandler {
	return &StockTransferHandler{
		createUC:  createUC,
		approveUC: approveUC,
		rejectUC:  rejectUC,
		shipUC:    shipUC,
		receiveUC: receiveUC,
		getUC:     getUC,
		listUC:    listUC,
		logger:    logger,
	}
}

// SetUserResolver menyuntikkan resolver nama pengguna untuk memperkaya response surat jalan mutasi.
func (h *StockTransferHandler) SetUserResolver(resolver UserResolver) {
	h.userResolver = resolver
}

// Create membuat permohonan mutasi stok antar cabang baru dan mencadangkan stok asal.
// Endpoint: POST /api/v1/inventory/transfers
func (h *StockTransferHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateStockTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "body request tidak valid: " + err.Error()})
		return
	}

	requestedBy := "admin"
	if claims := auth.GetClaims(r); claims != nil && claims.UserID != "" {
		requestedBy = claims.UserID
	}

	items := make([]domain.StockTransferItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, domain.StockTransferItemInput{
			ProductID:     it.ProductID,
			Quantity:      it.Quantity,
			SerialUnitIDs: it.SerialUnitIDs,
		})
	}

	cmd := application.CreateStockTransferCommand{
		FromLocationID: req.FromLocationID,
		ToLocationID:   req.ToLocationID,
		Notes:          req.Notes,
		RequestedBy:    requestedBy,
		Items:          items,
	}

	trf, err := h.createUC.Execute(r.Context(), cmd)
	if err != nil {
		if errors.Is(err, domain.ErrSameLocationTransfer) ||
			errors.Is(err, domain.ErrEmptyTransferItems) ||
			errors.Is(err, domain.ErrInvalidTransferQuantity) ||
			errors.Is(err, domain.ErrDuplicateTransferItem) {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}
		if errors.Is(err, application.ErrLocationNotFound) || errors.Is(err, application.ErrProductNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}

		h.logger.Error("gagal membuat stock transfer", "error", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, h.mapStockTransferToResponse(r.Context(), trf))
}

// List mengambil daftar riwayat dokumen transfer berdasarkan filter lokasi asal, tujuan, atau status.
// Endpoint: GET /api/v1/inventory/transfers
func (h *StockTransferHandler) List(w http.ResponseWriter, r *http.Request) {
	var filter domain.StockTransferFilter

	if fromLoc := r.URL.Query().Get("from_location_id"); fromLoc != "" {
		filter.FromLocationID = &fromLoc
	}
	if toLoc := r.URL.Query().Get("to_location_id"); toLoc != "" {
		filter.ToLocationID = &toLoc
	}
	if st := r.URL.Query().Get("status"); st != "" {
		statusVal := domain.TransferStatus(st)
		filter.Status = &statusVal
	}

	list, err := h.listUC.Execute(r.Context(), filter)
	if err != nil {
		h.logger.Error("gagal mengambil list stock transfer", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server"})
		return
	}

	writeJSON(w, http.StatusOK, h.mapTransfersList(r.Context(), list))
}

// Get mengambil detail satu dokumen transfer lengkap dengan rincian item barangnya.
// Endpoint: GET /api/v1/inventory/transfers/{id}
func (h *StockTransferHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID transfer wajib diisi"})
		return
	}

	trf, err := h.getUC.Execute(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTransferNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "dokumen mutasi stok tidak ditemukan"})
			return
		}
		h.logger.Error("gagal mengambil detail transfer", "error", err, "id", id)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server"})
		return
	}

	writeJSON(w, http.StatusOK, h.mapStockTransferToResponse(r.Context(), trf))
}

// Approve menyetujui dokumen transfer oleh Superadmin/Owner.
// Endpoint: POST /api/v1/inventory/transfers/{id}/approve
func (h *StockTransferHandler) Approve(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID transfer wajib diisi"})
		return
	}

	approvedBy := "superadmin"
	if claims := auth.GetClaims(r); claims != nil && claims.UserID != "" {
		approvedBy = claims.UserID
	}

	trf, err := h.approveUC.Execute(r.Context(), id, approvedBy)
	if err != nil {
		if errors.Is(err, domain.ErrTransferNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "dokumen mutasi stok tidak ditemukan"})
			return
		}
		if errors.Is(err, domain.ErrIllegalStatusTransition) {
			writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse{Error: "dokumen transfer tidak dalam status yang dapat disetujui (harus pending_approval)"})
			return
		}
		h.logger.Error("gagal approve stock transfer", "error", err, "id", id)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, h.mapStockTransferToResponse(r.Context(), trf))
}

// Reject menolak permohonan mutasi stok dan melepaskan reservasi stok asal.
// Endpoint: POST /api/v1/inventory/transfers/{id}/reject
func (h *StockTransferHandler) Reject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID transfer wajib diisi"})
		return
	}

	var req RejectStockTransferRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // reason opsional tapi disarankan

	rejectedBy := "superadmin"
	if claims := auth.GetClaims(r); claims != nil && claims.UserID != "" {
		rejectedBy = claims.UserID
	}

	trf, err := h.rejectUC.Execute(r.Context(), id, rejectedBy, req.Reason)
	if err != nil {
		if errors.Is(err, domain.ErrTransferNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "dokumen mutasi stok tidak ditemukan"})
			return
		}
		if errors.Is(err, domain.ErrIllegalStatusTransition) {
			writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse{Error: "dokumen transfer tidak dalam status yang dapat ditolak (harus pending_approval)"})
			return
		}
		h.logger.Error("gagal reject stock transfer", "error", err, "id", id)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, h.mapStockTransferToResponse(r.Context(), trf))
}

// Ship menandai barang telah berangkat (in_transit) dan memotong stok cabang asal.
// Endpoint: POST /api/v1/inventory/transfers/{id}/ship
func (h *StockTransferHandler) Ship(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID transfer wajib diisi"})
		return
	}

	trf, err := h.shipUC.Execute(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrTransferNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "dokumen mutasi stok tidak ditemukan"})
			return
		}
		if errors.Is(err, domain.ErrIllegalStatusTransition) {
			writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse{Error: "dokumen transfer belum disetujui untuk dikirim (harus approved)"})
			return
		}
		h.logger.Error("gagal ship stock transfer", "error", err, "id", id)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, h.mapStockTransferToResponse(r.Context(), trf))
}

// Receive menandai barang telah tiba di cabang tujuan (received), menambah stok tujuan, dan memindahkan lokasi serial.
// Endpoint: POST /api/v1/inventory/transfers/{id}/receive
func (h *StockTransferHandler) Receive(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID transfer wajib diisi"})
		return
	}

	receivedBy := "admin_gudang"
	if claims := auth.GetClaims(r); claims != nil && claims.UserID != "" {
		receivedBy = claims.UserID
	}

	trf, err := h.receiveUC.Execute(r.Context(), id, receivedBy)
	if err != nil {
		if errors.Is(err, domain.ErrTransferNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "dokumen mutasi stok tidak ditemukan"})
			return
		}
		if errors.Is(err, domain.ErrIllegalStatusTransition) {
			writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse{Error: "dokumen transfer belum dalam perjalanan (harus in_transit)"})
			return
		}
		h.logger.Error("gagal receive stock transfer", "error", err, "id", id)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, h.mapStockTransferToResponse(r.Context(), trf))
}

// Helper untuk mapping batch []*domain.StockTransfer ke []StockTransferResponse dengan resolusi nama
func (h *StockTransferHandler) mapTransfersList(ctx context.Context, list []*domain.StockTransfer) []StockTransferResponse {
	var allUserIDs []string
	for _, trf := range list {
		if trf.RequestedBy != "" {
			allUserIDs = append(allUserIDs, trf.RequestedBy)
		}
		if trf.ApprovedBy != nil && *trf.ApprovedBy != "" {
			allUserIDs = append(allUserIDs, *trf.ApprovedBy)
		}
		if trf.ReceivedBy != nil && *trf.ReceivedBy != "" {
			allUserIDs = append(allUserIDs, *trf.ReceivedBy)
		}
	}

	nameMap := make(map[string]string)
	if h.userResolver != nil && len(allUserIDs) > 0 {
		nameMap = h.userResolver.ResolveUserNames(ctx, allUserIDs)
	}

	res := make([]StockTransferResponse, 0, len(list))
	for _, trf := range list {
		res = append(res, h.mapTransferWithMap(trf, nameMap))
	}
	return res
}

// Helper untuk mapping domain.StockTransfer tunggal ke StockTransferResponse dengan resolusi nama
func (h *StockTransferHandler) mapStockTransferToResponse(ctx context.Context, trf *domain.StockTransfer) StockTransferResponse {
	var userIDs []string
	if trf.RequestedBy != "" {
		userIDs = append(userIDs, trf.RequestedBy)
	}
	if trf.ApprovedBy != nil && *trf.ApprovedBy != "" {
		userIDs = append(userIDs, *trf.ApprovedBy)
	}
	if trf.ReceivedBy != nil && *trf.ReceivedBy != "" {
		userIDs = append(userIDs, *trf.ReceivedBy)
	}

	nameMap := make(map[string]string)
	if h.userResolver != nil && len(userIDs) > 0 {
		nameMap = h.userResolver.ResolveUserNames(ctx, userIDs)
	}

	return h.mapTransferWithMap(trf, nameMap)
}

func (h *StockTransferHandler) mapTransferWithMap(trf *domain.StockTransfer, nameMap map[string]string) StockTransferResponse {
	items := make([]StockTransferItemResponse, 0, len(trf.Items))
	for _, it := range trf.Items {
		items = append(items, StockTransferItemResponse{
			ID:               it.ID,
			ProductID:        it.ProductID,
			Quantity:         it.Quantity,
			ReceivedQuantity: it.ReceivedQuantity,
			SerialUnitIDs:    it.SerialUnitIDs,
			CreatedAt:        it.CreatedAt,
		})
	}

	requestedByName := trf.RequestedBy
	if val, ok := nameMap[trf.RequestedBy]; ok && val != "" {
		requestedByName = val
	}

	var approvedByName *string
	if trf.ApprovedBy != nil && *trf.ApprovedBy != "" {
		val := *trf.ApprovedBy
		if mapped, ok := nameMap[*trf.ApprovedBy]; ok && mapped != "" {
			val = mapped
		}
		approvedByName = &val
	}

	var receivedByName *string
	if trf.ReceivedBy != nil && *trf.ReceivedBy != "" {
		val := *trf.ReceivedBy
		if mapped, ok := nameMap[*trf.ReceivedBy]; ok && mapped != "" {
			val = mapped
		}
		receivedByName = &val
	}

	return StockTransferResponse{
		ID:              trf.ID,
		TransferNumber:  trf.TransferNumber,
		FromLocationID:  trf.FromLocationID,
		ToLocationID:    trf.ToLocationID,
		Status:          string(trf.Status),
		Notes:           trf.Notes,
		RejectionReason: trf.RejectionReason,
		RequestedBy:     trf.RequestedBy,
		RequestedByName: requestedByName,
		ApprovedBy:      trf.ApprovedBy,
		ApprovedByName:  approvedByName,
		ReceivedBy:      trf.ReceivedBy,
		ReceivedByName:  receivedByName,
		Items:           items,
		CreatedAt:       trf.CreatedAt,
		UpdatedAt:       trf.UpdatedAt,
	}
}
