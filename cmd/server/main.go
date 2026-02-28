package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hls-orchestrator/internal/orchestrator"
	"hls-orchestrator/internal/platform/config"
	"hls-orchestrator/internal/platform/logger"
	"hls-orchestrator/internal/platform/metrics"

	"github.com/go-chi/chi/v5"
)

const shutdownTimeout = 10 * time.Second

func main() {
	_ = config.Load()

	port := config.GetEnv("PORT", "8080")
	windowSize := config.GetEnvInt("SLIDING_WINDOW_SIZE", 6)
	logLevel := config.GetEnv("LOG_LEVEL", "info")
	logFormat := config.GetEnv("LOG_FORMAT", "json")

	log := logger.New(logLevel, logFormat)

	repo := orchestrator.NewInMemoryRepository()
	svc := orchestrator.NewService(repo, windowSize)
	met := metrics.New()
	h := orchestrator.NewHandler(svc, log, met)

	r := chi.NewRouter()
	r.Use(logger.RequestLogger(log))
	r.Use(metrics.RequestMiddleware(met))
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		met.Handler(func() { met.SetActiveStreams(repo.ActiveStreamCount()) }).ServeHTTP(w, r)
	})
	r.Route("/streams/{stream_id}", func(r chi.Router) {
		r.Post("/end", h.EndStream)
		r.Route("/renditions/{rendition}", func(r chi.Router) {
			r.Post("/segments", h.RegisterSegment)
			r.Get("/playlist.m3u8", h.GetPlaylist)
		})
	})

	addr := ":" + port
	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	log.Info("server starting",
		"port", port,
		"sliding_window_size", windowSize,
		"log_level", logLevel,
	)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Info("shutdown signal received, draining connections")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown error", "error", err)
		os.Exit(1)
	}

	log.Info("server stopped")
}
