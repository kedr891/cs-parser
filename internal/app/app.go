// Package app configures and runs the application.
package app

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/internal/api"
	"github.com/kedr891/cs-parser/internal/api/handler"
	"github.com/kedr891/cs-parser/internal/api/middleware"
	apiRepo "github.com/kedr891/cs-parser/internal/api/repository"
	"github.com/kedr891/cs-parser/internal/api/router"
	"github.com/kedr891/cs-parser/internal/api/service"
	"github.com/kedr891/cs-parser/pkg/logger"
	"github.com/kedr891/cs-parser/pkg/postgres"
	"github.com/kedr891/cs-parser/pkg/redis"
)

// Run initializes and starts the API application.
func Run(cfg *config.Config) {
	// Initialize logger
	log := logger.New(cfg.Log.Level)
	log.Info("Starting CS2 Skin Tracker API", "version", cfg.App.Version)

	// Initialize PostgreSQL
	log.Info("Connecting to PostgreSQL...")
	pg, err := postgres.New(cfg.PG.URL, postgres.MaxPoolSize(cfg.PG.PoolMax))
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL", "error", err)
	}
	defer pg.Close()
	log.Info("PostgreSQL connected successfully")

	// Initialize Redis
	log.Info("Connecting to Redis...")
	rdb, err := redis.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatal("Failed to connect to Redis", "error", err)
	}
	defer rdb.Close()
	log.Info("Redis connected successfully")

	// Initialize adapters
	redisAdapter := api.NewRedisAdapter(rdb)
	logAdapter := api.NewLoggerAdapter(log)

	// Initialize repositories
	skinRepo := apiRepo.NewSkinRepository(pg)
	userRepo := apiRepo.NewUserRepository(pg)
	analyticsRepo := apiRepo.NewAnalyticsRepository(pg)

	// Initialize services with interfaces
	skinService := service.NewSkinService(skinRepo, redisAdapter, logAdapter)
	userService := service.NewUserService(userRepo, redisAdapter, cfg.JWT.Secret, logAdapter)
	analyticsService := service.NewAnalyticsService(analyticsRepo, redisAdapter, logAdapter)

	// Initialize handlers with services
	skinHandler := handler.NewSkinHandler(skinService, logAdapter)
	userHandler := handler.NewUserHandler(userService, logAdapter)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService, logAdapter)

	// Set Gin mode
	if cfg.Log.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin engine
	engine := gin.New()

	// Global middleware
	engine.Use(gin.Recovery())
	engine.Use(middleware.Logger(log))
	engine.Use(middleware.CORS())

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT.Secret, log)

	// Setup routes
	router.SetupRoutes(engine, skinHandler, userHandler, analyticsHandler, authMiddleware)

	// Health check
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": cfg.App.Name,
			"version": cfg.App.Version,
		})
	})

	// Create HTTP server with proper configuration
	addr := ":" + cfg.HTTP.Port
	srv := &http.Server{
		Addr:           addr,
		Handler:        engine,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// Start server in goroutine
	go func() {
		log.Info("Starting HTTP server", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", "error", err)
		}
	}()

	log.Info("API server started successfully", "addr", addr)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Info("Received shutdown signal", "signal", sig.String())

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info("Shutting down API server...")
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("API server stopped successfully")
}
