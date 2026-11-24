package handler

import (
	"encoding/json"
	"net/http"

	"project-ticketing-golang/internal/middleware"
	"project-ticketing-golang/internal/model"
	"project-ticketing-golang/internal/service"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(os *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: os}
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserContextKey).(string)

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" || len(idempotencyKey) > 100 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Header Idempotency-Key wajib diisi maksimal 100 karakter", nil)
		return
	}

	var req model.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", nil)
		return
	}

	order, isReplay, err := h.orderService.CreateOrder(r.Context(), userID, idempotencyKey, req)
	if err != nil {
		switch err.Error() {
		case "IDEMPOTENCY_KEY_REUSED":
			writeError(w, http.StatusUnprocessableEntity, "IDEMPOTENCY_KEY_REUSED", "Idempotency-Key sudah dipakai untuk request yang berbeda", nil)
		case "TICKET_LIMIT_EXCEEDED":
			writeError(w, http.StatusConflict, "TICKET_LIMIT_EXCEEDED", "Total tiket customer untuk event tersebut akan melebihi 5", nil)
		case "INSUFFICIENT_QUOTA":
			writeError(w, http.StatusConflict, "INSUFFICIENT_QUOTA", "Sisa tiket tidak mencukupi", nil)
		case "INVALID_STATE":
			writeError(w, http.StatusConflict, "INVALID_STATE", "Event tidak valid atau sudah dimulai", nil)
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		}
		return
	}

	if isReplay {
		w.Header().Set("Idempotent-Replayed", "true")
	}
	writeSuccess(w, http.StatusCreated, "Order berhasil dibuat, selesaikan pembayaran sebelum batas waktu", order, nil)
}

func (h *OrderHandler) Pay(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserContextKey).(string)
	orderID := r.PathValue("id")

	order, err := h.orderService.PayOrder(r.Context(), userID, orderID)
	if err != nil {
		if err == model.ErrNotFound {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Order tidak ditemukan", nil)
			return
		}
		writeError(w, http.StatusConflict, "INVALID_STATE", "Order tidak dapat dibayar karena status tidak valid atau sudah expired", nil)
		return
	}
	writeSuccess(w, http.StatusOK, "Pembayaran berhasil", order, nil)
}

func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserContextKey).(string)
	orderID := r.PathValue("id")

	order, err := h.orderService.CancelOrder(r.Context(), userID, orderID)
	if err != nil {
		if err == model.ErrNotFound {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Order tidak ditemukan", nil)
			return
		}
		writeError(w, http.StatusConflict, "INVALID_STATE", "Order tidak dapat dibatalkan karena status tidak valid atau event sudah dimulai", nil)
		return
	}
	writeSuccess(w, http.StatusOK, "Order berhasil dibatalkan", order, nil)
}

// func (h *EventHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
// 	writeSuccess(w, http.StatusOK, "Daftar event berhasil diambil", []interface{}{}, nil)
// }

// func (h *EventHandler) GetDetail(w http.ResponseWriter, r *http.Request) {

// 	writeSuccess(w, http.StatusOK, "Detail event berhasil diambil", map[string]string{}, nil)
// }
