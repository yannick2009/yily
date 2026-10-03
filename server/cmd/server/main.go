package main

import (
	"context"
	"errors"
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

// entry point for the server application.
func main() {
	// Load configuration settings
	cfg, err := config.Load()
	if err != nil {
		echo.New().Logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}
	_ = cfg

	// Initialize the database connection
	db, err := adapter.NewDB()
	if err != nil {
		echo.New().Logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	_ = db

	// Create a new server instance
	server := echo.New()

	// Middleware configuration
	server.Use(middleware.RequestLogger())                                         // Log incoming requests
	server.Use(middleware.Recover())                                               // Recover from panics and return a 500 error
	server.Use(middleware.BodyLimit(1 << 20))                                      // 1 MB
	server.Use(middleware.ContextTimeout(30 * time.Second))                        // Set a timeout for request context
	server.Use(middleware.CSRF())                                                  // Enable CSRF protection
	server.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(50.0))) // Rate limiting 50 requests per second
	server.Use(middleware.Secure())                                                // Enable security headers

	// Define API routes
	routes := server.Group("/api/v1")

	routes.GET("/healthcheck", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "OK!"})
	})

	// Setup graceful shutdown on SIGINT or SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sc := echo.StartConfig{
		Address: ":16529",
	}

	if err := sc.Start(ctx, server); err != nil && !errors.Is(err, http.ErrServerClosed) {
		server.Logger.Error("failed to start server", "error", err)
	}
}
