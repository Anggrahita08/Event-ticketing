package handler

import (
	"encoding/json"
	"net/http"

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
