package bootstrap

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

func InitParserRepository(storage *postgres.Postgres, log *logger.Logger) parser.Repository {
	return parser.NewRepository(storage, log)
}

func InitShardedParserRepository(shardManager *postgres.ShardManager, log *logger.Logger) parser.Repository {
	return parser.NewShardedRepository(shardManager, log)
}

func InitParserScheduler(
	repo parser.Repository,
	cache *redis.Redis,
	priceProducer *kafka.Producer,
	discoveryProducer *kafka.Producer,
	cfg *config.Config,
	log *logger.Logger,
) *parser.Scheduler {
	steamClient := parser.NewSteamClient(cache, log)
	steamAdapter := parser.NewSteamClientAdapter(steamClient)
	redisAdapter := parser.NewRedisAdapter(cache)
	priceProducerAdapter := parser.NewKafkaProducerAdapter(priceProducer)
	discoveryProducerAdapter := parser.NewKafkaProducerAdapter(discoveryProducer)
	logAdapter := parser.NewLoggerAdapter(log)

	parserService := parser.NewService(
		repo,
		steamAdapter,
		priceProducerAdapter,
		discoveryProducerAdapter,
		redisAdapter,
		logAdapter,
	)

	return parser.NewScheduler(
		parserService,
		cfg.Parser.IntervalMinutes,
		log,
	)
}

func RunScheduler(ctx context.Context, scheduler interface{ Start(context.Context) error }, log *logger.Logger) error {
	schedulerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Info("Starting scheduler...")
	go func() {
		if err := scheduler.Start(schedulerCtx); err != nil {
			log.Error("Scheduler error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	log.Info("Received shutdown signal", "signal", sig.String())

	log.Info("Shutting down scheduler...")
	cancel()

	log.Info("Scheduler stopped successfully")
	return nil
}
