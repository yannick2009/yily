package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	"github.com/yannick2009/yily/internal/service"
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
	auditService := service.NewAuditService(auditRepository, middleware.ActorFromContext)
	authService := service.NewAuthService(userRepository, sessionRepository)
	patService := service.NewPATService(userRepository, patRepository, auditService)
	projectService := service.NewProjectService(projectRepository, environmentRepository, tagRepository, auditService)
	environmentService := service.NewEnvironmentService(environmentRepository, variableRepository, auditService)
	variableService := service.NewVariableService(variableRepository, versionRepository, environmentRepository, auditService, cfg.MasterKey)
	tagService := service.NewTagService(tagRepository, auditService)
	dashboardService := service.NewDashboardService(projectRepository, environmentRepository, variableRepository, patRepository, auditService)

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

	// Serve a dist directory placed next to the executable. Missing builds are non-fatal.
	if err := registerClient(server); err != nil {
		server.Logger.Warn("client static files disabled", "error", err)
	}

	// Authentication and scope guards
	authMiddleware := middleware.RequireAuth(authService, patService)
	writeScope := middleware.RequireWriteScope()
	sessionOnly := middleware.RequireSession()

	// Define API routes
	api := server.Group("/api/v1")

	// Public authentication routes
	api.GET("/auth/status", authHandler.Status)
	api.POST("/auth/setup", authHandler.Setup)
	api.POST("/auth/login", authHandler.Login)

	// Read routes: any authenticated caller (session, read PAT, write PAT).
	readRoutes := api.Group("", authMiddleware)
	// Write routes: session or write-scoped PAT only.
	writeRoutes := api.Group("", authMiddleware, writeScope)
	// Admin routes: web session only — not accessible via any PAT.
	adminRoutes := api.Group("", authMiddleware, sessionOnly)

	// Auth and token management (session-only)
	adminRoutes.POST("/auth/logout", authHandler.Logout)
	adminRoutes.GET("/auth/me", authHandler.Me)
	adminRoutes.POST("/tokens", patHandler.Create)
	adminRoutes.GET("/tokens", patHandler.GetAll)
	adminRoutes.POST("/tokens/:id/rotate", patHandler.Rotate)
	adminRoutes.DELETE("/tokens/:id", patHandler.Revoke)

	// Token introspection (read): returns metadata about the PAT used to authenticate.
	readRoutes.GET("/tokens/me", patHandler.Me)

	// Dashboard (read)
	readRoutes.GET("/dashboard", dashboardHandler.Get)

	// Projects
	readRoutes.GET("/projects", projectHandler.GetAll)
	readRoutes.GET("/projects/:id", projectHandler.GetByID)
	writeRoutes.POST("/projects", projectHandler.Create)
	// Clearing every project is a UI-only action; it is not reachable with a PAT.
	adminRoutes.DELETE("/projects", projectHandler.Clear)
	writeRoutes.PATCH("/projects/:id", projectHandler.SetName)
	writeRoutes.DELETE("/projects/:id", projectHandler.Delete)

	// Environments
	readRoutes.GET("/projects/:projectID/environments", environmentHandler.GetAll)
	readRoutes.GET("/environments/:id", environmentHandler.GetByID)
	readRoutes.GET("/environments/:id/export", environmentHandler.Export)
	writeRoutes.POST("/projects/:projectID/environments", environmentHandler.Create)
	writeRoutes.PATCH("/environments/:id/name", environmentHandler.SetName)
	writeRoutes.PATCH("/environments/:id/description", environmentHandler.SetDescription)
	writeRoutes.DELETE("/environments/:id", environmentHandler.Delete)
	writeRoutes.POST("/environments/:id/clone", environmentHandler.Clone)
	writeRoutes.POST("/environments/:id/imports", environmentHandler.Import)

	// Variables
	readRoutes.GET("/environments/:environmentID/variables/values", variableHandler.GetValues)
	readRoutes.GET("/environments/:environmentID/variables", variableHandler.List)
	readRoutes.GET("/variables/:id", variableHandler.GetByID)
	readRoutes.GET("/variables/:id/value", variableHandler.GetValue)
	writeRoutes.POST("/environments/:environmentID/variables", variableHandler.Create)
	writeRoutes.PATCH("/variables/:id", variableHandler.SetName)
	writeRoutes.DELETE("/variables/:id", variableHandler.Delete)
	writeRoutes.PUT("/variables/:id/tags", variableHandler.SetTags)

	// Variable versions
	readRoutes.GET("/variables/:id/versions", variableHandler.GetAllVersions)
	readRoutes.GET("/variables/:id/versions/current", variableHandler.GetCurrentVersion)
	writeRoutes.POST("/variables/:id/versions", variableHandler.AddVersion)
	writeRoutes.PATCH("/variables/:id/versions/:versionID", variableHandler.SetVersionActive)
	writeRoutes.DELETE("/variables/:id/value", variableHandler.ClearValue)

	// Tags
	readRoutes.GET("/projects/:projectID/tags", tagHandler.GetAll)
	writeRoutes.POST("/projects/:projectID/tags", tagHandler.Create)
	writeRoutes.DELETE("/projects/:projectID/tags/:tagID", tagHandler.Delete)

	// Audit log
	readRoutes.GET("/audit", auditHandler.List)
	writeRoutes.GET("/audit/export", auditHandler.Export)

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

// registerClient serves the built SPA from a "dist" directory placed next to
// the server executable. It is a no-op when dist is absent, so development
// builds run without a client build.
func registerClient(server *echo.Echo) error {
	executable, err := os.Executable()
	if err != nil {
		return nil
	}

	dir := filepath.Join(filepath.Dir(executable), "dist")
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no client build alongside the server: normal in development
		}
		return fmt.Errorf("dist: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("dist path %q is not a directory", dir)
	}

	indexPath := filepath.Join(dir, "index.html")
	indexInfo, err := os.Stat(indexPath)
	if err != nil {
		return fmt.Errorf("index.html: %w", err)
	}
	if indexInfo.IsDir() {
		return fmt.Errorf("index.html in %q is a directory", dir)
	}

	server.Use(echomw.StaticWithConfig(echomw.StaticConfig{
		Filesystem: os.DirFS(dir),
		Root:       ".",
		Index:      "index.html",
		HTML5:      true,
		Skipper: func(c *echo.Context) bool {
			path := c.Request().URL.Path
			// API routes must keep their JSON 404s instead of falling back to the SPA.
			return path == "/api" || strings.HasPrefix(path, "/api/")
		},
	}))

	return nil
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
