package kafka

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

func NewWriter(brokers []string, topic string) *kafka.Writer {
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
}

func NewReader(brokers []string, topic, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
}

func CloseWriter(w *kafka.Writer) {
	if err := w.Close(); err != nil {
		log.Printf("kafka - CloseWriter error: %v", err)
	}
}

func CloseReader(r *kafka.Reader) {
	if err := r.Close(); err != nil {
		log.Printf("kafka - CloseReader error: %v", err)
	}
}

func SendMessage(ctx context.Context, w *kafka.Writer, key, value string) error {
	msg := kafka.Message{
		Key:   []byte(key),
		Value: []byte(value),
		Time:  time.Now(),
	}
	return w.WriteMessages(ctx, msg)
}
