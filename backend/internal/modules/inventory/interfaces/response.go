package interfaces

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

// writeJSON menulis JSON response dengan status code yang diberikan ke http.ResponseWriter.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("gagal encode JSON response", "error", err)
	}
}

// parseIntParam memparsing string query parameter menjadi integer positif dengan nilai default fallback.
func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return defaultVal
	}
	return v
}
