package bootstrap

import (
	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/pkg/redis"
)

func InitCache(cfg *config.Config) (*redis.Redis, error) {
	return redis.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
}
