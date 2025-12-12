package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/internal/price"
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
	log.Info("Starting CS2 Price Consumer Service", "version", cfg.App.Version)

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

	// Initialize Kafka Consumer for price updates
	log.Info("Initializing Kafka consumer...")
	consumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.TopicPriceUpdated,
		cfg.Kafka.GroupPriceConsumer,
	)
	defer consumer.Close()
	log.Info("Kafka consumer initialized")

	// Initialize Kafka Producer for alerts
	alertProducer, err := kafka.NewProducer(
		cfg.Kafka.Brokers,
		cfg.Kafka.TopicPriceAlert,
	)
	if err != nil {
		log.Fatal(fmt.Errorf("kafka.NewProducer: %w", err))
	}
	defer alertProducer.Close()

	// Initialize repository
	repo := price.NewRepository(pg, log)

	// Initialize analytics
	analytics := price.NewAnalytics(rdb, log)

	// Initialize price consumer
	priceConsumer := price.NewConsumer(
		consumer,
		alertProducer,
		repo,
		analytics,
		rdb,
		log,
	)

	// Context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consuming messages
	log.Info("Starting price consumer...")
	go func() {
		if err := priceConsumer.Start(ctx); err != nil && err != context.Canceled {
			log.Error("Consumer error", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Info("Received shutdown signal", "signal", sig.String())

	// Graceful shutdown
	log.Info("Shutting down price consumer...")
	cancel()

	log.Info("Price consumer stopped successfully")
}
