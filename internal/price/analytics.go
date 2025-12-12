package price

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kedr891/cs-parser/internal/entity"
	"github.com/kedr891/cs-parser/pkg/logger"
	"github.com/kedr891/cs-parser/pkg/redis"
	goredis "github.com/redis/go-redis/v9"
)

const (
	// Redis keys
	_keyTrending24h    = "analytics:trending:24h"
	_keyTrending7d     = "analytics:trending:7d"
	_keyMarketOverview = "analytics:market:overview"
	_keyViewsPrefix    = "analytics:views:"
	_keyPopularPrefix  = "analytics:popular:"

	// TTL
	_trendingTTL       = 10 * time.Minute
	_marketOverviewTTL = 10 * time.Minute
)

// Analytics - сервис аналитики
type Analytics struct {
	redis *redis.Redis
	log   *logger.Logger
}

// NewAnalytics - создать сервис аналитики
func NewAnalytics(redis *redis.Redis, log *logger.Logger) *Analytics {
	return &Analytics{
		redis: redis,
		log:   log,
	}
}

// UpdateTrending - обновить трендинг по изменению цены
func (a *Analytics) UpdateTrending(ctx context.Context, event *entity.PriceUpdateEvent) error {
	// Добавить в трендинг 24h
	if err := a.addToTrending(ctx, _keyTrending24h, event); err != nil {
		a.log.Warn("Failed to add to trending 24h", "error", err)
	}

	// Добавить в трендинг 7d
	if err := a.addToTrending(ctx, _keyTrending7d, event); err != nil {
		a.log.Warn("Failed to add to trending 7d", "error", err)
	}

	return nil
}

// addToTrending - добавить скин в трендинг sorted set
func (a *Analytics) addToTrending(ctx context.Context, key string, event *entity.PriceUpdateEvent) error {
	// Score = процент изменения цены
	score := event.PriceChange

	member := goredis.Z{
		Score:  score,
		Member: event.SkinID.String(),
	}

	if err := a.redis.ZAdd(ctx, key, member); err != nil {
		return fmt.Errorf("zadd trending: %w", err)
	}

	// Установить TTL
	if err := a.redis.Expire(ctx, key, _trendingTTL); err != nil {
		a.log.Warn("Failed to set trending TTL", "error", err)
	}

	return nil
}

// GetTrendingSkins - получить топ трендовых скинов
func (a *Analytics) GetTrendingSkins(ctx context.Context, period string, limit int64) ([]string, error) {
	key := _keyTrending24h
	if period == "7d" {
		key = _keyTrending7d
	}

	// Получить топ N скинов по убыванию score
	skinIDs, err := a.redis.ZRevRange(ctx, key, 0, limit-1)
	if err != nil {
		return nil, fmt.Errorf("get trending: %w", err)
	}

	return skinIDs, nil
}

// GetTrendingWithScores - получить трендинг со скорами
func (a *Analytics) GetTrendingWithScores(ctx context.Context, period string, limit int64) ([]goredis.Z, error) {
	key := _keyTrending24h
	if period == "7d" {
		key = _keyTrending7d
	}

	scores, err := a.redis.ZRevRangeWithScores(ctx, key, 0, limit-1)
	if err != nil {
		return nil, fmt.Errorf("get trending with scores: %w", err)
	}

	return scores, nil
}

// IncrementViewCount - инкремент счётчика просмотров скина
func (a *Analytics) IncrementViewCount(ctx context.Context, skinID string) error {
	key := _keyViewsPrefix + skinID
	return a.redis.IncrementCounter(ctx, key)
}

// GetViewCount - получить количество просмотров
func (a *Analytics) GetViewCount(ctx context.Context, skinID string) (int64, error) {
	key := _keyViewsPrefix + skinID
	return a.redis.GetCounter(ctx, key)
}

// UpdateMarketOverview - обновить обзор рынка
func (a *Analytics) UpdateMarketOverview(ctx context.Context, overview *entity.MarketOverview) error {
	data, err := json.Marshal(overview)
	if err != nil {
		return fmt.Errorf("marshal overview: %w", err)
	}

	if err := a.redis.SetCache(ctx, _keyMarketOverview, string(data), _marketOverviewTTL); err != nil {
		return fmt.Errorf("set market overview: %w", err)
	}

	return nil
}

// GetMarketOverview - получить обзор рынка из кэша
func (a *Analytics) GetMarketOverview(ctx context.Context) (*entity.MarketOverview, error) {
	data, err := a.redis.GetCache(ctx, _keyMarketOverview)
	if err != nil {
		return nil, fmt.Errorf("get market overview: %w", err)
	}

	var overview entity.MarketOverview
	if err := json.Unmarshal([]byte(data), &overview); err != nil {
		return nil, fmt.Errorf("unmarshal overview: %w", err)
	}

	return &overview, nil
}

// InvalidateMarketOverview - инвалидировать кэш обзора рынка
func (a *Analytics) InvalidateMarketOverview(ctx context.Context) error {
	return a.redis.DeleteCache(ctx, _keyMarketOverview)
}

