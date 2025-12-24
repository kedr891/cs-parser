package bootstrap

import (
	"context"

	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/pkg/postgres"
)

func InitPGStorage(ctx context.Context, cfg *config.Config) (*postgres.Postgres, error) {
	pg, err := postgres.New(cfg.PG.URL, postgres.MaxPoolSize(cfg.PG.PoolMax))
	if err != nil {
		return nil, err
	}
	return pg, nil
}

func InitShardManager(cfg *config.Config) (*postgres.ShardManager, error) {
	shardManager, err := postgres.NewShardManager(
		cfg.Shard.URLs,
		postgres.WithShardMaxPoolSize(cfg.PG.PoolMax),
	)
	if err != nil {
		return nil, err
	}
	return shardManager, nil
}
