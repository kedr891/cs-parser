package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cs-parser/config"
	"github.com/cs-parser/internal/api/handler"
	"github.com/cs-parser/internal/api/middleware"
	"github.com/cs-parser/internal/api/router"
	"github.com/cs-parser/pkg/logger"
	"github.com/cs-parser/pkg/postgres"
	"github.com/cs-parser/pkg/redis"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.NewConfig()
	if err != nil {
		panic("Failed to load config: " + err.Error())
	}

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

	// Initialize handlers
	skinHandler := handler.NewSkinHandler(pg, rdb, log)
	userHandler := handler.NewUserHandler(pg, rdb, cfg.JWT.Secret, log)
	analyticsHandler := handler.NewAnalyticsHandler(pg, rdb, log)

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

	// Start server
	addr := ":" + cfg.HTTP.Port
	log.Info("Starting HTTP server", "addr", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: engine,
	}

	// Start server in goroutine
	go func() {
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
