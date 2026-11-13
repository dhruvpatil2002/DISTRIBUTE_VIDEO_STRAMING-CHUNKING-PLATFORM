
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/config"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/database"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/middleware"
	"github.com/dhruvpatil2002/distribute-video-streaming-chunking-platform/internal/routes"
)

func main() {

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	log.Printf("Starting application in %s mode", cfg.AppEnv)

	// Create startup context
	startupCtx, cancel := context.WithTimeout(
		context.Background(),
		15*time.Second,
	)
	defer cancel()

	// Initialize PostgreSQL
	db, err := database.NewPostgres(
		startupCtx,
		cfg.DatabaseURL,
		cfg.DBMaxConns,
		cfg.DBMinConns,
	)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(startupCtx, db); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	// Register routes
	handler, shutdownStreams, err := routes.New(db, cfg)
if err != nil {
    log.Fatalf("Route initialization failed: %v", err)
}
defer shutdownStreams()

	// Add middleware
	handler = middleware.RequestLogger(handler)

	// Configure HTTP server
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Listen for shutdown signals
	shutdownCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverError := make(chan error, 1)

	go func() {
		log.Printf("API server running at http://localhost%s", cfg.HTTPAddr)

		serverError <- server.ListenAndServe()
	}()

	select {

	case err := <-serverError:

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server failed: %v", err)
		}

	case <-shutdownCtx.Done():

		log.Println("Shutdown signal received")

		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
			_ = server.Close()
		}

		log.Println("Server stopped")
	}
}
