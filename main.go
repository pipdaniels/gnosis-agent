package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gnosis-agent/internal/agents/runtime"
	"gnosis-agent/internal/agents/anomaly"
	"gnosis-agent/internal/config"
	"gnosis-agent/internal/db"
	"gnosis-agent/internal/handlers"
	"gnosis-agent/internal/services/auth"
	"gnosis-agent/internal/web"

	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Load configuration
	configPath := ""
	if len(os.Args) > 2 && os.Args[1] == "start" {
		if len(os.Args) > 3 && os.Args[2] == "--config" {
			configPath = os.Args[3]
		}
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	log.Printf("Starting GNOSISAGENT Platform for organization: %s", cfg.Org.Name)

	// Initialize MongoDB with database-level multi-tenancy
	mongoMgr, err := db.NewMongoManager(cfg.MongoDB.URI, cfg.MongoDB.Database, cfg.MongoDB.Timeout)
	if err != nil {
		log.Fatal("Failed to initialize MongoDB:", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := mongoMgr.Close(ctx); err != nil {
			log.Printf("Error closing MongoDB connection: %v", err)
		}
	}()

	// Create organization database if it doesn't exist
	ctx := context.Background()
	if err := mongoMgr.CreateOrgDatabase(ctx, cfg.Org.ID); err != nil {
		log.Printf("Warning: Could not create org database: %v", err)
	}

	// Initialize Echo web framework
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Serve static files from embedded filesystem
	e.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static/", http.FileServer(web.GetStaticFS()))))

	// Serve service worker
	e.GET("/sw.js", func(c echo.Context) error {
		return c.File("web/static/sw.js")
	})

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"status":  "healthy",
			"version": "1.0.0",
			"org":     cfg.Org.Name,
		})
	})

	// Initialize Go-ADK agent runtime
	agentRuntime := agents.NewAgentRuntime(context.Background())

	// Initialize MLPack bridge by registering the anomaly agent which uses it
	anomalyAgent := anomaly.NewAnomalyAgent(&cfg.Agents.Anomaly)
	if err := agentRuntime.Register(anomalyAgent); err != nil {
		log.Printf("Warning: Failed to register anomaly agent: %v", err)
	} else {
		log.Printf("Registered Anomaly Agent (MLPack enabled)")
	}

	// Initialize Auth Components
	authRepo := db.NewMongoAuthRepository()
	authService := auth.NewAuthService(authRepo, mongoMgr, cfg)
	authHandler := handlers.NewAuthHandler(authService)

	// Public Routes
	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/signin")
	})
	e.GET("/signup", authHandler.SignupPage)
	e.POST("/signup", authHandler.HandleSignup)
	e.GET("/signin", authHandler.SigninPage)
	e.POST("/signin", authHandler.HandleSignin)
	e.GET("/logout", authHandler.Logout)

	webHandler := handlers.NewWebHandler(mongoMgr, authService)

	// Invite Routes (Public)
	e.GET("/invite", webHandler.HandleAcceptInviteView)
	e.POST("/invite/accept", webHandler.HandleAcceptInviteSubmit)

	// Protected Group (Web)
	// Uses JWT from cookie
	protected := e.Group("")
	protected.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey:  []byte(cfg.Security.JWTSecret),
		TokenLookup: "cookie:auth_token",
		ErrorHandler: func(c echo.Context, err error) error {
			return c.Redirect(http.StatusFound, "/signin")
		},
	}))

	protected.GET("/dashboard", handlers.Dashboard)
	protected.GET("/profile", webHandler.HandleProfile)
	protected.GET("/datasets", webHandler.HandleDatasets)
	protected.GET("/targets", webHandler.HandleTargets)
	protected.GET("/decisions", webHandler.HandleDecisions)
	protected.POST("/api/decisions/generate", webHandler.HandleGenerateDecisions)
	protected.GET("/reports", webHandler.HandleReports)
	protected.POST("/api/reports/generate", webHandler.HandleGenerateReport)
	protected.GET("/upload", handlers.Upload)
	protected.POST("/upload", webHandler.HandleUpload)
	protected.POST("/api/upload", webHandler.HandleUpload)

	// Team Management (Protected)
	protected.GET("/team", webHandler.HandleTeamSettings)
	protected.GET("/settings/team", webHandler.HandleTeamSettings) // canonical URL per plan
	protected.POST("/team/invite", webHandler.HandleInviteUser)
	protected.POST("/team/status", webHandler.HandleUserStatusToggle)
	protected.DELETE("/team/delete/:id", webHandler.HandleUserDelete)

	// API Group (Headless / Programmatic)
	// Uses API Key from Header
	api := e.Group("/api")

		api.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
			KeyLookup: "header:" + cfg.Security.APIKeyHeader,
			Validator: func(key string, c echo.Context) (bool, error) {
				// We need OrgID to validate key, usually passed in header or encoded in key
				// For now, let's assume we search all orgs or we require X-Org-ID header
				// Simple approach: Check X-Org-ID header
				orgID := c.Request().Header.Get("X-Org-ID")
				if orgID == "" {
					return false, fmt.Errorf("missing X-Org-ID header")
				}
				
				user, err := authService.ValidateAPIKey(c.Request().Context(), key, orgID)
				if err != nil {
					return false, err
				}
				
				// Set user context
				c.Set("user", user)
				return true, nil
			},
		}))

	api.GET("/", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"message": "GNOSISAGENT API",
			"version": "0.1.0",
		})
	})

	// Start server
	address := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s", address)

	// Graceful shutdown
	go func() {
		if err := e.Start(address); err != nil {
			log.Printf("Server stopped: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
