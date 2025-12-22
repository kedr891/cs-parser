package price

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kedr891/cs-parser/internal/domain"
	"github.com/kedr891/cs-parser/internal/entity"
	"github.com/kedr891/cs-parser/pkg/kafka"
	kafkago "github.com/segmentio/kafka-go"
)

// Consumer - консьюмер событий изменения цен
type Consumer struct {
	consumer      *kafka.Consumer
	alertProducer domain.MessageProducer
	repo          Repository
	analytics     *Analytics
	cache         domain.CacheStorage
	log           domain.Logger
}

// NewConsumer - создать консьюмер
func NewConsumer(
	consumer *kafka.Consumer,
	alertProducer domain.MessageProducer,
	repo Repository,
	analytics *Analytics,
	cache domain.CacheStorage,
	log domain.Logger,
) *Consumer {
	return &Consumer{
		consumer:      consumer,
		alertProducer: alertProducer,
		repo:          repo,
		analytics:     analytics,
		cache:         cache,
		log:           log,
	}
}

// Start - запустить обработку сообщений
func (c *Consumer) Start(ctx context.Context) error {
	c.log.Info("Price consumer started")

	handler := kafka.MessageHandlerFunc(func(ctx context.Context, msg kafkago.Message) error {
		return c.handlePriceUpdate(ctx, msg)
	})

	// Запуск с retry (максимум 3 попытки)
	if err := c.consumer.ConsumeWithRetry(ctx, handler, 3); err != nil {
		return fmt.Errorf("consume messages: %w", err)
	}

	return nil
}

// handlePriceUpdate - обработать событие обновления цены
func (c *Consumer) handlePriceUpdate(ctx context.Context, msg kafkago.Message) error {
	// Десериализация события
	var event entity.PriceUpdateEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		c.log.Error("Failed to unmarshal price event", "error", err)
		return fmt.Errorf("unmarshal event: %w", err)
	}

	c.log.Debug("Processing price update",
		"skin_id", event.SkinID,
		"old_price", event.OldPrice,
		"new_price", event.NewPrice,
		"change", event.PriceChange,
	)

	// 1. Сохранить историю цены
	if err := c.savePriceHistory(ctx, &event); err != nil {
		c.log.Error("Failed to save price history", "error", err)
		return fmt.Errorf("save price history: %w", err)
	}

	// 2. Обновить кэш в Redis
	if err := c.updatePriceCache(ctx, &event); err != nil {
		c.log.Warn("Failed to update cache", "error", err)
		// Не возвращаем ошибку, так как это не критично
	}

	// 3. Обновить аналитику (trending, статистика)
	if err := c.analytics.UpdateTrending(ctx, &event); err != nil {
		c.log.Warn("Failed to update trending", "error", err)
	}

	// 4. Проверить watchlist и отправить алерты
	if err := c.processWatchlistAlerts(ctx, &event); err != nil {
		c.log.Error("Failed to process watchlist alerts", "error", err)
		// Продолжаем, чтобы не блокировать обработку
	}

	// 5. Обновить market overview
	if event.IsSignificantChange() {
		if err := c.analytics.InvalidateMarketOverview(ctx); err != nil {
			c.log.Warn("Failed to invalidate market overview", "error", err)
		}
	}

	c.log.Info("Price update processed successfully",
		"skin_id", event.SkinID,
		"price_change", event.PriceChange,
	)

	return nil
}

// savePriceHistory - сохранить историю цены в БД
func (c *Consumer) savePriceHistory(ctx context.Context, event *entity.PriceUpdateEvent) error {
	history := entity.NewPriceHistory(
		event.SkinID,
		event.NewPrice,
		event.Source,
		event.Volume24h,
	)

	if err := c.repo.SavePriceHistory(ctx, history); err != nil {
		return fmt.Errorf("save price history: %w", err)
	}

	return nil
}

// updatePriceCache - обновить кэш цены в Redis
func (c *Consumer) updatePriceCache(ctx context.Context, event *entity.PriceUpdateEvent) error {
	cacheKey := fmt.Sprintf("skin:price:%s", event.SkinID.String())

	cacheData := map[string]interface{}{
		"price":      event.NewPrice,
		"currency":   event.Currency,
		"updated_at": event.Timestamp.Unix(),
		"source":     event.Source,
		"volume":     event.Volume24h,
	}

	data, err := json.Marshal(cacheData)
	if err != nil {
		return fmt.Errorf("marshal cache data: %w", err)
	}

	// TTL 5 минут (до следующего парсинга)
	if err := c.cache.Set(ctx, cacheKey, string(data), 5*time.Minute); err != nil {
		return fmt.Errorf("set cache: %w", err)
	}

	return nil
}

// processWatchlistAlerts - обработать алерты для watchlist
func (c *Consumer) processWatchlistAlerts(ctx context.Context, event *entity.PriceUpdateEvent) error {
	// Получить всех пользователей, отслеживающих этот скин
	watchlists, err := c.repo.GetWatchlistsBySkinID(ctx, event.SkinID)
	if err != nil {
		return fmt.Errorf("get watchlists: %w", err)
	}

	if len(watchlists) == 0 {
		return nil // никто не отслеживает
	}

	c.log.Debug("Found watchlists", "count", len(watchlists), "skin_id", event.SkinID)

	// Проверить условия для каждого watchlist
	for _, wl := range watchlists {
		if !wl.ShouldNotify(event.NewPrice, event.OldPrice) {
			continue
		}

		// Создать событие алерта
		alertEvent := entity.NewPriceAlertNotification(
			wl.UserID,
			event.SkinID,
			event.MarketHashName,
			event.OldPrice,
			event.NewPrice,
		)

		// Добавить target price если есть
		if wl.TargetPrice != nil {
			alertEvent.TargetPrice = wl.TargetPrice
			alertEvent.NotificationType = entity.TypeTargetReached
		}

		// Отправить в Kafka для notification service
		if err := c.alertProducer.WriteMessage(ctx, wl.UserID.String(), alertEvent); err != nil {
			c.log.Error("Failed to send alert",
				"user_id", wl.UserID,
				"skin_id", event.SkinID,
				"error", err,
			)
			continue
		}

		c.log.Info("Alert sent",
			"user_id", wl.UserID,
			"skin_id", event.SkinID,
			"type", alertEvent.NotificationType,
		)
	}

	return nil
}

// GetStats - получить статистику консьюмера
func (c *Consumer) GetStats(ctx context.Context) *ConsumerStats {
	stats := c.consumer.Stats()
	lag := c.consumer.Lag()

	return &ConsumerStats{
		Messages:   stats.Messages,
		Bytes:      stats.Bytes,
		Lag:        lag,
		Offset:     stats.Offset,
		LastUpdate: time.Now(),
	}
}

// ConsumerStats - статистика консьюмера
type ConsumerStats struct {
	Messages   int64     `json:"messages"`
	Bytes      int64     `json:"bytes"`
	Lag        int64     `json:"lag"`
	Offset     int64     `json:"offset"`
	LastUpdate time.Time `json:"last_update"`
}

// HandleSkinDiscovered - обработать событие обнаружения нового скина
func (c *Consumer) HandleSkinDiscovered(ctx context.Context, msg kafkago.Message) error {
	var event entity.SkinDiscoveredEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	c.log.Info("New skin discovered",
		"name", event.MarketHashName,
		"price", event.InitialPrice,
	)

	// Можно добавить дополнительную логику:
	// - Уведомление администраторов
	// - Автоматическое добавление в trending
	// - Инициализация аналитики для нового скина

	return nil
}
