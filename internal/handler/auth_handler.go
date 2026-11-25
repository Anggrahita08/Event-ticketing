package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"project-ticketing-golang/internal/middleware"
	"project-ticketing-golang/internal/model"
	"project-ticketing-golang/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", nil)
		return
	}

	user, err := h.authService.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, model.ErrConflict) {
			writeError(w, http.StatusConflict, "CONFLICT", "Email sudah terdaftar", nil)
			return
		}
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	writeSuccess(w, http.StatusCreated, "Registrasi berhasil", user, nil)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", nil)
		return
	}

	token, user, err := h.authService.Login(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Email atau password salah", nil)
		return
	}

	resData := map[string]interface{}{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"user":         user,
	}
	writeSuccess(w, http.StatusOK, "Login berhasil", resData, nil)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserContextKey).(string)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token tidak valid", nil)
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Gagal mengambil data user", nil)
		return
	}

	writeSuccess(w, http.StatusOK, "Data user berhasil diambil", user, nil)
}

// --- HELPER ENVELOPE RESPONSE ---
func writeSuccess(w http.ResponseWriter, status int, message string, data interface{}, meta interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(model.SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func writeError(w http.ResponseWriter, status int, code string, message string, details []model.ErrorDetail) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	var res model.ErrorResponse
	res.Success = false
	res.Message = message
	res.Error.Code = code
	res.Error.Details = details
	if res.Error.Details == nil {
		res.Error.Details = []model.ErrorDetail{}
	}
	json.NewEncoder(w).Encode(res)
}
