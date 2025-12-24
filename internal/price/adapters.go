package price

import (
	"context"
	"encoding/json"
	"time"

	"github.com/kedr891/cs-parser/internal/domain"
	"github.com/kedr891/cs-parser/pkg/kafka"
	"github.com/kedr891/cs-parser/pkg/logger"
	"github.com/kedr891/cs-parser/pkg/redis"
	goredis "github.com/redis/go-redis/v9"
)

type RedisAdapter struct {
	redis *redis.Redis
}

func NewRedisAdapter(rdb *redis.Redis) domain.CacheStorage {
	return &RedisAdapter{
		redis: rdb,
	}
}

func (a *RedisAdapter) Get(ctx context.Context, key string) (string, error) {
	return a.redis.GetCache(ctx, key)
}

func (a *RedisAdapter) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return a.redis.SetCache(ctx, key, value, ttl)
}

func (a *RedisAdapter) Delete(ctx context.Context, key string) error {
	return a.redis.DeleteCache(ctx, key)
}

func (a *RedisAdapter) Exists(ctx context.Context, key string) (bool, error) {
	result, err := a.redis.Client.Exists(ctx, key).Result()
	return result > 0, err
}

func (a *RedisAdapter) IncrementRateLimit(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	return a.redis.IncrementRateLimit(ctx, key, ttl)
}

func (a *RedisAdapter) GetRateLimit(ctx context.Context, key string) (int64, error) {
	return a.redis.GetRateLimit(ctx, key)
}

func (a *RedisAdapter) Increment(ctx context.Context, key string) (int64, error) {
	return a.redis.Client.Incr(ctx, key).Result()
}

func (a *RedisAdapter) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return a.redis.SetCache(ctx, key, string(data), ttl)
}

func (a *RedisAdapter) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := a.redis.GetCache(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (a *RedisAdapter) ZAdd(ctx context.Context, key string, score float64, member string) error {
	return a.redis.Client.ZAdd(ctx, key, goredis.Z{
		Score:  score,
		Member: member,
	}).Err()
}

func (a *RedisAdapter) ZIncrBy(ctx context.Context, key string, increment float64, member string) (float64, error) {
	return a.redis.Client.ZIncrBy(ctx, key, increment, member).Result()
}

func (a *RedisAdapter) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return a.redis.ZRevRange(ctx, key, start, stop)
}

func (a *RedisAdapter) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]domain.ZMember, error) {
	results, err := a.redis.Client.ZRevRangeWithScores(ctx, key, start, stop).Result()
	if err != nil {
		return nil, err
	}

	members := make([]domain.ZMember, len(results))
	for i, z := range results {
		members[i] = domain.ZMember{
			Member: z.Member.(string),
			Score:  z.Score,
		}
	}
	return members, nil
}

func (a *RedisAdapter) HSet(ctx context.Context, key, field string, value interface{}) error {
	return a.redis.Client.HSet(ctx, key, field, value).Err()
}

func (a *RedisAdapter) HGet(ctx context.Context, key, field string) (string, error) {
	return a.redis.Client.HGet(ctx, key, field).Result()
}

func (a *RedisAdapter) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return a.redis.Client.HGetAll(ctx, key).Result()
}

type KafkaProducerAdapter struct {
	producer *kafka.Producer
}

func NewKafkaProducerAdapter(producer *kafka.Producer) domain.MessageProducer {
	return &KafkaProducerAdapter{
		producer: producer,
	}
}

func (a *KafkaProducerAdapter) WriteMessage(ctx context.Context, key string, value interface{}) error {
	return a.producer.WriteMessage(ctx, key, value)
}

func (a *KafkaProducerAdapter) Close() error {
	return a.producer.Close()
}

type LoggerAdapter struct {
	logger *logger.Logger
}

func NewLoggerAdapter(log *logger.Logger) domain.Logger {
	return &LoggerAdapter{
		logger: log,
	}
}

func (a *LoggerAdapter) Debug(msg string, args ...interface{}) {
	a.logger.Debug(msg, args...)
}

func (a *LoggerAdapter) Info(msg string, args ...interface{}) {
	a.logger.Info(msg, args...)
}

func (a *LoggerAdapter) Warn(msg string, args ...interface{}) {
	a.logger.Warn(msg, args...)
}

func (a *LoggerAdapter) Error(msg string, args ...interface{}) {
	a.logger.Error(msg, args...)
}

func (a *LoggerAdapter) Fatal(msg string, args ...interface{}) {
	a.logger.Fatal(msg, args...)
}
