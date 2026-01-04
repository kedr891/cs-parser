package priceupdateconsumer

import (
	"context"

	"github.com/kedr891/cs-parser/internal/models"
)

type priceUpdateProcessor interface {
	Handle(ctx context.Context, event *models.PriceUpdateEvent) error
}

type PriceUpdateConsumer struct {
	processor   priceUpdateProcessor
	kafkaBroker []string
	topicName   string
	groupID     string
}

func NewPriceUpdateConsumer(
	processor priceUpdateProcessor,
	kafkaBroker []string,
	topicName string,
	groupID string,
) *PriceUpdateConsumer {
	return &PriceUpdateConsumer{
		processor:   processor,
		kafkaBroker: kafkaBroker,
		topicName:   topicName,
		groupID:     groupID,
	}
}
