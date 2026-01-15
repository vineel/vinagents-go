package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/vineel/vinagents-go/internal/config"
	"github.com/vineel/vinagents-go/internal/db"
	"github.com/vineel/vinagents-go/internal/handler"
	"github.com/vineel/vinagents-go/internal/middleware"
	"github.com/vineel/vinagents-go/internal/repository"
	"github.com/vineel/vinagents-go/internal/service"
	"github.com/vineel/vinagents-go/pkg/jwt"

	// Import agents to register them
	_ "github.com/vineel/vinagents-go/internal/agent/simple"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Set up logging
	setupLogging(cfg)

	ctx := context.Background()

	// Create database pool
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("connected to database")

	// Create River client (for enqueueing jobs)
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		slog.Error("failed to create river client", "error", err)
		os.Exit(1)
	}

	// Create repositories
	userRepo := repository.NewUserRepository(pool)
	refreshTokenRepo := repository.NewRefreshTokenRepository(pool)
	agentRunRepo := repository.NewAgentRunRepository(pool)
	agentRunMessageRepo := repository.NewAgentRunMessageRepository(pool)

	// Create JWT manager
	jwtManager := jwt.NewManager(
		cfg.JWTSecret,
		cfg.JWTExpiresIn,
		cfg.JWTRefreshSecret,
		cfg.JWTRefreshExpiresIn,
	)

	// Create services
	authService := service.NewAuthService(userRepo, refreshTokenRepo, jwtManager)
	agentService := service.NewAgentService(agentRunRepo, agentRunMessageRepo, riverClient, pool)

	// Create handlers
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userRepo, jwtManager)
	agentHandler := handler.NewAgentHandler(agentService)

	// Create router
	router := setupRouter(cfg, jwtManager, healthHandler, authHandler, userHandler, agentHandler)

	// Create server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		slog.Info("starting API server", "port", cfg.Port, "prefix", cfg.APIPrefix)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server stopped")
}

func setupLogging(cfg *config.Config) {
	var handler slog.Handler
	if cfg.IsProduction() {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	slog.SetDefault(slog.New(handler))
}

func setupRouter(
	cfg *config.Config,
	jwtManager *jwt.Manager,
	healthHandler *handler.HealthHandler,
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	agentHandler *handler.AgentHandler,
) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.ErrorHandler())

	// CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// API routes
	api := router.Group(cfg.APIPrefix)
	{
		// Public routes
		healthHandler.RegisterRoutes(api)
		authHandler.RegisterRoutes(api)

		// Protected routes
		authMiddleware := middleware.Auth(jwtManager)
		userHandler.RegisterRoutes(api, authMiddleware)
		agentHandler.RegisterRoutes(api, authMiddleware)
	}

	return router
}
