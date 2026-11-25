package handler

import (
	"net/http"
	"strconv"

	"project-ticketing-golang/internal/repository"
)

type AuditLogHandler struct {
	repo repository.AuditLogRepository
}

func NewAuditLogHandler(repo repository.AuditLogRepository) *AuditLogHandler {
	return &AuditLogHandler{repo: repo}
}

func (h *AuditLogHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	logs, totalItems, err := h.repo.GetLogs(r.Context(), q.Get("entity_type"), q.Get("entity_id"), page, limit)
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
	writeSuccess(w, http.StatusOK, "Daftar audit log berhasil diambil", logs, meta)
}
