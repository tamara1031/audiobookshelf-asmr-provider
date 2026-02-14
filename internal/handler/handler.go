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

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		query = r.URL.Query().Get("query")
	}

	if query == "" {
		http.Error(w, "query parameter 'q' or 'query' is required", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Search(r.Context(), query)
	if err != nil {
		slog.Error("Search failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) SearchSingle(w http.ResponseWriter, r *http.Request, providerID string) {
	query := r.URL.Query().Get("q")
	if query == "" {
		query = r.URL.Query().Get("query")
	}

	if query == "" {
		http.Error(w, "query parameter 'q' or 'query' is required", http.StatusBadRequest)
		return
	}

	resp, err := h.service.SearchByProviderID(r.Context(), providerID, query)
	if err != nil {
		slog.Error("Search failed", "provider", providerID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
