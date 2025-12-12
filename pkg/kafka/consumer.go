package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	_defaultMaxWait        = 10 * time.Second
	_defaultMinBytes       = 1
	_defaultMaxBytes       = 10e6 // 10MB
	_defaultCommitInterval = time.Second
)

// Consumer -.
type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer -.
func NewConsumer(brokers []string, topic, groupID string, opts ...ConsumerOption) *Consumer {
	config := &consumerConfig{
		maxWait:        _defaultMaxWait,
		minBytes:       _defaultMinBytes,
		maxBytes:       _defaultMaxBytes,
		commitInterval: _defaultCommitInterval,
	}

	for _, opt := range opts {
		opt(config)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       config.minBytes,
		MaxBytes:       config.maxBytes,
		MaxWait:        config.maxWait,
		CommitInterval: config.commitInterval,
		StartOffset:    kafka.LastOffset,
		// Для at-least-once семантики
		QueueCapacity: 100,
	})

	return &Consumer{
		reader: reader,
	}
}

// ReadMessage - прочитать одно сообщение (блокирующая операция)
func (c *Consumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("kafka consumer - read message: %w", err)
	}
	return msg, nil
}

// FetchMessage - получить сообщение без автоматического коммита
func (c *Consumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	msg, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return kafka.Message{}, fmt.Errorf("kafka consumer - fetch message: %w", err)
	}
	return msg, nil
}

// CommitMessages - закоммитить сообщения вручную
func (c *Consumer) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	if err := c.reader.CommitMessages(ctx, msgs...); err != nil {
		return fmt.Errorf("kafka consumer - commit messages: %w", err)
	}
	return nil
}

// Close -.
func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

// Stats - статистика консьюмера
func (c *Consumer) Stats() kafka.ReaderStats {
	return c.reader.Stats()
}

// Lag - отставание консьюмера
func (c *Consumer) Lag() int64 {
	return c.reader.Lag()
}

// consumerConfig -.
type consumerConfig struct {
	maxWait        time.Duration
	minBytes       int
	maxBytes       int
	commitInterval time.Duration
}

// ConsumerOption -.
type ConsumerOption func(*consumerConfig)

// WithMaxWait -.
func WithMaxWait(duration time.Duration) ConsumerOption {
	return func(c *consumerConfig) {
		c.maxWait = duration
	}
}

// WithMinBytes -.
func WithMinBytes(bytes int) ConsumerOption {
	return func(c *consumerConfig) {
		c.minBytes = bytes
	}
}

// WithMaxBytes -.
func WithMaxBytes(bytes int) ConsumerOption {
	return func(c *consumerConfig) {
		c.maxBytes = bytes
	}
}

// WithCommitInterval -.
func WithCommitInterval(interval time.Duration) ConsumerOption {
	return func(c *consumerConfig) {
		c.commitInterval = interval
	}
}

// --- Message Handler Interface ---

// MessageHandler - интерфейс обработчика сообщений
type MessageHandler interface {
	Handle(ctx context.Context, msg kafka.Message) error
}

// MessageHandlerFunc - функция-обработчик
type MessageHandlerFunc func(ctx context.Context, msg kafka.Message) error

// Handle -.
func (f MessageHandlerFunc) Handle(ctx context.Context, msg kafka.Message) error {
	return f(ctx, msg)
}

// --- Consumer Loop ---

// Consume - запустить цикл обработки сообщений
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.ReadMessage(ctx)
			if err != nil {
				return fmt.Errorf("read message: %w", err)
			}

			if err := handler.Handle(ctx, msg); err != nil {
				// Логируем ошибку, но продолжаем обработку
				// В продакшене здесь может быть retry логика или DLQ
				return fmt.Errorf("handle message: %w", err)
			}
		}
	}
}

// ConsumeWithRetry - обработка с ручным коммитом и retry
func (c *Consumer) ConsumeWithRetry(ctx context.Context, handler MessageHandler, maxRetries int) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.FetchMessage(ctx)
			if err != nil {
				return fmt.Errorf("fetch message: %w", err)
			}

			// Попытки обработки с retry
			var lastErr error
			for attempt := 0; attempt <= maxRetries; attempt++ {
				if err := handler.Handle(ctx, msg); err != nil {
					lastErr = err
					time.Sleep(time.Second * time.Duration(attempt+1)) // exponential backoff
					continue
				}

				// Успешная обработка - коммитим
				if err := c.CommitMessages(ctx, msg); err != nil {
					return fmt.Errorf("commit message: %w", err)
				}
				lastErr = nil
				break
			}

			if lastErr != nil {
				// Все попытки исчерпаны
				// В продакшене здесь отправка в DLQ
				return fmt.Errorf("message processing failed after %d retries: %w", maxRetries, lastErr)
			}
		}
	}
}

// --- Helper для десериализации ---

// UnmarshalMessage - десериализовать сообщение в структуру
func UnmarshalMessage(msg kafka.Message, v interface{}) error {
	if err := json.Unmarshal(msg.Value, v); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}
	return nil
}

// GetHeader - получить значение header
func GetHeader(msg kafka.Message, key string) (string, bool) {
	for _, h := range msg.Headers {
		if h.Key == key {
			return string(h.Value), true
		}
	}
	return "", false
}
