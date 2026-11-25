package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"project-ticketing-golang/internal/middleware"
	"project-ticketing-golang/internal/model"
	"project-ticketing-golang/internal/service"
)

type EventHandler struct {
	eventService *service.EventService
}

func NewEventHandler(es *service.EventService) *EventHandler {
	return &EventHandler{eventService: es}
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	adminID, _ := r.Context().Value(middleware.UserContextKey).(string)
	var req model.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", nil)
		return
	}

	event, err := h.eventService.CreateEvent(r.Context(), adminID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	writeSuccess(w, http.StatusCreated, "Event berhasil dibuat", event, nil)
}

func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	adminID, _ := r.Context().Value(middleware.UserContextKey).(string)
	id := r.PathValue("id")
	var req model.UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", nil)
		return
	}

	event, err := h.eventService.UpdateEvent(r.Context(), adminID, id, req)
	if err != nil {
		if err.Error() == "VERSION_CONFLICT" {
			writeError(w, http.StatusConflict, "VERSION_CONFLICT", "Event sudah diubah oleh pengguna lain, muat ulang data terbaru", nil)
			return
		}
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	writeSuccess(w, http.StatusOK, "Event berhasil diperbarui", event, nil)
}

func (h *EventHandler) Publish(w http.ResponseWriter, r *http.Request) {
	adminID, _ := r.Context().Value(middleware.UserContextKey).(string)
	id := r.PathValue("id")

	event, err := h.eventService.PublishEvent(r.Context(), adminID, id)
	if err != nil {
		writeError(w, http.StatusConflict, "INVALID_STATE", err.Error(), nil)
		return
	}
	writeSuccess(w, http.StatusOK, "Event berhasil diterbitkan", event, nil)
}

func (h *EventHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	adminID, _ := r.Context().Value(middleware.UserContextKey).(string)
	id := r.PathValue("id")
	var req model.CancelEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Input tidak valid", nil)
		return
	}

	event, cancelledOrders, err := h.eventService.CancelEvent(r.Context(), adminID, id, req)
	if err != nil {
		if err.Error() == "INVALID_STATE" {
			writeError(w, http.StatusConflict, "INVALID_STATE", "Event sudah dibatalkan", nil)
			return
		}
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	// Custom response untuk cancel sesuai brief
	responseData := map[string]interface{}{
		"event":            event,
		"cancelled_orders": cancelledOrders,
	}
	writeSuccess(w, http.StatusOK, "Event berhasil dibatalkan", responseData, nil)
}

// Ganti fungsi ListPublic dan GetDetail di event_handler.go dengan ini:

func (h *EventHandler) ListPublic(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "limit maksimal 100", nil)
		return
	}

	sortParam := query.Get("sort")
	if sortParam != "" && sortParam != "start_at" && sortParam != "price" && sortParam != "created_at" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "sort tidak valid", nil)
		return
	}
	orderParam := query.Get("order")
	if orderParam != "" && orderParam != "asc" && orderParam != "desc" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "order tidak valid", nil)
		return
	}

	filter := map[string]interface{}{
		"search":     query.Get("search"),
		"status":     "published",
		"start_from": query.Get("start_from"),
		"start_to":   query.Get("start_to"),
		"sort":       sortParam,
		"order":      orderParam,
	}

	events, totalItems, err := h.eventService.GetEvents(r.Context(), "public", filter, page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}

	totalPages := (totalItems + limit - 1) / limit
	meta := map[string]interface{}{
		"page":        page,
		"limit":       limit,
		"total_items": totalItems,
		"total_pages": totalPages,
	}
	writeSuccess(w, http.StatusOK, "Daftar event berhasil diambil", events, meta)
}

func (h *EventHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	role := "public" // Default
	id := r.PathValue("id")

	event, err := h.eventService.GetEventDetail(r.Context(), role, id)
	if err != nil {
		if err == model.ErrNotFound {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "Event tidak ditemukan", nil)
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		return
	}
	writeSuccess(w, http.StatusOK, "Detail event berhasil diambil", event, nil)
}
