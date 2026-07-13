package handlers

import (
	"encoding/json"
	"net/http"
)

// APIResponse est la structure d'enveloppe stable demandée
type APIResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data,omitempty"`
	Error   any  `json:"error,omitempty"`
	Meta    any  `json:"meta,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any, meta any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{Success: true, Data: data, Meta: meta})
}

func JSONError(w http.ResponseWriter, status int, err any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err})
}
