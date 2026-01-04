package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	_defaultConnAttempts = 10
	_defaultConnTimeout  = time.Second
)

type Redis struct {
	connAttempts int
	connTimeout  time.Duration

	Client *redis.Client
}

func New(addr, password string, db int, opts ...Option) (*Redis, error) {
	r := &Redis{
		connAttempts: _defaultConnAttempts,
		connTimeout:  _defaultConnTimeout,
	}

	for _, opt := range opts {
		opt(r)
	}

	r.Client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	var err error
	for r.connAttempts > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), r.connTimeout)
		err = r.Client.Ping(ctx).Err()
		cancel()

		if err == nil {
			break
		}

		time.Sleep(r.connTimeout)
		r.connAttempts--
	}

	if err != nil {
		return nil, fmt.Errorf("redis - New - connAttempts == 0: %w", err)
	}

	return r, nil
}

func (r *Redis) Close() error {
	if r.Client != nil {
		return r.Client.Close()
	}
	return nil
}

type Option func(*Redis)

func ConnAttempts(attempts int) Option {
	return func(r *Redis) {
		r.connAttempts = attempts
	}
}

func ConnTimeout(timeout time.Duration) Option {
	return func(r *Redis) {
		r.connTimeout = timeout
	}
}

func (r *Redis) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

func CacheKey(prefix string, id string) string {
	return fmt.Sprintf("%s:%s", prefix, id)
}

func (r *Redis) SetCache(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.Client.Set(ctx, key, value, ttl).Err()
}

func (r *Redis) GetCache(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

func (r *Redis) DeleteCache(ctx context.Context, keys ...string) error {
	return r.Client.Del(ctx, keys...).Err()
}

func (r *Redis) IncrementRateLimit(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := r.Client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}

	return incr.Val(), nil
}

func (r *Redis) GetRateLimit(ctx context.Context, key string) (int64, error) {
	val, err := r.Client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

func (r *Redis) IncrementCounter(ctx context.Context, key string) error {
	return r.Client.Incr(ctx, key).Err()
}

func (r *Redis) GetCounter(ctx context.Context, key string) (int64, error) {
	val, err := r.Client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

func (r *Redis) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return r.Client.ZAdd(ctx, key, members...).Err()
}

func (r *Redis) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return r.Client.ZRevRangeWithScores(ctx, key, start, stop).Result()
}

func (r *Redis) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.Client.ZRevRange(ctx, key, start, stop).Result()
}
func (r *Redis) HSet(ctx context.Context, key string, values ...interface{}) error {
	return r.Client.HSet(ctx, key, values...).Err()
}

func (r *Redis) HGet(ctx context.Context, key, field string) (string, error) {
	return r.Client.HGet(ctx, key, field).Result()
}

func (r *Redis) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.Client.HGetAll(ctx, key).Result()
}

func (r *Redis) LPush(ctx context.Context, key string, values ...interface{}) error {
	return r.Client.LPush(ctx, key, values...).Err()
}

func (r *Redis) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.Client.LRange(ctx, key, start, stop).Result()
}

func (r *Redis) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.Client.Keys(ctx, pattern).Result()
}

func (r *Redis) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return r.Client.Scan(ctx, cursor, match, count).Result()
}

func (r *Redis) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.Client.Expire(ctx, key, ttl).Err()
}

func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.Client.TTL(ctx, key).Result()
}

func (r *Redis) Pipeline() redis.Pipeliner {
	return r.Client.Pipeline()
}

func (r *Redis) Stats() *redis.PoolStats {
	return r.Client.PoolStats()
}
