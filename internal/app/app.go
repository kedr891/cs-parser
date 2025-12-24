package app

import (
	"context"
	"fmt"

	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/internal/bootstrap"
	"github.com/kedr891/cs-parser/internal/parser"
)

func RunAPI(ctx context.Context, cfg *config.Config) error {
	storage, err := bootstrap.InitPGStorage(ctx, cfg)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	defer storage.Close()

	cache, err := bootstrap.InitCache(cfg)
	if err != nil {
		return fmt.Errorf("init cache: %w", err)
	}
	defer cache.Close()

	logger := bootstrap.InitLogger(cfg)
	logger.Info("Starting CS2 Skin Tracker API", "version", cfg.App.Version)

	api := bootstrap.InitAPI(storage, cache, cfg, logger)
	server := bootstrap.InitHTTPServer(cfg, api, logger)

	return bootstrap.RunHTTPServer(ctx, server, logger)
}

func RunPriceConsumer(ctx context.Context, cfg *config.Config) error {
	storage, err := bootstrap.InitPGStorage(ctx, cfg)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	defer storage.Close()

	cache, err := bootstrap.InitCache(cfg)
	if err != nil {
		return fmt.Errorf("init cache: %w", err)
	}
	defer cache.Close()

	logger := bootstrap.InitLogger(cfg)
	logger.Info("Starting CS2 Price Consumer Service", "version", cfg.App.Version)

	kafkaConsumer := bootstrap.InitKafkaConsumer(cfg, cfg.Kafka.TopicPriceUpdated, cfg.Kafka.GroupPriceConsumer)
	defer kafkaConsumer.Close()

	alertProducer, err := bootstrap.InitKafkaProducer(cfg, cfg.Kafka.TopicPriceAlert)
	if err != nil {
		return fmt.Errorf("init alert producer: %w", err)
	}
	defer alertProducer.Close()

	priceConsumer := bootstrap.InitPriceConsumer(storage, cache, kafkaConsumer, alertProducer, logger)

	return bootstrap.RunConsumer(ctx, priceConsumer, logger)
}

func RunParser(ctx context.Context, cfg *config.Config) error {
	logger := bootstrap.InitLogger(cfg)
	logger.Info("Starting CS2 Skin Parser Service", "version", cfg.App.Version)

	var repository parser.Repository
	var closeStorage func()

	if cfg.IsShardingEnabled() {
		logger.Info("Sharding is ENABLED", "shards_count", len(cfg.Shard.URLs))
		shardManager, err := bootstrap.InitShardManager(cfg)
		if err != nil {
			return fmt.Errorf("init shard manager: %w", err)
		}
		closeStorage = func() { shardManager.Close() }
		repository = bootstrap.InitShardedParserRepository(shardManager, logger)
		logger.Info("ShardManager initialized successfully", "shards", shardManager.ShardsCount())
	} else {
		logger.Info("Sharding is DISABLED, using single PostgreSQL instance")
		pg, err := bootstrap.InitPGStorage(ctx, cfg)
		if err != nil {
			return fmt.Errorf("init storage: %w", err)
		}
		closeStorage = func() { pg.Close() }
		repository = bootstrap.InitParserRepository(pg, logger)
	}
	defer closeStorage()

	cache, err := bootstrap.InitCache(cfg)
	if err != nil {
		return fmt.Errorf("init cache: %w", err)
	}
	defer cache.Close()

	priceProducer, err := bootstrap.InitKafkaProducer(cfg, cfg.Kafka.TopicPriceUpdated)
	if err != nil {
		return fmt.Errorf("init price producer: %w", err)
	}
	defer priceProducer.Close()

	discoveryProducer, err := bootstrap.InitKafkaProducer(cfg, cfg.Kafka.TopicSkinDiscovered)
	if err != nil {
		return fmt.Errorf("init discovery producer: %w", err)
	}
	defer discoveryProducer.Close()

	scheduler := bootstrap.InitParserScheduler(repository, cache, priceProducer, discoveryProducer, cfg, logger)

	return bootstrap.RunScheduler(ctx, scheduler, logger)
}

func RunNotification(ctx context.Context, cfg *config.Config) error {
	storage, err := bootstrap.InitPGStorage(ctx, cfg)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	defer storage.Close()

	cache, err := bootstrap.InitCache(cfg)
	if err != nil {
		return fmt.Errorf("init cache: %w", err)
	}
	defer cache.Close()

	logger := bootstrap.InitLogger(cfg)
	logger.Info("Starting CS2 Notification Service", "version", cfg.App.Version)

	kafkaConsumer := bootstrap.InitKafkaConsumer(cfg, cfg.Kafka.TopicPriceAlert, cfg.Kafka.GroupNotification)
	defer kafkaConsumer.Close()

	notificationConsumer := bootstrap.InitNotificationConsumer(storage, cache, kafkaConsumer, logger)

	return bootstrap.RunConsumer(ctx, notificationConsumer, logger)
}
