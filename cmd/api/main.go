package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"snipvault/internal/auth"
	"snipvault/internal/cleanup"
	"snipvault/internal/database"
	"snipvault/internal/httpapi"
	"snipvault/internal/mailer"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	_ = godotenv.Load()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		logger.Error("JWT_SECRET must be at least 32 characters")
		os.Exit(1)
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startupCtx, cancelStartup := context.WithTimeout(rootCtx, 10*time.Second)
	pool, err := pgxpool.New(startupCtx, databaseURL)
	if err != nil {
		logger.Error("database pool creation failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(startupCtx); err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	if err := database.Migrate(startupCtx, pool); err != nil {
		logger.Error("database migration failed", "error", err)
		os.Exit(1)
	}
	cancelStartup()

	pasteStore := database.NewPasteStore(pool)
	emailSender := mailer.NewBrevo(os.Getenv("BREVO_API_KEY"), os.Getenv("MAIL_FROM_EMAIL"), os.Getenv("MAIL_FROM_NAME"))
	authService := auth.NewService(database.NewUserStore(pool), jwtSecret, emailSender)
	handler := httpapi.NewHandler(pasteStore, authService)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	cleanupInterval := 5 * time.Minute
	if configured := os.Getenv("CLEANUP_INTERVAL"); configured != "" {
		if parsed, err := time.ParseDuration(configured); err == nil && parsed > 0 {
			cleanupInterval = parsed
		} else {
			logger.Warn("invalid CLEANUP_INTERVAL; using default", "value", configured, "default", cleanupInterval)
		}
	}
	go cleanup.Run(rootCtx, pasteStore, cleanupInterval, logger)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("SnipVault API started", "address", "http://localhost:"+port)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-rootCtx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server failed", "error", err)
		}
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		_ = server.Close()
	}
	logger.Info("SnipVault stopped")
}
