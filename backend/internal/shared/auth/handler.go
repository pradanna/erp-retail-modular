package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Handler menangani request HTTP untuk autentikasi dan manajemen staf.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler membuat instance baru HTTP Handler untuk auth.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes mendaftarkan seluruh endpoint autentikasi dan pengguna ke router http.ServeMux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, authMiddleware func(http.Handler) http.Handler) {
	// Endpoint Publik
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)

	// Endpoint Terproteksi Token
	mux.Handle("GET /api/v1/auth/me", authMiddleware(http.HandlerFunc(h.Me)))
	mux.Handle("POST /api/v1/auth/verify-password", authMiddleware(http.HandlerFunc(h.VerifyPassword)))
	mux.Handle("POST /api/v1/auth/change-password", authMiddleware(http.HandlerFunc(h.ChangePassword)))

	// Endpoint Manajemen Pengguna (Staf)
	mux.Handle("GET /api/v1/users", authMiddleware(http.HandlerFunc(h.ListUsers)))
	mux.Handle("POST /api/v1/users", authMiddleware(http.HandlerFunc(h.CreateUser)))
	mux.Handle("GET /api/v1/users/{id}", authMiddleware(http.HandlerFunc(h.GetUser)))
	mux.Handle("PATCH /api/v1/users/{id}/status", authMiddleware(http.HandlerFunc(h.SetUserStatus)))
}

// LoginRequest request body untuk login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserResponse representasi data publik pengguna.
type UserResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Username   string    `json:"username"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
	LocationID *string   `json:"location_id,omitempty"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// LoginResponse response setelah login berhasil.
type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

// VerifyPasswordRequest request body untuk Step-up Authentication.
type VerifyPasswordRequest struct {
	Password string `json:"password"`
}

// VerifyPasswordResponse response Step-up Authentication.
type VerifyPasswordResponse struct {
	Verified bool `json:"verified"`
}

// CreateUserRequest request body untuk membuat user baru.
type CreateUserRequest struct {
	Name       string  `json:"name"`
	Username   string  `json:"username"`
	Email      string  `json:"email"`
	Password   string  `json:"password"`
	Role       string  `json:"role"`
	LocationID *string `json:"location_id,omitempty"`
}

// ChangePasswordRequest request body untuk ganti password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// SetUserStatusRequest request body untuk toggle status user.
type SetUserStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// Login menangani autentikasi username/email dan password.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "body request tidak valid")
		return
	}

	token, user, err := h.service.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrWrongPassword):
			h.writeError(w, http.StatusUnauthorized, "username atau kata sandi tidak sesuai")
		case errors.Is(err, ErrUserInactive):
			h.writeError(w, http.StatusForbidden, "akun ini telah dinonaktifkan")
		default:
			h.logger.Error("gagal proses login", "error", err)
			h.writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, LoginResponse{
		Token: token,
		User:  toUserResponse(user),
	})
}

// Me mengembalikan profil pengguna yang sedang login berdasarkan JWT token.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil || claims.UserID == "" {
		h.writeError(w, http.StatusUnauthorized, "token tidak valid")
		return
	}

	user, err := h.service.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			h.writeError(w, http.StatusNotFound, "pengguna tidak ditemukan")
			return
		}
		h.logger.Error("gagal mengambil profil", "error", err)
		h.writeError(w, http.StatusInternalServerError, "gagal mengambil profil")
		return
	}

	h.writeJSON(w, http.StatusOK, toUserResponse(user))
}

// VerifyPassword menangani Step-up Authentication verifikasi ulang password.
func (h *Handler) VerifyPassword(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil || claims.UserID == "" {
		h.writeError(w, http.StatusUnauthorized, "token tidak valid")
		return
	}

	var req VerifyPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "body request tidak valid")
		return
	}

	verified, err := h.service.VerifyPassword(r.Context(), claims.UserID, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrWrongPassword):
			h.writeError(w, http.StatusUnauthorized, "kata sandi tidak sesuai")
		case errors.Is(err, ErrPasswordRequired):
			h.writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrUserInactive):
			h.writeError(w, http.StatusForbidden, err.Error())
		default:
			h.logger.Error("gagal verifikasi step-up auth", "error", err)
			h.writeError(w, http.StatusInternalServerError, "gagal verifikasi kata sandi")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, VerifyPasswordResponse{Verified: verified})
}

