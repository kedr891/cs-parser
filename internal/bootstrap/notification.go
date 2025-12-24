package bootstrap

import (
	"github.com/kedr891/cs-parser/internal/notification"
	"github.com/kedr891/cs-parser/pkg/kafka"
	"github.com/kedr891/cs-parser/pkg/logger"
	"github.com/kedr891/cs-parser/pkg/postgres"
	"github.com/kedr891/cs-parser/pkg/redis"
)

func InitNotificationConsumer(
	pg *postgres.Postgres,
	cache *redis.Redis,
	consumer *kafka.Consumer,
	log *logger.Logger,
) *notification.Consumer {
	repo := notification.NewRepository(pg, log)
	redisAdapter := notification.NewRedisAdapter(cache)
	logAdapter := notification.NewLoggerAdapter(log)

	notifService := notification.NewService(
		repo,
		redisAdapter,
		logAdapter,
	)

	return notification.NewConsumer(
		consumer,
		notifService,
		log,
	)
}
