package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"audiobookshelf-asmr-provider/internal/service"
)

type Handler struct {
	service *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{
		service: svc,
	}
}

// SearchAll handles searches across all providers.
func (h *Handler) SearchAll(w http.ResponseWriter, r *http.Request) {
	h._Search(w, r, "all")
}

// Search handles searches for a specific provider extracted from path values.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	h._Search(w, r, r.PathValue("provider"))
}

// _Search is a shared helper for executing searches.
func (h *Handler) _Search(w http.ResponseWriter, r *http.Request, providerID string) {
	if providerID == "" {
		providerID = "all"
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		query = r.URL.Query().Get("query")
	}

	if query == "" {
		http.Error(w, "query parameter 'q' or 'query' is required", http.StatusBadRequest)
		return
	}

	slog.Debug("Search request", "provider", providerID, "query", query, "url_params", r.URL.Query())

	resp, err := h.service.SearchByProviderID(r.Context(), providerID, query)
	if err != nil {
		slog.Error("Search failed", "provider", providerID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("Search response", "provider", providerID, "response", resp)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
