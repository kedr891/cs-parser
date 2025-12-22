package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/internal/parser"
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
	log.Info("Starting CS2 Skin Parser Service", "version", cfg.App.Version)

	// Initialize repository with sharding support
	var repo parser.Repository

	if cfg.IsShardingEnabled() {
		log.Info("Sharding is ENABLED", "shards_count", len(cfg.Shard.URLs))

		// Create ShardManager
		shardManager, err := postgres.NewShardManager(
			cfg.Shard.URLs,
			postgres.WithShardMaxPoolSize(cfg.PG.PoolMax),
		)
		if err != nil {
			log.Fatal("Failed to create shard manager", "error", err)
		}
		defer shardManager.Close()

		log.Info("ShardManager initialized successfully", "shards", shardManager.ShardsCount())

		// Create sharded repository
		repo = parser.NewShardedRepository(shardManager, log)
	} else {
		log.Info("Sharding is DISABLED, using single PostgreSQL instance")

		// Initialize PostgreSQL (single instance)
		pg, err := postgres.New(cfg.PG.URL, postgres.MaxPoolSize(cfg.PG.PoolMax))
		if err != nil {
			log.Fatal("Failed to connect to PostgreSQL", "error", err)
		}
		defer pg.Close()
		log.Info("PostgreSQL connected successfully")

		// Create regular repository
		repo = parser.NewRepository(pg, log)
	}

	// Initialize Redis
	log.Info("Connecting to Redis...")
	rdb, err := redis.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatal("Failed to connect to Redis", "error", err)
	}
	defer rdb.Close()
	log.Info("Redis connected successfully")

	// Initialize Kafka Producer
	log.Info("Initializing Kafka producer...")
	priceProducer, err := kafka.NewProducer(
		cfg.Kafka.Brokers,
		cfg.Kafka.TopicPriceUpdated,
		kafka.WithBatchSize(50),
	)
	if err != nil {
		log.Fatal("Failed to create price producer", "error", err)
	}
	defer priceProducer.Close()

	discoveryProducer, err := kafka.NewProducer(
		cfg.Kafka.Brokers,
		cfg.Kafka.TopicSkinDiscovered,
		kafka.WithBatchSize(20),
	)
	if err != nil {
		log.Fatal("Failed to create discovery producer", "error", err)
	}
	defer discoveryProducer.Close()
	log.Info("Kafka producers initialized")

	// Initialize Steam client
	steamClient := parser.NewSteamClient(rdb, log)

	// Initialize adapters
	steamAdapter := parser.NewSteamClientAdapter(steamClient)
	redisAdapter := parser.NewRedisAdapter(rdb)
	priceProducerAdapter := parser.NewKafkaProducerAdapter(priceProducer)
	discoveryProducerAdapter := parser.NewKafkaProducerAdapter(discoveryProducer)
	logAdapter := parser.NewLoggerAdapter(log)

	// Initialize parser service
	parserService := parser.NewService(
		repo,
		steamAdapter,
		priceProducerAdapter,
		discoveryProducerAdapter,
		redisAdapter,
		logAdapter,
	)

	// Initialize scheduler
	scheduler := parser.NewScheduler(
		parserService,
		cfg.Parser.IntervalMinutes,
		log,
	)

	// Context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start scheduler
	log.Info("Starting parser scheduler", "interval_minutes", cfg.Parser.IntervalMinutes)
	go func() {
		if err := scheduler.Start(ctx); err != nil {
			log.Error("Scheduler error", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Info("Received shutdown signal", "signal", sig.String())

	// Graceful shutdown
	log.Info("Shutting down parser service...")
	cancel()

	log.Info("Parser service stopped successfully")
}
