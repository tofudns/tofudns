package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/exp/slog"

	"github.com/tofudns/tofudns/internal/frontend"
)

type Config struct {
	Port     string `envconfig:"PORT" default:"8080"`
	LogLevel string `envconfig:"LOG_LEVEL" default:"debug"`
}

func main() {
	// Load config
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Initialize logger with configured log level
	logLevel := slog.LevelInfo
	switch config.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: logLevel,
			},
		),
	)
	slog.SetDefault(logger)

	// Create a new Chi router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Construct frontend service
	frontendService, err := frontend.New()
	if err != nil {
		logger.Error("Failed to create frontend service", "error", err)
		os.Exit(1)
	}

	// Mount the frontend handler
	r.Mount("/", frontendService)

	// Set up the server
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: r,
	}

	// Create listeners
	listener, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		logger.Error("Failed to create listener", "error", err)
		os.Exit(1)
	}

	// Start the server
	go func() {
		logger.Info("Starting server", "address", srv.Addr)
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	// Shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown server", "error", err)
		os.Exit(1)
	}

	logger.Info("Server shutdown")
}
