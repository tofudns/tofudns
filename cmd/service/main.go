package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "embed"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/kelseyhightower/envconfig"

	"github.com/tofudns/tofudns/internal/email"
	"github.com/tofudns/tofudns/internal/frontend"
	"github.com/tofudns/tofudns/internal/recordmanager"
	"github.com/tofudns/tofudns/internal/storage"
)

type Config struct {
	Port           string `envconfig:"PORT" default:"8080"`
	LogLevel       string `envconfig:"LOG_LEVEL" default:"debug"`
	DatabaseDriver string `envconfig:"DATABASE_DRIVER" default:"postgres"`
	DatabaseURL    string `envconfig:"DATABASE_URL" default:"postgres://tofudns:tofudns@localhost:5432/tofudns?sslmode=disable"`
	JWTSecret      string `envconfig:"JWT_SECRET" required:"true"`
	Postmark       struct {
		ServerToken string `envconfig:"POSTMARK_SERVER_TOKEN" required:"true"`
		FromEmail   string `envconfig:"POSTMARK_EMAIL_FROM" default:"noreply@tofudns.net"`
	}
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

	// Setup database client
	db, err := sql.Open(config.DatabaseDriver, config.DatabaseURL)
	if err != nil {
		logger.Error("Failed to create database client", "error", err)
		os.Exit(1)
	}

	// Run database migrations
	if err := runDatabaseMigrations(db); err != nil {
		logger.Error("Failed to run database migrations", "error", err)
		os.Exit(1)
	}

	// Construct the database client
	dbClient := storage.New(db)

	// Create the record manager
	records := recordmanager.New(dbClient)

	// Create a new Chi router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Create the email service
	emailService := email.NewPostmarkService(email.PostmarkConfig{
		ServerToken: config.Postmark.ServerToken,
		FromEmail:   config.Postmark.FromEmail,
	})

	// Create the frontend service
	frontendService, err := frontend.New(logger, records, dbClient, emailService, config.JWTSecret)
	if err != nil {
		logger.Error("Failed to create frontend service", "error", err)
		os.Exit(1)
	}

	// Route the frontend handler
	r.Route("/", frontendService.Router)

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

func runDatabaseMigrations(db *sql.DB) error {
	// Construct the database driver
	migrateDatabaseDriver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	// Construct the migration source
	migrateSourceDriver, err := iofs.New(storage.MigrationsFS, "migrations")
	if err != nil {
		return err
	}

	// Create migration instance
	migrationClient, err := migrate.NewWithInstance(
		"iofs",
		migrateSourceDriver,
		"postgres",
		migrateDatabaseDriver,
	)
	if err != nil {
		return err
	}

	// Run migrations
	err = migrationClient.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
