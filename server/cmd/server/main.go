package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
	"github.com/yannick2009/yily/internal/adapter"
	"github.com/yannick2009/yily/internal/config"
	"github.com/yannick2009/yily/internal/handler"
	"github.com/yannick2009/yily/internal/middleware"
	"github.com/yannick2009/yily/internal/repository"
	"github.com/yannick2009/yily/internal/services"
)

// entry point for the server application.
func main() {
	// Load configuration settings
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	// Initialize the database connection
	db := adapter.NewDB()

	// Repositories
	userRepository := repository.NewUserRepository(db)
	sessionRepository := repository.NewSessionRepository(db)
	patRepository := repository.NewPATRepository(db)
	projectRepository := repository.NewProjectRepository(db)
	environmentRepository := repository.NewEnvironmentRepository(db)
	variableRepository := repository.NewVariableRepository(db)
	versionRepository := repository.NewVariableVersionRepository(db)
	tagRepository := repository.NewTagRepository(db)
	auditRepository := repository.NewAuditRepository(db)

	// Services
	auditService := services.NewAuditService(auditRepository)
	authService := services.NewAuthService(userRepository, sessionRepository)
	patService := services.NewPATService(userRepository, patRepository, auditService)
	projectService := services.NewProjectService(projectRepository, environmentRepository, auditService)
	environmentService := services.NewEnvironmentService(environmentRepository, auditService)
	variableService := services.NewVariableService(variableRepository, versionRepository, auditService, cfg.MasterKey)
	tagService := services.NewTagService(tagRepository, auditService)
	dashboardService := services.NewDashboardService(projectRepository, environmentRepository, variableRepository, patRepository, auditService)

	// Handlers
	authHandler := handler.NewAuthHandler(authService, auditService)
	patHandler := handler.NewPATHandler(patService)
	projectHandler := handler.NewProjectHandler(projectService)
	environmentHandler := handler.NewEnvironmentHandler(environmentService, variableService)
	variableHandler := handler.NewVariableHandler(variableService)
	tagHandler := handler.NewTagHandler(tagService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	auditHandler := handler.NewAuditHandler(auditService)

	// Create a new server instance
	server := echo.New()

	// Validator
	server.Validator = NewValidator()

	// Middleware configuration
	server.Use(echomw.RequestLogger())                                     // Log incoming requests
	server.Use(echomw.Recover())                                           // Recover from panics and return a 500 error
	server.Use(echomw.BodyLimit(1 << 20))                                  // 1 MB
	server.Use(echomw.ContextTimeout(30 * time.Second))                    // Set a timeout for request context
	server.Use(echomw.RateLimiter(echomw.NewRateLimiterMemoryStore(50.0))) // Rate limiting 50 requests per second
	server.Use(echomw.Secure())                                            // Enable security headers

	// Authentication middleware
	authMiddleware := middleware.RequireAuth(authService, patService)

	// Define API routes
	api := server.Group("/api/v1")

	// Public authentication routes
	api.POST("/auth/setup", authHandler.Setup)
	api.POST("/auth/login", authHandler.Login)

	// Protected API routes (require a valid server-side session or Bearer PAT)
	routes := api.Group("", authMiddleware)

	// Authentication and token management
	routes.POST("/auth/logout", authHandler.Logout)
	routes.GET("/auth/me", authHandler.Me)
	routes.POST("/tokens", patHandler.Create)
	routes.GET("/tokens", patHandler.GetAll)
	routes.DELETE("/tokens/:id", patHandler.Revoke)

	// Dashboard
	routes.GET("/dashboard", dashboardHandler.Get)

	// Projects
	routes.POST("/projects", projectHandler.Create)
	routes.GET("/projects", projectHandler.GetAll)
	routes.DELETE("/projects", projectHandler.Clear)
	routes.GET("/projects/:id", projectHandler.GetByID)
	routes.PATCH("/projects/:id", projectHandler.SetName)
	routes.DELETE("/projects/:id", projectHandler.Delete)

	// Environments
	routes.POST("/projects/:projectID/environments", environmentHandler.Create)
	routes.GET("/projects/:projectID/environments", environmentHandler.GetAll)
	routes.GET("/environments/:id", environmentHandler.GetByID)
	routes.PATCH("/environments/:id/name", environmentHandler.SetName)
	routes.PATCH("/environments/:id/description", environmentHandler.SetDescription)
	routes.DELETE("/environments/:id", environmentHandler.Delete)
	routes.POST("/environments/:id/imports", environmentHandler.Import)

	// Variables
	routes.POST("/environments/:environmentID/variables", variableHandler.Create)
	routes.GET("/environments/:environmentID/variables", variableHandler.List)
	routes.GET("/variables/:id", variableHandler.GetByID)
	routes.PATCH("/variables/:id", variableHandler.SetName)
	routes.DELETE("/variables/:id", variableHandler.Delete)
	routes.PUT("/variables/:id/tags", variableHandler.SetTags)

	// Variable versions
	routes.POST("/variables/:id/versions", variableHandler.AddVersion)
	routes.GET("/variables/:id/versions", variableHandler.GetAllVersions)
	routes.GET("/variables/:id/versions/current", variableHandler.GetCurrentVersion)
	routes.PATCH("/variables/:id/versions/:versionID", variableHandler.SetVersionActive)
	routes.DELETE("/variables/:id/value", variableHandler.ClearValue)

	// Tags
	routes.POST("/projects/:projectID/tags", tagHandler.Create)
	routes.GET("/projects/:projectID/tags", tagHandler.GetAll)
	routes.DELETE("/projects/:projectID/tags/:tagID", tagHandler.Delete)

	// Audit log
	routes.GET("/audit", auditHandler.List)

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

// CustomValidator wraps go-playground/validator and implements echo.Validator.
type CustomValidator struct {
	validator *validator.Validate
}

// NewValidator creates a new CustomValidator.
func NewValidator() *CustomValidator {
	return &CustomValidator{validator: validator.New()}
}

// Validate validates the provided struct against its `validate` tags.
func (cv *CustomValidator) Validate(i any) error {
	return cv.validator.Struct(i)
}
