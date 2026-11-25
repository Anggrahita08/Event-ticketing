package handler

import (
	"encoding/json"
	"net/http"
)

func writeSuccess(
	w http.ResponseWriter,
	status int,
	message string,
	data interface{},
	meta interface{},
) {
	writeJSON(w, status, map[string]interface{}{
		"success": true,
		"message": message,
		"data":    data,
		"meta":    meta,
	})
}

func writeError(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
	details interface{},
) {
	errData := map[string]interface{}{
		"code":    code,
		"message": message,
	}

	if details != nil {
		errData["details"] = details
	}

	writeJSON(w, status, map[string]interface{}{
		"success": false,
		"message": message,
		"error":   errData,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
