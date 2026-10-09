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
	"github.com/labstack/echo/v5/middleware"
	"github.com/yannick2009/yily/internal/adapter"
	"github.com/yannick2009/yily/internal/config"
	"github.com/yannick2009/yily/internal/handler"
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
	_ = cfg // reserved for future auth via cfg.MasterKey

	// Initialize the database connection
	db := adapter.NewDB()

	// Repositories
	projectRepository := repository.NewProjectRepository(db)
	environmentRepository := repository.NewEnvironmentRepository(db)
	variableRepository := repository.NewVariableRepository(db)
	versionRepository := repository.NewVariableVersionRepository(db)
	tagRepository := repository.NewTagRepository(db)

	// Services
	projectService := services.NewProjectService(projectRepository, environmentRepository)
	environmentService := services.NewEnvironmentService(environmentRepository)
	variableService := services.NewVariableService(variableRepository, versionRepository)
	tagService := services.NewTagService(tagRepository)

	// Handlers
	projectHandler := handler.NewProjectHandler(projectService)
	environmentHandler := handler.NewEnvironmentHandler(environmentService)
	variableHandler := handler.NewVariableHandler(variableService)
	tagHandler := handler.NewTagHandler(tagService)

	// Create a new server instance
	server := echo.New()

	// Validator
	server.Validator = NewValidator()

	// Middleware configuration
	server.Use(middleware.RequestLogger())                                         // Log incoming requests
	server.Use(middleware.Recover())                                               // Recover from panics and return a 500 error
	server.Use(middleware.BodyLimit(1 << 20))                                      // 1 MB
	server.Use(middleware.ContextTimeout(30 * time.Second))                        // Set a timeout for request context
	server.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(50.0))) // Rate limiting 50 requests per second
	server.Use(middleware.Secure())                                                // Enable security headers

	// Define API routes
	routes := server.Group("/api/v1")

	// Projects
	routes.POST("/projects", projectHandler.Create)
	routes.GET("/projects", projectHandler.GetAll)
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
