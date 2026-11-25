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

	writeSuccess(
		w,
		http.StatusCreated,
		"Registrasi berhasil",
		model.UserResponse(user),
		nil,
	)
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

	data := map[string]interface{}{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"user":         model.UserResponse(user),
	}

	writeSuccess(w, http.StatusOK, "Login berhasil", data, nil)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserContextKey).(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Token tidak valid", nil)
		return
	}

	writeSuccess(
		w,
		http.StatusOK,
		"Data user berhasil diambil",
		map[string]string{"id": userID},
		nil,
	)
}
