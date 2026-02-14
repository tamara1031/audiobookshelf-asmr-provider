package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"audiobookshelf-asmr-provider/internal/config"
	"audiobookshelf-asmr-provider/internal/domain/cache"
	"audiobookshelf-asmr-provider/internal/domain/provider"
	"audiobookshelf-asmr-provider/internal/handler"
	"audiobookshelf-asmr-provider/internal/service"
)

func main() {
	// Initialize structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	providers := provider.NewAll()
	slog.Info("Loaded providers", "count", len(providers))

	memCache := cache.NewMemoryCache()
	svc := service.NewService(memCache, providers...)
	h := handler.NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/search", h.Search)

	for _, p := range svc.Providers() {
		providerID := p.ID()
		path := "/api/" + providerID + "/search"
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			h.SearchSingle(w, r, providerID)
		})
		slog.Debug("Registered provider endpoint", "path", path)
	}

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Starting server", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server exiting")
}
