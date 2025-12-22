package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/internal/notification"
	"github.com/kedr891/cs-parser/pkg/kafka"
	"github.com/kedr891/cs-parser/pkg/logger"
	"github.com/kedr891/cs-parser/pkg/postgres"
	"github.com/kedr891/cs-parser/pkg/redis"
)

func main() {
	// Load configuration
	cfg, err := config.NewConfig()
	if err != nil {
		panic("Failed to load config: " + err.Error())
	}

	// Initialize logger
	log := logger.New(cfg.Log.Level)
	log.Info("Starting CS2 Notification Service", "version", cfg.App.Version)

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

	// Initialize Kafka Consumer for price alerts
	log.Info("Initializing Kafka consumer...")
	consumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.TopicPriceAlert,
		cfg.Kafka.GroupNotification,
	)
	defer consumer.Close()
	log.Info("Kafka consumer initialized")

	// Initialize repository
	repo := notification.NewRepository(pg, log)

	// Initialize adapters
	redisAdapter := notification.NewRedisAdapter(rdb)
	logAdapter := notification.NewLoggerAdapter(log)

	// Initialize notification service with interfaces
	notifService := notification.NewService(
		repo,
		redisAdapter,
		logAdapter,
	)

	// Initialize notification consumer
	notifConsumer := notification.NewConsumer(
		consumer,
		notifService,
		log, // Consumer still uses concrete logger
	)

	// Context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consuming messages
	log.Info("Starting notification consumer...")
	go func() {
		if err := notifConsumer.Start(ctx); err != nil && err != context.Canceled {
			log.Error("Consumer error", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Info("Received shutdown signal", "signal", sig.String())

	// Graceful shutdown
	log.Info("Shutting down notification service...")
	cancel()

	log.Info("Notification service stopped successfully")
}
