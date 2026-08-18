package handler

import (
	"encoding/json"
	"net/http"

	"TestTask_Bazis/internal/handler/middleware"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func getUserID(r *http.Request) uint64 {
	return middleware.GetUserID(r.Context())
}
