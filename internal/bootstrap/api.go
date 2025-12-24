package bootstrap

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
	"github.com/kedr891/cs-parser/internal/domain"
	"github.com/kedr891/cs-parser/pkg/logger"
	"github.com/kedr891/cs-parser/pkg/postgres"
	"github.com/kedr891/cs-parser/pkg/redis"
)

type APIComponents struct {
	Adapters     *APIAdapters
	Repositories *APIRepositories
	Services     *APIServices
	Handlers     *APIHandlers
}

type APIAdapters struct {
	Redis  domain.CacheStorage
	Logger domain.Logger
}

type APIRepositories struct {
	Skin      apiRepo.SkinRepository
	User      apiRepo.UserRepository
	Analytics apiRepo.AnalyticsRepository
}

type APIServices struct {
	Skin      *service.SkinService
	User      *service.UserService
	Analytics *service.AnalyticsService
}

type APIHandlers struct {
	Skin      *handler.SkinHandler
	User      *handler.UserHandler
	Analytics *handler.AnalyticsHandler
}

func InitAPI(storage *postgres.Postgres, cache *redis.Redis, cfg *config.Config, log *logger.Logger) *APIComponents {
	adapters := &APIAdapters{
		Redis:  api.NewRedisAdapter(cache),
		Logger: api.NewLoggerAdapter(log),
	}

	repositories := &APIRepositories{
		Skin:      apiRepo.NewSkinRepository(storage),
		User:      apiRepo.NewUserRepository(storage),
		Analytics: apiRepo.NewAnalyticsRepository(storage),
	}

	services := &APIServices{
		Skin:      service.NewSkinService(repositories.Skin, adapters.Redis, adapters.Logger),
		User:      service.NewUserService(repositories.User, adapters.Redis, cfg.JWT.Secret, adapters.Logger),
		Analytics: service.NewAnalyticsService(repositories.Analytics, adapters.Redis, adapters.Logger),
	}

	handlers := &APIHandlers{
		Skin:      handler.NewSkinHandler(services.Skin, adapters.Logger),
		User:      handler.NewUserHandler(services.User, adapters.Logger),
		Analytics: handler.NewAnalyticsHandler(services.Analytics, adapters.Logger),
	}

	return &APIComponents{
		Adapters:     adapters,
		Repositories: repositories,
		Services:     services,
		Handlers:     handlers,
	}
}

func InitHTTPServer(cfg *config.Config, api *APIComponents, log *logger.Logger) *http.Server {
	if cfg.Log.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	engine.Use(gin.Recovery())
	engine.Use(middleware.Logger(log))
	engine.Use(middleware.CORS())

	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT.Secret, log)

	router.SetupRoutes(engine, api.Handlers.Skin, api.Handlers.User, api.Handlers.Analytics, authMiddleware)

	engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": cfg.App.Name,
			"version": cfg.App.Version,
		})
	})

	addr := ":" + cfg.HTTP.Port
	return &http.Server{
		Addr:           addr,
		Handler:        engine,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

func RunHTTPServer(ctx context.Context, srv *http.Server, log *logger.Logger) error {
	go func() {
		log.Info("Starting HTTP server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Failed to start server", "error", err)
		}
	}()

	log.Info("API server started successfully", "addr", srv.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Info("Received shutdown signal", "signal", sig.String())

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Info("Shutting down API server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
		return err
	}

	log.Info("API server stopped successfully")
	return nil
}
