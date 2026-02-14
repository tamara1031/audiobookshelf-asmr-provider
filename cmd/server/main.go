package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httphandler "audiobookshelf-asmr-provider/internal/adapter/http"
	"audiobookshelf-asmr-provider/internal/adapter/provider"
	"audiobookshelf-asmr-provider/internal/service"
)

func main() {
	providers := provider.NewAll()
	log.Printf("Loaded %d provider(s)", len(providers))

	svc := service.NewService(providers...)
	handler := httphandler.NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/search", handler.Search)

	for _, p := range svc.Providers() {
		providerID := p.ID()
		path := "/api/" + providerID + "/search"
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			handler.SearchSingle(w, r, providerID)
		})
		log.Printf("Registered provider endpoint: %s", path)
	}

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting server on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