// ChangePassword menangani penggantian kata sandi mandiri pengguna.
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil || claims.UserID == "" {
		h.writeError(w, http.StatusUnauthorized, "token tidak valid")
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "body request tidak valid")
		return
	}

	if err := h.service.ChangePassword(r.Context(), claims.UserID, req.OldPassword, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, ErrWrongPassword):
			h.writeError(w, http.StatusUnauthorized, "kata sandi lama tidak sesuai")
		case errors.Is(err, ErrInvalidPassword):
			h.writeError(w, http.StatusBadRequest, err.Error())
		default:
			h.logger.Error("gagal ganti kata sandi", "error", err)
			h.writeError(w, http.StatusInternalServerError, "gagal mengganti kata sandi")
		}
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "kata sandi berhasil diperbarui"})
}

// ListUsers menampilkan daftar staf pengguna (hanya superadmin dan owner).
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil || (claims.Role != string(UserRoleSuperadmin) && claims.Role != string(UserRoleOwner)) {
		h.writeError(w, http.StatusForbidden, "hanya superadmin atau owner yang dapat melihat daftar pengguna")
		return
	}

	var roleFilter *UserRole
	if rParam := r.URL.Query().Get("role"); rParam != "" {
		rVal := UserRole(rParam)
		if rVal.IsValid() {
			roleFilter = &rVal
		}
	}

	var locFilter *string
	if locParam := r.URL.Query().Get("location_id"); locParam != "" {
		locFilter = &locParam
	}

	isActiveOnly := r.URL.Query().Get("active_only") == "true"

	users, err := h.service.ListUsers(r.Context(), roleFilter, locFilter, isActiveOnly)
	if err != nil {
		h.logger.Error("gagal list pengguna", "error", err)
		h.writeError(w, http.StatusInternalServerError, "gagal mengambil daftar pengguna")
		return
	}

	resp := make([]UserResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, toUserResponse(u))
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// CreateUser mendaftarkan staf/pengguna baru ke sistem.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil || (claims.Role != string(UserRoleSuperadmin) && claims.Role != string(UserRoleOwner)) {
		h.writeError(w, http.StatusForbidden, "hanya superadmin atau owner yang dapat mendaftarkan staf baru")
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "body request tidak valid")
		return
	}

	role := UserRole(req.Role)
	user, err := h.service.CreateUser(r.Context(), req.Name, req.Username, req.Email, req.Password, role, req.LocationID)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidName),
			errors.Is(err, ErrInvalidUsername),
			errors.Is(err, ErrInvalidEmail),
			errors.Is(err, ErrInvalidPassword),
			errors.Is(err, ErrInvalidRole):
			h.writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrUsernameExists),
			errors.Is(err, ErrEmailExists):
			h.writeError(w, http.StatusConflict, err.Error())
		default:
			h.logger.Error("gagal membuat staf baru", "error", err)
			h.writeError(w, http.StatusInternalServerError, "gagal membuat staf baru")
		}
		return
	}

	h.writeJSON(w, http.StatusCreated, toUserResponse(user))
}

// GetUser mengambil detail 1 staf berdasarkan ID.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil || (claims.Role != string(UserRoleSuperadmin) && claims.Role != string(UserRoleOwner)) {
		h.writeError(w, http.StatusForbidden, "hanya superadmin atau owner yang dapat melihat detail staf")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "parameter ID wajib diisi")
		return
	}

	user, err := h.service.GetProfile(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			h.writeError(w, http.StatusNotFound, "pengguna tidak ditemukan")
			return
		}
		h.logger.Error("gagal mengambil detail staf", "error", err)
		h.writeError(w, http.StatusInternalServerError, "gagal mengambil detail staf")
		return
	}

	h.writeJSON(w, http.StatusOK, toUserResponse(user))
}

// SetUserStatus mengubah status aktif/nonaktif staf.
func (h *Handler) SetUserStatus(w http.ResponseWriter, r *http.Request) {
	claims := GetClaims(r)
	if claims == nil || (claims.Role != string(UserRoleSuperadmin) && claims.Role != string(UserRoleOwner)) {
		h.writeError(w, http.StatusForbidden, "hanya superadmin atau owner yang dapat mengubah status staf")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "parameter ID wajib diisi")
		return
	}

	var req SetUserStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "body request tidak valid")
		return
	}

	user, err := h.service.SetUserStatus(r.Context(), id, req.IsActive)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			h.writeError(w, http.StatusNotFound, "pengguna tidak ditemukan")
			return
		}
		h.logger.Error("gagal ubah status staf", "error", err)
		h.writeError(w, http.StatusInternalServerError, "gagal mengubah status staf")
		return
	}

	h.writeJSON(w, http.StatusOK, toUserResponse(user))
}

func toUserResponse(u *User) UserResponse {
	return UserResponse{
		ID:         u.ID,
		Name:       u.Name,
		Username:   u.Username,
		Email:      u.Email,
		Role:       string(u.Role),
		LocationID: u.LocationID,
		IsActive:   u.IsActive,
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
