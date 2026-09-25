package audit

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/erp-retail/backend/internal/shared/auth"
)

// Handler menangani endpoint HTTP untuk membaca riwayat log audit sistem.
type Handler struct {
	service     *Service
	permService auth.PermissionService
	log         *slog.Logger
}

// NewHandler membuat instance Handler baru.
func NewHandler(service *Service, permService auth.PermissionService, log *slog.Logger) *Handler {
	return &Handler{
		service:     service,
		permService: permService,
		log:         log,
	}
}

// RegisterRoutes mendaftarkan endpoint audit log ke router utama.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	mux.Handle("GET /api/shared/audit-logs", authMiddleware(http.HandlerFunc(h.listAuditLogs)))
}

func (h *Handler) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		h.errorJSON(w, http.StatusUnauthorized, "Token autentikasi tidak valid")
		return
	}

	// Hanya role owner/admin atau yang memiliki izin audit yang boleh melihat log
	if claims.Role != "owner" && claims.Role != "superadmin" && claims.Role != "admin" {
		h.errorJSON(w, http.StatusForbidden, "Akses ditolak: Hanya pemilik toko/admin yang dapat melihat log audit")
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	filter := AuditFilter{
		Module: q.Get("module"),
		Action: q.Get("action"),
		UserID: q.Get("user_id"),
		Page:   page,
		Limit:  limit,
	}

	if startStr := q.Get("start_date"); startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			filter.StartDate = &t
		}
	}
	if endStr := q.Get("end_date"); endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			// Set ke akhir hari (23:59:59)
			endOfDay := t.Add(24*time.Hour - time.Second)
			filter.EndDate = &endOfDay
		}
	}

	logs, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		h.log.Error("gagal mengambil audit logs", "error", err)
		h.errorJSON(w, http.StatusInternalServerError, "Gagal mengambil log audit: "+err.Error())
		return
	}

	totalPages := (total + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"data": logs,
		"meta": map[string]any{
			"page":        page,
			"limit":       limit,
			"total_items": total,
			"total_pages": totalPages,
		},
	})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) errorJSON(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, map[string]string{"error": msg})
}
