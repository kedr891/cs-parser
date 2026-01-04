package bootstrap

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"

	"github.com/kedr891/cs-parser/config"
	skinservice "github.com/kedr891/cs-parser/internal/services/skinService"
)

func InitCache(cfg *config.Config) (*skinservice.SkinsCache, func()) {
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port)
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Panicf("ошибка инициализации кеша, %v", err)
	}

	c := skinservice.NewSkinsCache(client, 1800)
	closeFn := func() {
		client.Close()
	}
	return c, closeFn
}
