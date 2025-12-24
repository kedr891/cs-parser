package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kedr891/cs-parser/internal/entity"
	"github.com/kedr891/cs-parser/pkg/kafka"
	"github.com/kedr891/cs-parser/pkg/logger"
	kafkago "github.com/segmentio/kafka-go"
)

type Consumer struct {
	consumer *kafka.Consumer
	service  *Service
	log      *logger.Logger
}

func NewConsumer(
	consumer *kafka.Consumer,
	service *Service,
	log *logger.Logger,
) *Consumer {
	return &Consumer{
		consumer: consumer,
		service:  service,
		log:      log,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	c.log.Info("Notification consumer started")

	handler := kafka.MessageHandlerFunc(func(ctx context.Context, msg kafkago.Message) error {
		return c.handlePriceAlert(ctx, msg)
	})

	if err := c.consumer.ConsumeWithRetry(ctx, handler, 3); err != nil {
		return fmt.Errorf("consume messages: %w", err)
	}

	return nil
}

func (c *Consumer) handlePriceAlert(ctx context.Context, msg kafkago.Message) error {
	var alert entity.PriceAlertNotification
	if err := json.Unmarshal(msg.Value, &alert); err != nil {
		c.log.Error("Failed to unmarshal alert", "error", err)
		return fmt.Errorf("unmarshal alert: %w", err)
	}

	c.log.Debug("Processing price alert",
		"user_id", alert.UserID,
		"skin_id", alert.SkinID,
		"type", alert.NotificationType,
		"price_change", alert.PriceChange,
	)

	alreadySent, err := c.service.WasAlertSent(ctx, alert.UserID, alert.SkinID)
	if err != nil {
		c.log.Warn("Failed to check alert status", "error", err)
	}

	if alreadySent {
		c.log.Debug("Alert already sent recently, skipping",
			"user_id", alert.UserID,
			"skin_id", alert.SkinID,
		)
		return nil
	}

	preferences, err := c.service.GetUserPreferences(ctx, alert.UserID)
	if err != nil {
		c.log.Warn("Failed to get user preferences", "error", err)
		preferences = entity.NewNotificationPreferences(alert.UserID)
	}

	absChange := alert.PriceChange
	if absChange < 0 {
		absChange = -absChange
	}

	if !preferences.ShouldSend(alert.NotificationType, absChange) {
		c.log.Debug("Alert filtered by user preferences",
			"user_id", alert.UserID,
			"type", alert.NotificationType,
			"price_change", alert.PriceChange,
		)
		return nil
	}

	title, message := alert.GenerateMessage()
	notification := entity.NewNotification(
		alert.UserID,
		alert.NotificationType,
		title,
		message,
	)

	notification.SetPriority(alert.GetPriority())

	notification.AddData("skin_id", alert.SkinID.String())
	notification.AddData("old_price", alert.OldPrice)
	notification.AddData("current_price", alert.CurrentPrice)
	notification.AddData("price_change", alert.PriceChange)
	if alert.SkinImageURL != "" {
		notification.AddData("skin_image_url", alert.SkinImageURL)
	}

	if err := c.service.CreateNotification(ctx, notification); err != nil {
		c.log.Error("Failed to create notification", "error", err)
		return fmt.Errorf("create notification: %w", err)
	}

	if err := c.service.SendNotification(ctx, notification, preferences); err != nil {
		c.log.Error("Failed to send notification", "error", err)
	}

	if err := c.service.TrackAlertSent(ctx, alert.UserID, alert.SkinID); err != nil {
		c.log.Warn("Failed to track alert", "error", err)
	}

	c.log.Info("Price alert processed successfully",
		"user_id", alert.UserID,
		"skin_id", alert.SkinID,
		"notification_id", notification.ID,
	)

	return nil
}

func (c *Consumer) GetStats(ctx context.Context) *ConsumerStats {
	stats := c.consumer.Stats()
	lag := c.consumer.Lag()

	return &ConsumerStats{
		Messages: stats.Messages,
		Bytes:    stats.Bytes,
		Lag:      lag,
		Offset:   stats.Offset,
	}
}

type ConsumerStats struct {
	Messages int64 `json:"messages"`
	Bytes    int64 `json:"bytes"`
	Lag      int64 `json:"lag"`
	Offset   int64 `json:"offset"`
}
