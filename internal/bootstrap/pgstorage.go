package bootstrap

import (
	"context"
	"log"

	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/internal/storage/db"
	"github.com/kedr891/cs-parser/internal/storage/pgstorage"
	"github.com/kedr891/cs-parser/internal/storage/sharding"
)

func InitPGStorage(cfg *config.Config) *pgstorage.Storage {
	ctx := context.Background()

	if cfg.IsShardingEnabled() {
		manager, err := sharding.NewWeaponShardManagerFromURLs(ctx, cfg.Shard.URLs)
		if err != nil {
			log.Panicf("ошибка инициализации шардинга, %v", err)
		}
		storage, err := pgstorage.NewWithSharding(ctx, manager)
		if err != nil {
			log.Panicf("ошибка инициализации БД с шардингом, %v", err)
		}
		return storage
	}

	pg, err := db.New(cfg.DatabaseURL(), db.MaxPoolSize(10))
	if err != nil {
		log.Panicf("ошибка инициализации БД, %v", err)
	}

	storage, err := pgstorage.New(ctx, pg)
	if err != nil {
		log.Panicf("ошибка инициализации БД, %v", err)
	}

	return storage
}
