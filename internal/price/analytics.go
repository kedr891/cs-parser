package price

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kedr891/cs-parser/internal/domain"
	"github.com/kedr891/cs-parser/internal/entity"
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
	cache domain.CacheStorage
	log   domain.Logger
}

// NewAnalytics - создать сервис аналитики
func NewAnalytics(cache domain.CacheStorage, log domain.Logger) *Analytics {
	return &Analytics{
		cache: cache,
		log:   log,
	}
}

// UpdateTrending - обновить трендинг по изменению цены
func (a *Analytics) UpdateTrending(ctx context.Context, event *entity.PriceUpdateEvent) error {
	// Используем ZIncrBy для накопления изменений цены
	score := event.PriceChange

	// Добавить в трендинг 24h
	if _, err := a.cache.ZIncrBy(ctx, _keyTrending24h, score, event.SkinID.String()); err != nil {
		a.log.Warn("Failed to update trending 24h", "error", err)
	}

	// Добавить в трендинг 7d
	if _, err := a.cache.ZIncrBy(ctx, _keyTrending7d, score, event.SkinID.String()); err != nil {
		a.log.Warn("Failed to update trending 7d", "error", err)
	}

	return nil
}

// addToTrending - добавить скин в трендинг sorted set (для обратной совместимости)
func (a *Analytics) addToTrending(ctx context.Context, key string, event *entity.PriceUpdateEvent) error {
	score := event.PriceChange
	return a.cache.ZAdd(ctx, key, score, event.SkinID.String())
}

// GetTrendingSkins - получить топ трендовых скинов (только ID)
func (a *Analytics) GetTrendingSkins(ctx context.Context, period string, limit int64) ([]string, error) {
	key := a.getTrendingKey(period)
	if key == "" {
		return nil, fmt.Errorf("invalid period: %s", period)
	}

	skinIDs, err := a.cache.ZRevRange(ctx, key, 0, limit-1)
	if err != nil {
		return nil, fmt.Errorf("get trending skins: %w", err)
	}

	return skinIDs, nil
}

// GetTrendingWithScores - получить трендинг со скорами
func (a *Analytics) GetTrendingWithScores(ctx context.Context, period string, limit int64) ([]domain.ZMember, error) {
	key := a.getTrendingKey(period)
	if key == "" {
		return nil, fmt.Errorf("invalid period: %s", period)
	}

	members, err := a.cache.ZRevRangeWithScores(ctx, key, 0, limit-1)
	if err != nil {
		return nil, fmt.Errorf("get trending with scores: %w", err)
	}

	return members, nil
}

// getTrendingKey - получить ключ для трендинга по периоду
func (a *Analytics) getTrendingKey(period string) string {
	switch period {
	case "24h":
		return _keyTrending24h
	case "7d":
		return _keyTrending7d
	default:
		return ""
	}
}

// IncrementViewCount - инкремент счётчика просмотров скина
func (a *Analytics) IncrementViewCount(ctx context.Context, skinID string) error {
	key := _keyViewsPrefix + skinID
	_, err := a.cache.IncrementRateLimit(ctx, key, 24*time.Hour)
	return err
}

// GetViewCount - получить количество просмотров
func (a *Analytics) GetViewCount(ctx context.Context, skinID string) (int64, error) {
	key := _keyViewsPrefix + skinID
	return a.cache.GetRateLimit(ctx, key)
}

// UpdateMarketOverview - обновить обзор рынка
func (a *Analytics) UpdateMarketOverview(ctx context.Context, overview *entity.MarketOverview) error {
	data, err := json.Marshal(overview)
	if err != nil {
		return fmt.Errorf("marshal overview: %w", err)
	}

	if err := a.cache.Set(ctx, _keyMarketOverview, string(data), _marketOverviewTTL); err != nil {
		return fmt.Errorf("set market overview: %w", err)
	}

	return nil
}

// GetMarketOverview - получить обзор рынка из кэша
func (a *Analytics) GetMarketOverview(ctx context.Context) (*entity.MarketOverview, error) {
	data, err := a.cache.Get(ctx, _keyMarketOverview)
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
	return a.cache.Delete(ctx, _keyMarketOverview)
}

// RecordPopularSearch - записать популярный поисковый запрос
func (a *Analytics) RecordPopularSearch(ctx context.Context, query string) error {
	key := _keyPopularPrefix + "searches"
	_, err := a.cache.ZIncrBy(ctx, key, 1, query)
	return err
}

// GetPopularSearches - получить популярные поисковые запросы
func (a *Analytics) GetPopularSearches(ctx context.Context, limit int64) ([]string, error) {
	key := _keyPopularPrefix + "searches"

	queries, err := a.cache.ZRevRange(ctx, key, 0, limit-1)
	if err != nil {
		return nil, fmt.Errorf("get popular searches: %w", err)
	}

	return queries, nil
}

// GetPopularSearchesWithScores - получить популярные запросы с количеством поисков
func (a *Analytics) GetPopularSearchesWithScores(ctx context.Context, limit int64) ([]domain.ZMember, error) {
	key := _keyPopularPrefix + "searches"

	members, err := a.cache.ZRevRangeWithScores(ctx, key, 0, limit-1)
	if err != nil {
		return nil, fmt.Errorf("get popular searches with scores: %w", err)
	}

	return members, nil
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

	return a.cache.Set(ctx, key, string(jsonData), 24*time.Hour)
}

// GetPriceVolatility - получить волатильность цены
func (a *Analytics) GetPriceVolatility(ctx context.Context, skinID string) (float64, error) {
	key := fmt.Sprintf("analytics:volatility:%s", skinID)

	data, err := a.cache.Get(ctx, key)
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
	return a.cache.Set(ctx, key, "1", time.Hour)
}

// WasAlertSent - был ли уже отправлен алерт
func (a *Analytics) WasAlertSent(ctx context.Context, userID, skinID string) (bool, error) {
	key := fmt.Sprintf("alerts:sent:%s:%s", userID, skinID)

	_, err := a.cache.Get(ctx, key)
	if err != nil {
		return false, nil // алерт не отправлялся
	}

	return true, nil // алерт уже отправлялся
}

// ClearTrending - очистить трендинг (для admin endpoints)
func (a *Analytics) ClearTrending(ctx context.Context) error {
	if err := a.cache.Delete(ctx, _keyTrending24h); err != nil {
		return err
	}
	return a.cache.Delete(ctx, _keyTrending7d)
}

// GetAnalyticsStats - получить общую статистику аналитики
func (a *Analytics) GetAnalyticsStats(ctx context.Context) (*AnalyticsStats, error) {
	// Упрощённая версия - для полной реализации нужен метод ZCard в интерфейсе
	// или можно использовать ZRevRange с большим лимитом и посчитать длину
	return &AnalyticsStats{
		TrendingCount24h: 0, // требует ZCard или подсчета через ZRevRange
		TrendingCount7d:  0, // требует ZCard или подсчета через ZRevRange
		TotalViews:       0, // требует сканирования всех ключей views:*
		UpdatedAt:        time.Now(),
	}, nil
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

	// Обрабатываем каждое событие
	for _, event := range events {
		if err := a.UpdateTrending(ctx, event); err != nil {
			a.log.Warn("Failed to update trending for event", "skin_id", event.SkinID, "error", err)
		}
	}

	return nil
}
