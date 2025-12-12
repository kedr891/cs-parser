package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cs-parser/internal/entity"
	"github.com/cs-parser/pkg/kafka"
	"github.com/cs-parser/pkg/logger"
	kafkago "github.com/segmentio/kafka-go"
)

// Consumer - консьюмер уведомлений
type Consumer struct {
	consumer *kafka.Consumer
	service  *Service
	log      *logger.Logger
}

// NewConsumer - создать консьюмер уведомлений
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

// Start - запустить обработку уведомлений
func (c *Consumer) Start(ctx context.Context) error {
	c.log.Info("Notification consumer started")

	handler := kafka.MessageHandlerFunc(func(ctx context.Context, msg kafkago.Message) error {
		return c.handlePriceAlert(ctx, msg)
	})

	// Запуск с retry (максимум 3 попытки)
	if err := c.consumer.ConsumeWithRetry(ctx, handler, 3); err != nil {
		return fmt.Errorf("consume messages: %w", err)
	}

	return nil
}

// handlePriceAlert - обработать событие алерта цены
func (c *Consumer) handlePriceAlert(ctx context.Context, msg kafkago.Message) error {
	// Десериализация события
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

	// Проверить, не отправляли ли мы уже этот алерт недавно
	alreadySent, err := c.service.WasAlertSent(ctx, alert.UserID, alert.SkinID)
	if err != nil {
		c.log.Warn("Failed to check alert status", "error", err)
	}

	if alreadySent {
		c.log.Debug("Alert already sent recently, skipping",
			"user_id", alert.UserID,
			"skin_id", alert.SkinID,
		)
		return nil // Не возвращаем ошибку, просто пропускаем
	}

	// Получить настройки уведомлений пользователя
	preferences, err := c.service.GetUserPreferences(ctx, alert.UserID)
	if err != nil {
		c.log.Warn("Failed to get user preferences", "error", err)
		// Продолжаем с настройками по умолчанию
		preferences = entity.NewNotificationPreferences(alert.UserID)
	}

	// Проверить, нужно ли отправлять уведомление
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

	// Создать уведомление
	title, message := alert.GenerateMessage()
	notification := entity.NewNotification(
		alert.UserID,
		alert.NotificationType,
		title,
		message,
	)

	// Установить приоритет
	notification.SetPriority(alert.GetPriority())

	// Добавить данные
	notification.AddData("skin_id", alert.SkinID.String())
	notification.AddData("old_price", alert.OldPrice)
	notification.AddData("current_price", alert.CurrentPrice)
	notification.AddData("price_change", alert.PriceChange)
	if alert.SkinImageURL != "" {
		notification.AddData("skin_image_url", alert.SkinImageURL)
	}

	// Сохранить уведомление в БД
	if err := c.service.CreateNotification(ctx, notification); err != nil {
		c.log.Error("Failed to create notification", "error", err)
		return fmt.Errorf("create notification: %w", err)
	}

	// Отправить уведомление через активные каналы
	if err := c.service.SendNotification(ctx, notification, preferences); err != nil {
		c.log.Error("Failed to send notification", "error", err)
		// Не возвращаем ошибку, так как уведомление уже сохранено
	}

	// Отметить, что алерт отправлен
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

// GetStats - получить статистику консьюмера
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

// ConsumerStats - статистика консьюмера
type ConsumerStats struct {
	Messages int64 `json:"messages"`
	Bytes    int64 `json:"bytes"`
	Lag      int64 `json:"lag"`
	Offset   int64 `json:"offset"`
}
