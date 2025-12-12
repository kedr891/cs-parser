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

// Redis -.
type Redis struct {
	connAttempts int
	connTimeout  time.Duration

	Client *redis.Client
}

// New -.
func New(addr, password string, db int, opts ...Option) (*Redis, error) {
	r := &Redis{
		connAttempts: _defaultConnAttempts,
		connTimeout:  _defaultConnTimeout,
	}

	// Custom options
	for _, opt := range opts {
		opt(r)
	}

	r.Client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Пробуем подключиться с повторами
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

// Close -.
func (r *Redis) Close() error {
	if r.Client != nil {
		return r.Client.Close()
	}
	return nil
}

// Option -.
type Option func(*Redis)

// ConnAttempts -.
func ConnAttempts(attempts int) Option {
	return func(r *Redis) {
		r.connAttempts = attempts
	}
}

// ConnTimeout -.
func ConnTimeout(timeout time.Duration) Option {
	return func(r *Redis) {
		r.connTimeout = timeout
	}
}

// Ping - проверка соединения
func (r *Redis) Ping(ctx context.Context) error {
	return r.Client.Ping(ctx).Err()
}

// --- Кэш helpers ---

// CacheKey - генерация ключа кэша
func CacheKey(prefix string, id string) string {
	return fmt.Sprintf("%s:%s", prefix, id)
}

// SetCache - сохранить в кэш с TTL
func (r *Redis) SetCache(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return r.Client.Set(ctx, key, value, ttl).Err()
}

// GetCache - получить из кэша
func (r *Redis) GetCache(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

// DeleteCache - удалить из кэша
func (r *Redis) DeleteCache(ctx context.Context, keys ...string) error {
	return r.Client.Del(ctx, keys...).Err()
}

// --- Rate Limiting ---

// IncrementRateLimit - инкремент счётчика rate limit
func (r *Redis) IncrementRateLimit(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := r.Client.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}

	return incr.Val(), nil
}

// GetRateLimit - получить значение счётчика
func (r *Redis) GetRateLimit(ctx context.Context, key string) (int64, error) {
	val, err := r.Client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// --- Analytics ---

// IncrementCounter - инкремент счётчика
func (r *Redis) IncrementCounter(ctx context.Context, key string) error {
	return r.Client.Incr(ctx, key).Err()
}

// GetCounter - получить значение счётчика
func (r *Redis) GetCounter(ctx context.Context, key string) (int64, error) {
	val, err := r.Client.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// --- Sorted Sets (для трендинга) ---

// ZAdd - добавить в sorted set
func (r *Redis) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return r.Client.ZAdd(ctx, key, members...).Err()
}

// ZRevRangeWithScores - получить топ элементов (по убыванию)
func (r *Redis) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return r.Client.ZRevRangeWithScores(ctx, key, start, stop).Result()
}

// ZRevRange - получить топ элементов (только ключи)
func (r *Redis) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.Client.ZRevRange(ctx, key, start, stop).Result()
}

// --- Hash operations ---

// HSet - установить поле в hash
func (r *Redis) HSet(ctx context.Context, key string, values ...interface{}) error {
	return r.Client.HSet(ctx, key, values...).Err()
}

// HGet - получить поле из hash
func (r *Redis) HGet(ctx context.Context, key, field string) (string, error) {
	return r.Client.HGet(ctx, key, field).Result()
}

// HGetAll - получить все поля из hash
func (r *Redis) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.Client.HGetAll(ctx, key).Result()
}

// --- List operations ---

// LPush - добавить в начало списка
func (r *Redis) LPush(ctx context.Context, key string, values ...interface{}) error {
	return r.Client.LPush(ctx, key, values...).Err()
}

// LRange - получить элементы списка
func (r *Redis) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.Client.LRange(ctx, key, start, stop).Result()
}

// --- Pattern operations ---

// Keys - найти ключи по паттерну (использовать осторожно на продакшене!)
func (r *Redis) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.Client.Keys(ctx, pattern).Result()
}

// Scan - более безопасная альтернатива Keys
func (r *Redis) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return r.Client.Scan(ctx, cursor, match, count).Result()
}

// --- Expiration ---

// Expire - установить TTL для ключа
func (r *Redis) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return r.Client.Expire(ctx, key, ttl).Err()
}

// TTL - получить оставшееся время жизни ключа
func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.Client.TTL(ctx, key).Result()
}

// --- Batch operations ---

// Pipeline - создать pipeline для batch операций
func (r *Redis) Pipeline() redis.Pipeliner {
	return r.Client.Pipeline()
}

// --- Health check ---

// Stats - статистика клиента
func (r *Redis) Stats() *redis.PoolStats {
	return r.Client.PoolStats()
}
