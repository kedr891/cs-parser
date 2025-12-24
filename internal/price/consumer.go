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

type Consumer struct {
	consumer      *kafka.Consumer
	alertProducer domain.MessageProducer
	repo          Repository
	analytics     *Analytics
	cache         domain.CacheStorage
	log           domain.Logger
}

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

func (c *Consumer) Start(ctx context.Context) error {
	c.log.Info("Price consumer started")

	handler := kafka.MessageHandlerFunc(func(ctx context.Context, msg kafkago.Message) error {
		return c.handlePriceUpdate(ctx, msg)
	})

	if err := c.consumer.ConsumeWithRetry(ctx, handler, 3); err != nil {
		return fmt.Errorf("consume messages: %w", err)
	}

	return nil
}

func (c *Consumer) handlePriceUpdate(ctx context.Context, msg kafkago.Message) error {
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

	if err := c.savePriceHistory(ctx, &event); err != nil {
		c.log.Error("Failed to save price history", "error", err)
		return fmt.Errorf("save price history: %w", err)
	}
	if err := c.updatePriceCache(ctx, &event); err != nil {
		c.log.Warn("Failed to update cache", "error", err)
	}

	if err := c.analytics.UpdateTrending(ctx, &event); err != nil {
		c.log.Warn("Failed to update trending", "error", err)
	}

	if err := c.processWatchlistAlerts(ctx, &event); err != nil {
		c.log.Error("Failed to process watchlist alerts", "error", err)
	}

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

	if err := c.cache.Set(ctx, cacheKey, string(data), 5*time.Minute); err != nil {
		return fmt.Errorf("set cache: %w", err)
	}

	return nil
}

func (c *Consumer) processWatchlistAlerts(ctx context.Context, event *entity.PriceUpdateEvent) error {
	watchlists, err := c.repo.GetWatchlistsBySkinID(ctx, event.SkinID)
	if err != nil {
		return fmt.Errorf("get watchlists: %w", err)
	}

	if len(watchlists) == 0 {
		return nil
	}

	c.log.Debug("Found watchlists", "count", len(watchlists), "skin_id", event.SkinID)

	for _, wl := range watchlists {
		if !wl.ShouldNotify(event.NewPrice, event.OldPrice) {
			continue
		}

		alertEvent := entity.NewPriceAlertNotification(
			wl.UserID,
			event.SkinID,
			event.MarketHashName,
			event.OldPrice,
			event.NewPrice,
		)

		if wl.TargetPrice != nil {
			alertEvent.TargetPrice = wl.TargetPrice
			alertEvent.NotificationType = entity.TypeTargetReached
		}

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

type ConsumerStats struct {
	Messages   int64     `json:"messages"`
	Bytes      int64     `json:"bytes"`
	Lag        int64     `json:"lag"`
	Offset     int64     `json:"offset"`
	LastUpdate time.Time `json:"last_update"`
}

func (c *Consumer) HandleSkinDiscovered(ctx context.Context, msg kafkago.Message) error {
	var event entity.SkinDiscoveredEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	c.log.Info("New skin discovered",
		"name", event.MarketHashName,
		"price", event.InitialPrice,
	)

	return nil
}
