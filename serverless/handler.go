package serverless

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"snipvault/internal/auth"
	"snipvault/internal/database"
	"snipvault/internal/httpapi"
	"snipvault/internal/mailer"
)

var (
	initialize sync.Once
	app        http.Handler
	startupErr error
)

// Handler initializes one app instance per warm function and serves the request.
func Handler(w http.ResponseWriter, r *http.Request) {
	initialize.Do(func() {
		startupErr = initializeApp()
	})
	if startupErr != nil {
		http.Error(w, "service initialization failed", http.StatusServiceUnavailable)
		return
	}
	app.ServeHTTP(w, r)
}

func initializeApp() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("parse database configuration: %w", err)
	}
	config.MaxConns = 2
	config.MinConns = 0
	config.MaxConnIdleTime = 30 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("connect to database: %w", err)
	}
	if err := database.Migrate(ctx, pool); err != nil {
		pool.Close()
		return fmt.Errorf("migrate database: %w", err)
	}

	pasteStore := database.NewPasteStore(pool)
	emailSender := mailer.NewBrevo(os.Getenv("BREVO_API_KEY"), os.Getenv("MAIL_FROM_EMAIL"), os.Getenv("MAIL_FROM_NAME"))
	authService := auth.NewService(database.NewUserStore(pool), jwtSecret, emailSender)
	app = httpapi.NewHandler(pasteStore, authService)
	return nil
}
