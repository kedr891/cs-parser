package bootstrap

import (
	"fmt"

	"github.com/kedr891/cs-parser/config"
	priceupdateconsumer "github.com/kedr891/cs-parser/internal/consumer/price_update_consumer"
	priceupdateprocessor "github.com/kedr891/cs-parser/internal/services/processors/price_update_processor"
)

func InitPriceUpdateConsumer(
	cfg *config.Config,
	processor *priceupdateprocessor.PriceUpdateProcessor,
) *priceupdateconsumer.PriceUpdateConsumer {
	kafkaBrokers := []string{fmt.Sprintf("%s:%d", cfg.Kafka.Host, cfg.Kafka.Port)}
	return priceupdateconsumer.NewPriceUpdateConsumer(
		processor,
		kafkaBrokers,
		cfg.Kafka.TopicPriceUpdated,
		cfg.Kafka.GroupPriceConsumer,
	)
}
