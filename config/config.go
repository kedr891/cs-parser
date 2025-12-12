package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/caarlos0/env/v11"
)

type (
	// Config -.
	Config struct {
		App      App
		HTTP     HTTP
		Log      Log
		PG       PG
		Redis    Redis
		Kafka    Kafka
		RabbitMQ RabbitMQ
		NATS     NATS
		Parser   Parser
		JWT      JWT
		Metrics  Metrics
		Swagger  Swagger
	}

	// App -.
	App struct {
		Name    string `env:"APP_NAME" envDefault:"cs-parser"`
		Version string `env:"APP_VERSION" envDefault:"1.0.0"`
	}

	// HTTP -.
	HTTP struct {
		Port string `env:"HTTP_PORT" envDefault:"8080"`
	}

	// Log -.
	Log struct {
		Level string `env:"LOG_LEVEL" envDefault:"info"`
	}

	// PG -.
	PG struct {
		PoolMax int    `env:"PG_POOL_MAX" envDefault:"10"`
		URL     string `env:"PG_URL,required"`
	}

	// Redis -.
	Redis struct {
		Addr     string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
		Password string `env:"REDIS_PASSWORD"`
		DB       int    `env:"REDIS_DB" envDefault:"0"`
	}

	// Kafka -.
	Kafka struct {
		Brokers             []string `env:"KAFKA_BROKERS" envSeparator:","`
		TopicPriceUpdated   string   `env:"KAFKA_TOPIC_PRICE_UPDATED" envDefault:"skin.price.updated"`
		TopicSkinDiscovered string   `env:"KAFKA_TOPIC_SKIN_DISCOVERED" envDefault:"skin.discovered"`
		TopicPriceAlert     string   `env:"KAFKA_TOPIC_PRICE_ALERT" envDefault:"notification.price_alert"`
		GroupPriceConsumer  string   `env:"KAFKA_GROUP_PRICE_CONSUMER" envDefault:"price-consumer-group"`
		GroupNotification   string   `env:"KAFKA_GROUP_NOTIFICATION" envDefault:"notification-consumer-group"`
	}

	// RabbitMQ -.
	RabbitMQ struct {
		URL            string `env:"RMQ_URL,required"`
		ServerExchange string `env:"RMQ_SERVER_EXCHANGE" envDefault:"cs_parser_rpc"`
	}

	// NATS -.
	NATS struct {
		URL            string `env:"NATS_URL,required"`
		ServerExchange string `env:"NATS_SERVER_EXCHANGE" envDefault:"cs_parser_rpc"`
	}

	// Parser -.
	Parser struct {
		IntervalMinutes    int `env:"PARSER_INTERVAL_MINUTES" envDefault:"5"`
		RateLimitPerMinute int `env:"PARSER_RATE_LIMIT_PER_MINUTE" envDefault:"60"`
	}

	// JWT -.
	JWT struct {
		Secret          string `env:"JWT_SECRET,required"`
		ExpirationHours int    `env:"JWT_EXPIRATION_HOURS" envDefault:"168"` // 7 days
	}

	// Metrics -.
	Metrics struct {
		Enabled bool `env:"METRICS_ENABLED" envDefault:"true"`
	}

	// Swagger -.
	Swagger struct {
		Enabled bool `env:"SWAGGER_ENABLED" envDefault:"false"`
	}
)

// NewConfig returns app config.
func NewConfig() (*Config, error) {
	cfg := &Config{}

	// Custom parser for Kafka brokers
	opts := env.Options{
		FuncMap: map[reflect.Type]env.ParserFunc{
			reflect.TypeOf([]string{}): func(v string) (interface{}, error) {
				if v == "" {
					return []string{}, nil
				}
				return strings.Split(v, ","), nil
			},
		},
	}

	if err := env.ParseWithOptions(cfg, opts); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	// Validate Kafka brokers
	if len(cfg.Kafka.Brokers) == 0 {
		cfg.Kafka.Brokers = []string{"localhost:9092"}
	}

	return cfg, nil
}
