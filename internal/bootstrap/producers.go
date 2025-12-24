package bootstrap

import (
	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/pkg/kafka"
)

type Producers struct {
	PriceUpdated   *kafka.Producer
	SkinDiscovered *kafka.Producer
	PriceAlert     *kafka.Producer
}

func InitProducers(cfg *config.Config) (*Producers, error) {
	priceProducer, err := kafka.NewProducer(
		cfg.Kafka.Brokers,
		cfg.Kafka.TopicPriceUpdated,
		kafka.WithBatchSize(50),
	)
	if err != nil {
		return nil, err
	}

	discoveryProducer, err := kafka.NewProducer(
		cfg.Kafka.Brokers,
		cfg.Kafka.TopicSkinDiscovered,
		kafka.WithBatchSize(20),
	)
	if err != nil {
		priceProducer.Close()
		return nil, err
	}

	alertProducer, err := kafka.NewProducer(
		cfg.Kafka.Brokers,
		cfg.Kafka.TopicPriceAlert,
	)
	if err != nil {
		priceProducer.Close()
		discoveryProducer.Close()
		return nil, err
	}

	return &Producers{
		PriceUpdated:   priceProducer,
		SkinDiscovered: discoveryProducer,
		PriceAlert:     alertProducer,
	}, nil
}

func (p *Producers) Close() {
	if p.PriceUpdated != nil {
		p.PriceUpdated.Close()
	}
	if p.SkinDiscovered != nil {
		p.SkinDiscovered.Close()
	}
	if p.PriceAlert != nil {
		p.PriceAlert.Close()
	}
}
