package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App     AppConfig     `yaml:"app"`
	HTTP    HTTPConfig    `yaml:"http"`
	Log     LogConfig     `yaml:"log"`
	PG      PGConfig      `yaml:"pg"`
	Shard   ShardConfig   `yaml:"shard"`
	Redis   RedisConfig   `yaml:"redis"`
	Kafka   KafkaConfig   `yaml:"kafka"`
	Parser  ParserConfig  `yaml:"parser"`
	JWT     JWTConfig     `yaml:"jwt"`
	Metrics MetricsConfig `yaml:"metrics"`
	Swagger SwaggerConfig `yaml:"swagger"`
}

type AppConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type HTTPConfig struct {
	Port string `yaml:"port"`
}

type LogConfig struct {
	Level string `yaml:"level"`
}

type PGConfig struct {
	PoolMax int    `yaml:"poolMax"`
	URL     string `yaml:"url"`
}

type ShardConfig struct {
	Enabled bool     `yaml:"enabled"`
	URLs    []string `yaml:"urls"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type KafkaConfig struct {
	Brokers             []string `yaml:"brokers"`
	TopicPriceUpdated   string   `yaml:"topicPriceUpdated"`
	TopicSkinDiscovered string   `yaml:"topicSkinDiscovered"`
	TopicPriceAlert     string   `yaml:"topicPriceAlert"`
	GroupPriceConsumer  string   `yaml:"groupPriceConsumer"`
	GroupNotification   string   `yaml:"groupNotification"`
}

type ParserConfig struct {
	IntervalMinutes    int `yaml:"intervalMinutes"`
	RateLimitPerMinute int `yaml:"rateLimitPerMinute"`
}

type JWTConfig struct {
	Secret          string `yaml:"secret"`
	ExpirationHours int    `yaml:"expirationHours"`
}

type MetricsConfig struct {
	Enabled bool `yaml:"enabled"`
}

type SwaggerConfig struct {
	Enabled bool `yaml:"enabled"`
}

func NewConfig() (*Config, error) {
	return LoadConfig("")
}

func LoadConfig(filename string) (*Config, error) {
	cfg := &Config{}

	if strings.TrimSpace(filename) != "" {
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
		}
	}

	cfg.applyEnv()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) applyEnv() {
	if env := strings.TrimSpace(os.Getenv("APP_NAME")); env != "" {
		c.App.Name = env
	}
	if env := strings.TrimSpace(os.Getenv("APP_VERSION")); env != "" {
		c.App.Version = env
	}

	if env := strings.TrimSpace(os.Getenv("HTTP_PORT")); env != "" {
		c.HTTP.Port = env
	}

	if env := strings.TrimSpace(os.Getenv("LOG_LEVEL")); env != "" {
		c.Log.Level = env
	}

	if env := strings.TrimSpace(os.Getenv("PG_POOL_MAX")); env != "" {
		if poolMax, err := strconv.Atoi(env); err == nil {
			c.PG.PoolMax = poolMax
		}
	}
	if env := strings.TrimSpace(os.Getenv("PG_URL")); env != "" {
		c.PG.URL = env
	}

	if env := strings.TrimSpace(os.Getenv("SHARD_ENABLED")); env != "" {
		c.Shard.Enabled = env == "true" || env == "1"
	}
	if env := strings.TrimSpace(os.Getenv("SHARD_URLS")); env != "" {
		parts := strings.Split(env, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		c.Shard.URLs = parts
	}

	if env := strings.TrimSpace(os.Getenv("REDIS_ADDR")); env != "" {
		c.Redis.Addr = env
	}
	if env := strings.TrimSpace(os.Getenv("REDIS_PASSWORD")); env != "" {
		c.Redis.Password = env
	}
	if env := strings.TrimSpace(os.Getenv("REDIS_DB")); env != "" {
		if db, err := strconv.Atoi(env); err == nil {
			c.Redis.DB = db
		}
	}

	if env := strings.TrimSpace(os.Getenv("KAFKA_BROKERS")); env != "" {
		parts := strings.Split(env, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		c.Kafka.Brokers = parts
	}
	if env := strings.TrimSpace(os.Getenv("KAFKA_TOPIC_PRICE_UPDATED")); env != "" {
		c.Kafka.TopicPriceUpdated = env
	}
	if env := strings.TrimSpace(os.Getenv("KAFKA_TOPIC_SKIN_DISCOVERED")); env != "" {
		c.Kafka.TopicSkinDiscovered = env
	}
	if env := strings.TrimSpace(os.Getenv("KAFKA_TOPIC_PRICE_ALERT")); env != "" {
		c.Kafka.TopicPriceAlert = env
	}
	if env := strings.TrimSpace(os.Getenv("KAFKA_GROUP_PRICE_CONSUMER")); env != "" {
		c.Kafka.GroupPriceConsumer = env
	}
	if env := strings.TrimSpace(os.Getenv("KAFKA_GROUP_NOTIFICATION")); env != "" {
		c.Kafka.GroupNotification = env
	}

	if env := strings.TrimSpace(os.Getenv("PARSER_INTERVAL_MINUTES")); env != "" {
		if interval, err := strconv.Atoi(env); err == nil {
			c.Parser.IntervalMinutes = interval
		}
	}
	if env := strings.TrimSpace(os.Getenv("PARSER_RATE_LIMIT_PER_MINUTE")); env != "" {
		if rateLimit, err := strconv.Atoi(env); err == nil {
			c.Parser.RateLimitPerMinute = rateLimit
		}
	}

	if env := strings.TrimSpace(os.Getenv("JWT_SECRET")); env != "" {
		c.JWT.Secret = env
	}
	if env := strings.TrimSpace(os.Getenv("JWT_EXPIRATION_HOURS")); env != "" {
		if hours, err := strconv.Atoi(env); err == nil {
			c.JWT.ExpirationHours = hours
		}
	}

	if env := strings.TrimSpace(os.Getenv("METRICS_ENABLED")); env != "" {
		c.Metrics.Enabled = env == "true" || env == "1"
	}

	if env := strings.TrimSpace(os.Getenv("SWAGGER_ENABLED")); env != "" {
		c.Swagger.Enabled = env == "true" || env == "1"
	}
}

func (c *Config) validate() error {
	if !c.IsShardingEnabled() && c.PG.URL == "" {
		return fmt.Errorf("PG_URL is required when sharding is disabled")
	}

	if c.IsShardingEnabled() && len(c.Shard.URLs) == 0 {
		return fmt.Errorf("SHARD_URLS is required when sharding is enabled")
	}

	if len(c.Kafka.Brokers) == 0 {
		c.Kafka.Brokers = []string{"localhost:9092"}
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	return nil
}

func (c *Config) GetShardURLs() []string {
	if c.Shard.Enabled && len(c.Shard.URLs) > 0 {
		return c.Shard.URLs
	}
	return []string{c.PG.URL}
}

func (c *Config) IsShardingEnabled() bool {
	return c.Shard.Enabled && len(c.Shard.URLs) > 1
}
