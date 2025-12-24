package bootstrap

import (
	"github.com/kedr891/cs-parser/internal/price"
	"github.com/kedr891/cs-parser/pkg/kafka"
	"github.com/kedr891/cs-parser/pkg/logger"
	"github.com/kedr891/cs-parser/pkg/postgres"
	"github.com/kedr891/cs-parser/pkg/redis"
)

func InitPriceConsumer(
	storage *postgres.Postgres,
	cache *redis.Redis,
	consumer *kafka.Consumer,
	alertProducer *kafka.Producer,
	log *logger.Logger,
) *price.Consumer {
	repo := price.NewRepository(storage, log)
	redisAdapter := price.NewRedisAdapter(cache)
	alertProducerAdapter := price.NewKafkaProducerAdapter(alertProducer)
	logAdapter := price.NewLoggerAdapter(log)
	analytics := price.NewAnalytics(redisAdapter, logAdapter)

	return price.NewConsumer(
		consumer,
		alertProducerAdapter,
		repo,
		analytics,
		redisAdapter,
		logAdapter,
	)
}
