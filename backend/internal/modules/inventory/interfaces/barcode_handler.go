package interfaces

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/erp-retail/backend/internal/modules/inventory/application"
	"github.com/erp-retail/backend/internal/modules/inventory/domain"
)

// BarcodeHandler menangani request HTTP terkait pendaftaran dan scan barcode produk.
type BarcodeHandler struct {
	addBarcodeUC     *application.AddBarcodeUseCase
	deleteBarcodeUC  *application.DeleteBarcodeUseCase
	listBarcodesUC   *application.ListBarcodesByProductUseCase
	lookupBarcodeUC  *application.LookupProductByBarcodeUseCase
	logger           *slog.Logger
}

func NewBarcodeHandler(
	addBarcodeUC *application.AddBarcodeUseCase,
	deleteBarcodeUC *application.DeleteBarcodeUseCase,
	listBarcodesUC *application.ListBarcodesByProductUseCase,
	lookupBarcodeUC *application.LookupProductByBarcodeUseCase,
	logger *slog.Logger,
) *BarcodeHandler {
	return &BarcodeHandler{
		addBarcodeUC:    addBarcodeUC,
		deleteBarcodeUC: deleteBarcodeUC,
		listBarcodesUC:  listBarcodesUC,
		lookupBarcodeUC: lookupBarcodeUC,
		logger:          logger,
	}
}

// Add menangani pendaftaran barcode pabrik baru ke suatu produk.
// Endpoint: POST /api/v1/inventory/products/{id}/barcodes
func (h *BarcodeHandler) Add(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID produk wajib diisi"})
		return
	}

	var req AddBarcodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "body request tidak valid: " + err.Error()})
		return
	}

	cmd := application.AddBarcodeCommand{
		ProductID: productID,
		Barcode:   req.Barcode,
		IsPrimary: req.IsPrimary,
	}

	barcode, err := h.addBarcodeUC.Execute(r.Context(), cmd)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrProductNotFound):
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: err.Error()})
		case errors.Is(err, domain.ErrDuplicateBarcode):
			writeJSON(w, http.StatusConflict, ErrorResponse{Error: err.Error()})
		case errors.Is(err, domain.ErrInvalidBarcodeCode),
			errors.Is(err, domain.ErrBarcodeHasInvalidChar),
			errors.Is(err, domain.ErrInvalidProductID):
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		default:
			h.logger.Error("gagal menambahkan barcode", "error", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, toBarcodeResponse(barcode))
}

// Delete menangani penghapusan barcode dari produk.
// Endpoint: DELETE /api/v1/inventory/products/{id}/barcodes/{barcode_id}
func (h *BarcodeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	barcodeID := r.PathValue("barcode_id")
	if barcodeID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter barcode_id wajib diisi"})
		return
	}

	if err := h.deleteBarcodeUC.Execute(r.Context(), barcodeID); err != nil {
		if errors.Is(err, domain.ErrBarcodeNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		h.logger.Error("gagal menghapus barcode", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal menghapus barcode"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "barcode berhasil dihapus"})
}

// ListByProduct mengambil seluruh barcode yang terdaftar pada suatu produk.
// Endpoint: GET /api/v1/inventory/products/{id}/barcodes
func (h *BarcodeHandler) ListByProduct(w http.ResponseWriter, r *http.Request) {
	productID := r.PathValue("id")
	if productID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter ID produk wajib diisi"})
		return
	}

	barcodes, err := h.listBarcodesUC.Execute(r.Context(), productID)
	if err != nil {
		if errors.Is(err, application.ErrProductNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		h.logger.Error("gagal mengambil daftar barcode produk", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "gagal mengambil daftar barcode"})
		return
	}

	resp := make([]*BarcodeResponse, 0, len(barcodes))
	for _, b := range barcodes {
		resp = append(resp, toBarcodeResponse(b))
	}

	writeJSON(w, http.StatusOK, resp)
}

// Lookup menangani scan scanner barcode kasir (mencari produk berdasarkan string barcode).
// Endpoint: GET /api/v1/inventory/barcodes/lookup?code=8806091234567
func (h *BarcodeHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "parameter query 'code' wajib diisi"})
		return
	}

	product, barcode, err := h.lookupBarcodeUC.Execute(r.Context(), code)
	if err != nil {
		if errors.Is(err, application.ErrProductNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "produk dengan barcode tersebut tidak ditemukan"})
			return
		}
		h.logger.Error("gagal lookup barcode", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "terjadi kesalahan internal server"})
		return
	}

	writeJSON(w, http.StatusOK, ProductLookupResponse{
		Product:        toProductResponse(product),
		ScannedBarcode: toBarcodeResponse(barcode),
	})
}

func toBarcodeResponse(b *domain.ProductBarcode) *BarcodeResponse {
	return &BarcodeResponse{
		ID:        b.ID,
		ProductID: b.ProductID,
		Barcode:   b.Barcode,
		IsPrimary: b.IsPrimary,
		CreatedAt: b.CreatedAt,
	}
}
