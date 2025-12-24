package bootstrap

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/internal/notification"
	"github.com/kedr891/cs-parser/internal/price"
	"github.com/kedr891/cs-parser/pkg/kafka"
	"github.com/kedr891/cs-parser/pkg/logger"
	"github.com/kedr891/cs-parser/pkg/postgres"
	"github.com/kedr891/cs-parser/pkg/redis"
)

type Consumers struct {
	Price        *price.Consumer
	Notification *notification.Consumer
}

func InitConsumers(cfg *config.Config, storage *postgres.Postgres, cache *redis.Redis, producers *Producers, log *logger.Logger) *Consumers {
	priceConsumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.TopicPriceUpdated,
		cfg.Kafka.GroupPriceConsumer,
	)

	notificationConsumer := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.TopicPriceAlert,
		cfg.Kafka.GroupNotification,
	)

	priceRepo := price.NewRepository(storage, log)
	priceRedisAdapter := price.NewRedisAdapter(cache)
	priceAlertProducerAdapter := price.NewKafkaProducerAdapter(producers.PriceAlert)
	priceLogAdapter := price.NewLoggerAdapter(log)
	analytics := price.NewAnalytics(priceRedisAdapter, priceLogAdapter)

	priceConsumerService := price.NewConsumer(
		priceConsumer,
		priceAlertProducerAdapter,
		priceRepo,
		analytics,
		priceRedisAdapter,
		priceLogAdapter,
	)

	notificationRepo := notification.NewRepository(storage, log)
	notificationRedisAdapter := notification.NewRedisAdapter(cache)
	notificationLogAdapter := notification.NewLoggerAdapter(log)

	notifService := notification.NewService(
		notificationRepo,
		notificationRedisAdapter,
		notificationLogAdapter,
	)

	notificationConsumerService := notification.NewConsumer(
		notificationConsumer,
		notifService,
		log,
	)

	return &Consumers{
		Price:        priceConsumerService,
		Notification: notificationConsumerService,
	}
}

func RunConsumer(ctx context.Context, consumer interface{ Start(context.Context) error }, log *logger.Logger) error {
	consumerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Info("Starting consumer...")
	go func() {
		if err := consumer.Start(consumerCtx); err != nil && err != context.Canceled {
			log.Error("Consumer error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Info("Received shutdown signal", "signal", sig.String())

	log.Info("Shutting down consumer...")
	cancel()

	log.Info("Consumer stopped successfully")
	return nil
}