// RecordPopularSearch - записать популярный поисковый запрос
func (a *Analytics) RecordPopularSearch(ctx context.Context, query string) error {
	key := _keyPopularPrefix + "searches"

	member := goredis.Z{
		Score:  1, // инкремент на 1
		Member: query,
	}

	return a.redis.ZAdd(ctx, key, member)
}

// GetPopularSearches - получить популярные поисковые запросы
func (a *Analytics) GetPopularSearches(ctx context.Context, limit int64) ([]string, error) {
	key := _keyPopularPrefix + "searches"
	return a.redis.ZRevRange(ctx, key, 0, limit-1)
}

// UpdatePriceVolatility - обновить волатильность цены
func (a *Analytics) UpdatePriceVolatility(ctx context.Context, skinID string, volatility float64) error {
	key := fmt.Sprintf("analytics:volatility:%s", skinID)

	data := map[string]interface{}{
		"volatility": volatility,
		"updated_at": time.Now().Unix(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal volatility: %w", err)
	}

	return a.redis.SetCache(ctx, key, string(jsonData), 24*time.Hour)
}

// GetPriceVolatility - получить волатильность цены
func (a *Analytics) GetPriceVolatility(ctx context.Context, skinID string) (float64, error) {
	key := fmt.Sprintf("analytics:volatility:%s", skinID)

	data, err := a.redis.GetCache(ctx, key)
	if err != nil {
		return 0, fmt.Errorf("get volatility: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return 0, fmt.Errorf("unmarshal volatility: %w", err)
	}

	if vol, ok := result["volatility"].(float64); ok {
		return vol, nil
	}

	return 0, fmt.Errorf("volatility not found")
}

// TrackPriceAlert - отследить отправленный алерт (чтобы не спамить)
func (a *Analytics) TrackPriceAlert(ctx context.Context, userID, skinID string) error {
	key := fmt.Sprintf("alerts:sent:%s:%s", userID, skinID)

	// Сохраняем на 1 час, чтобы не отправлять алерт повторно
	return a.redis.SetCache(ctx, key, "1", time.Hour)
}

// WasAlertSent - был ли уже отправлен алерт
func (a *Analytics) WasAlertSent(ctx context.Context, userID, skinID string) (bool, error) {
	key := fmt.Sprintf("alerts:sent:%s:%s", userID, skinID)

	_, err := a.redis.GetCache(ctx, key)
	if err != nil {
		return false, nil // алерт не отправлялся
	}

	return true, nil // алерт уже отправлялся
}

// ClearTrending - очистить трендинг (для admin endpoints)
func (a *Analytics) ClearTrending(ctx context.Context) error {
	if err := a.redis.DeleteCache(ctx, _keyTrending24h); err != nil {
		return err
	}
	return a.redis.DeleteCache(ctx, _keyTrending7d)
}

// GetAnalyticsStats - получить общую статистику аналитики
func (a *Analytics) GetAnalyticsStats(ctx context.Context) (*AnalyticsStats, error) {
	// Количество записей в трендинге
	trending24hCount, _ := a.getTrendingCount(ctx, _keyTrending24h)
	trending7dCount, _ := a.getTrendingCount(ctx, _keyTrending7d)

	// Общее количество просмотров (суммируем все счётчики views)
	// Это упрощённый вариант, в продакшене лучше использовать отдельный счётчик
	totalViews := int64(0)

	return &AnalyticsStats{
		TrendingCount24h: trending24hCount,
		TrendingCount7d:  trending7dCount,
		TotalViews:       totalViews,
		UpdatedAt:        time.Now(),
	}, nil
}

// getTrendingCount - получить количество элементов в трендинге
func (a *Analytics) getTrendingCount(ctx context.Context, key string) (int64, error) {
	// Используем ZCARD для получения количества элементов в sorted set
	count, err := a.redis.Client.ZCard(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	return count, nil
}

// AnalyticsStats - статистика аналитики
type AnalyticsStats struct {
	TrendingCount24h int64     `json:"trending_count_24h"`
	TrendingCount7d  int64     `json:"trending_count_7d"`
	TotalViews       int64     `json:"total_views"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// BatchUpdateTrending - пакетное обновление трендинга
func (a *Analytics) BatchUpdateTrending(ctx context.Context, events []*entity.PriceUpdateEvent) error {
	if len(events) == 0 {
		return nil
	}

	pipe := a.redis.Pipeline()

	for _, event := range events {
		member := goredis.Z{
			Score:  event.PriceChange,
			Member: event.SkinID.String(),
		}

		pipe.ZAdd(ctx, _keyTrending24h, member)
		pipe.ZAdd(ctx, _keyTrending7d, member)
	}

	pipe.Expire(ctx, _keyTrending24h, _trendingTTL)
	pipe.Expire(ctx, _keyTrending7d, _trendingTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("batch update trending: %w", err)
	}

	return nil
}
