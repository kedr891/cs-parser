package bootstrap

import (
	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/pkg/kafka"
)

func InitKafkaProducer(cfg *config.Config, topic string, opts ...kafka.ProducerOption) (*kafka.Producer, error) {
	return kafka.NewProducer(cfg.Kafka.Brokers, topic, opts...)
}

func InitKafkaConsumer(cfg *config.Config, topic, groupID string) *kafka.Consumer {
	return kafka.NewConsumer(cfg.Kafka.Brokers, topic, groupID)
}
