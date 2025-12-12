package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	_defaultWriteTimeout = 10 * time.Second
	_defaultBatchSize    = 100
	_defaultBatchTimeout = time.Second
)

// Producer - Kafka message producer.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer — создаёт продюсер Kafka и возвращает ошибку в случае неправильной конфигурации.
func NewProducer(brokers []string, topic string, opts ...ProducerOption) (*Producer, error) {
	if len(brokers) == 0 {
		return nil, errors.New("kafka producer: brokers list is empty")
	}

	if topic == "" {
		return nil, errors.New("kafka producer: topic is empty")
	}

	config := &producerConfig{
		writeTimeout: _defaultWriteTimeout,
		batchSize:    _defaultBatchSize,
		batchTimeout: _defaultBatchTimeout,
		compression:  kafka.Snappy,
	}

	for _, opt := range opts {
		opt(config)
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: config.writeTimeout,
		BatchSize:    config.batchSize,
		BatchTimeout: config.batchTimeout,
		Compression:  config.compression,
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}

	// Мини-тест соединения
	err := writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte("init"),
		Value: []byte("init"),
		Time:  time.Now(),
	})

	if err != nil {
		return nil, fmt.Errorf("kafka producer: failed init write: %w", err)
	}

	return &Producer{writer: writer}, nil
}

// WriteMessage - отправить одно сообщение
func (p *Producer) WriteMessage(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("kafka producer - marshal: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("kafka producer - write message: %w", err)
	}

	return nil
}

// WriteMessages - отправить несколько сообщений
func (p *Producer) WriteMessages(ctx context.Context, messages []kafka.Message) error {
	if err := p.writer.WriteMessages(ctx, messages...); err != nil {
		return fmt.Errorf("kafka producer - write messages: %w", err)
	}
	return nil
}

// Close — закрывает продюсер
func (p *Producer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

func (p *Producer) Stats() kafka.WriterStats {
	return p.writer.Stats()
}

// producerConfig - internal options
type producerConfig struct {
	writeTimeout time.Duration
	batchSize    int
	batchTimeout time.Duration
	compression  kafka.Compression
}

// ProducerOption — настройки
type ProducerOption func(*producerConfig)

func WithWriteTimeout(timeout time.Duration) ProducerOption {
	return func(c *producerConfig) {
		c.writeTimeout = timeout
	}
}

func WithBatchSize(size int) ProducerOption {
	return func(c *producerConfig) {
		c.batchSize = size
	}
}

func WithBatchTimeout(timeout time.Duration) ProducerOption {
	return func(c *producerConfig) {
		c.batchTimeout = timeout
	}
}

func WithCompression(compression kafka.Compression) ProducerOption {
	return func(c *producerConfig) {
		c.compression = compression
	}
}

// NewMessage - helper
func NewMessage(key string, value interface{}) (kafka.Message, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("marshal message: %w", err)
	}

	return kafka.Message{
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
	}, nil
}

func NewMessageWithHeaders(key string, value interface{}, headers map[string]string) (kafka.Message, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("marshal message: %w", err)
	}

	var kafkaHeaders []kafka.Header
	for k, v := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{Key: k, Value: []byte(v)})
	}

	return kafka.Message{
		Key:     []byte(key),
		Value:   data,
		Headers: kafkaHeaders,
		Time:    time.Now(),
	}, nil
}
