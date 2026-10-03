package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/yannick2009/yily/internal/adapter"
	"github.com/yannick2009/yily/internal/config"
)

// requestTimeout is the maximum time a handler is allowed to run.
const requestTimeout = 30 * time.Second

// entry point for the server application.
func main() {
	// All the logic lives in run() so that deferred calls (closing the
	// database, etc.) are executed before the process exits. os.Exit skips
	// deferred functions, so it is only called here, after run returns.
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration settings. The server refuses to start with an
	// invalid configuration rather than running in a broken state.
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	// Initialize the database connection.
	db, err := adapter.NewDB(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get database handle: %w", err)
	}

	// Ensure the database connection is closed cleanly when run() exits.
	// We handle and log any error to satisfy errcheck and keep track of shutdown failures.
	defer func() {
		if err := sqlDB.Close(); err != nil {
			slog.Error("failed to close database", "error", err)
		}
	}()

	// The context is cancelled on Ctrl+C (SIGINT) or SIGTERM (docker stop,
	// systemd), which triggers the graceful shutdown of the HTTP server.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	sc := echo.StartConfig{
		Address: cfg.Addr,
		// Time given to in-flight requests to finish after a shutdown signal.
		GracefulTimeout: 10 * time.Second,
		// Go's http.Server has no timeouts by default: a client could keep a
		// connection open forever by sending headers very slowly (Slowloris).
		BeforeServeFunc: func(s *http.Server) error {
			s.ReadHeaderTimeout = 5 * time.Second
			s.ReadTimeout = 15 * time.Second
			s.WriteTimeout = requestTimeout + 5*time.Second // longer than the handler timeout
			s.IdleTimeout = 60 * time.Second
			return nil
		},
	}
	slog.Info("starting server", "addr", cfg.Addr, "db", cfg.DBPath)
	if err := sc.Start(ctx, newServer(sqlDB)); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server: %w", err)
	}
	slog.Info("server stopped gracefully")
	return nil
}

func newServer(sqlDB *sql.DB) *echo.Echo {
	server := echo.New()
	// Middleware configuration
	server.Use(middleware.RequestLogger())                                         // Log incoming requests
	server.Use(middleware.Recover())                                               // Recover from panics and return a 500 error
	server.Use(middleware.BodyLimit(1 << 20))                                      // 1 MB
	server.Use(middleware.ContextTimeout(requestTimeout))                          // Set a timeout for request context
	server.Use(middleware.CSRF())                                                  // Enable CSRF protection
	server.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(50.0))) // Rate limiting 50 requests per second
	server.Use(middleware.Secure())                                                // Enable security headers
	// Define API routes
	routes := server.Group("/api/v1")
	// The healthcheck also pings the database, so that orchestrators
	// (Docker, Kubernetes) detect a server that runs but cannot work.
	routes.GET("/healthcheck", func(c *echo.Context) error {
		if err := sqlDB.PingContext(c.Request().Context()); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"message": "database unavailable"})
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "OK!"})
	})
	return server
}
