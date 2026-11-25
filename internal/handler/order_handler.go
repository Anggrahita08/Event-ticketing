package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

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

func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserContextKey).(string)
	role, _ := r.Context().Value(middleware.UserRoleKey).(string)
	isAdmin := role == "admin"

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	orders, totalItems, err := h.orderService.GetOrders(r.Context(), userID, isAdmin, q.Get("status"), q.Get("event_id"), page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	if limit < 1 {
		limit = 10
	}
	if page < 1 {
		page = 1
	}
	totalPages := (totalItems + limit - 1) / limit
	meta := map[string]interface{}{"page": page, "limit": limit, "total_items": totalItems, "total_pages": totalPages}
	writeSuccess(w, http.StatusOK, "Daftar order berhasil diambil", orders, meta)
}

func (h *OrderHandler) Detail(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserContextKey).(string)
	role, _ := r.Context().Value(middleware.UserRoleKey).(string)
	isAdmin := role == "admin"
	orderID := r.PathValue("id")

	order, err := h.orderService.GetOrderByID(r.Context(), userID, isAdmin, orderID)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Order tidak ditemukan", nil)
		return
	}
	writeSuccess(w, http.StatusOK, "Detail order berhasil diambil", order, nil)
}
