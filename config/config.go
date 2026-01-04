package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Shard    ShardConfig    `yaml:"shard"`
	Redis    RedisConfig    `yaml:"redis"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	GRPC     GRPCConfig     `yaml:"grpc"`
	Gateway  GatewayConfig  `yaml:"gateway"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DBName   string `yaml:"name"`
	SSLMode  string `yaml:"ssl_mode"`
}

type ShardConfig struct {
	Enabled bool     `yaml:"enabled"`
	URLs    []string `yaml:"urls"`
}

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type KafkaConfig struct {
	Host                string `yaml:"host"`
	Port                int    `yaml:"port"`
	TopicPriceUpdated   string `yaml:"topicPriceUpdated"`
	TopicSkinDiscovered string `yaml:"topicSkinDiscovered"`
	TopicPriceAlert     string `yaml:"topicPriceAlert"`
	GroupPriceConsumer  string `yaml:"groupPriceConsumer"`
}

type GRPCConfig struct {
	Port string `yaml:"port"`
}

type GatewayConfig struct {
	Port        string `yaml:"port"`
	SwaggerPath string `yaml:"swaggerPath"`
}

func LoadConfig(filename string) (*Config, error) {
	if strings.TrimSpace(filename) == "" {
		return nil, fmt.Errorf("config filename is required")
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &config, nil
}

func (c *Config) IsShardingEnabled() bool {
	return c.Shard.Enabled && len(c.Shard.URLs) == 3
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.Username,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}
