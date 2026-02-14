package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"audiobookshelf-asmr-provider/internal/service"
)

// Handler handles HTTP requests for the metadata provider API.
type Handler struct {
	service *service.Service
}

// NewHandler creates a new Handler with the given service.
func NewHandler(service *service.Service) *Handler {
	return &Handler{service: service}
}

// extractQuery reads the search query from the request, trying "q" first then "query".
func extractQuery(r *http.Request) string {
	if q := r.URL.Query().Get("q"); q != "" {
		return q
	}
	return r.URL.Query().Get("query")
}

// Search handles aggregated search requests across all providers.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := extractQuery(r)
	if query == "" {
		slog.Warn("Search request missing query")
		http.Error(w, "missing query", http.StatusBadRequest)
		return
	}

	slog.Info("Handling search request", "query", query)

	results, err := h.service.Search(r.Context(), query)
	if err != nil {
		slog.Error("Search failed", "error", err, "query", query)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		slog.Error("Failed to encode search results", "error", err)
	}
}

// SearchSingle handles search requests for a specific provider.
func (h *Handler) SearchSingle(w http.ResponseWriter, r *http.Request, providerID string) {
	query := extractQuery(r)
	if query == "" {
		slog.Warn("SearchSingle request missing query", "provider", providerID)
		http.Error(w, "missing query", http.StatusBadRequest)
		return
	}

	slog.Info("Handling single provider search request", "provider", providerID, "query", query)

	results, err := h.service.SearchByProviderID(r.Context(), providerID, query)
	if err != nil {
		slog.Error("Single provider search failed", "provider", providerID, "error", err, "query", query)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		slog.Error("Failed to encode single provider search results", "error", err, "provider", providerID)
	}
}
