package priceupdateconsumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/kedr891/cs-parser/internal/models"
	"github.com/segmentio/kafka-go"
)

func (c *PriceUpdateConsumer) Consume(ctx context.Context) error {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:           c.kafkaBroker,
		GroupID:           c.groupID,
		Topic:             c.topicName,
		HeartbeatInterval: 3 * time.Second,
		SessionTimeout:    30 * time.Second,
	})
	defer r.Close()

	slog.Info("PriceUpdateConsumer started", "topic", c.topicName, "group", c.groupID)

	for {
		select {
		case <-ctx.Done():
			slog.Info("PriceUpdateConsumer stopped")
			return ctx.Err()
		default:
			msg, err := r.ReadMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return err
				}
				slog.Error("PriceUpdateConsumer.Consume error", "error", err.Error())
				continue
			}

			var event models.PriceUpdateEvent
			err = json.Unmarshal(msg.Value, &event)
			if err != nil {
				slog.Error("Failed to unmarshal price update event", "error", err)
				continue
			}

			err = c.processor.Handle(ctx, &event)
			if err != nil {
				slog.Error("Failed to handle price update", "error", err, "skin_id", event.SkinID)
			}
		}
	}
}
