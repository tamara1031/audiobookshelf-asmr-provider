package http

import (
	"encoding/json"
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
		http.Error(w, "missing query", http.StatusBadRequest)
		return
	}

	results, err := h.service.Search(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}

// SearchSingle handles search requests for a specific provider.
func (h *Handler) SearchSingle(w http.ResponseWriter, r *http.Request, providerID string) {
	query := extractQuery(r)
	if query == "" {
		http.Error(w, "missing query", http.StatusBadRequest)
		return
	}

	results, err := h.service.SearchByProviderID(r.Context(), providerID, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}
