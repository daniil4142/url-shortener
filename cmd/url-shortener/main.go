package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/daniil4142/url-shortener/internal/config"
	"github.com/daniil4142/url-shortener/internal/http-server/handlers/redirect"
	"github.com/daniil4142/url-shortener/internal/http-server/handlers/url/save"
	"github.com/daniil4142/url-shortener/internal/lib/logger/sl"
	"github.com/daniil4142/url-shortener/internal/storage/sqlite"
	"github.com/joho/godotenv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/exp/slog"
)

func main() {
	godotenv.Load()

	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("starting service", slog.String("env", cfg.Env))
	log.Debug("debug are enabled")

	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	// router.Use(mwLogger.New(log))
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/{alias}", redirect.New(log, storage))
	router.Post("/url", save.New(log, storage))

	log.Info("starting server", slog.String("address", cfg.Address))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Error("failed to start server")
		}
	}()

	log.Info("server started")

	<-done
	log.Info("stopping server")

}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case "local":
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case "dev":
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case "prod":
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}
	return log
}
