package pgstorage

import (
	"context"
	"embed"
	"fmt"
	"sort"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"

	"github.com/kedr891/cs-parser/internal/storage/db"
	"github.com/kedr891/cs-parser/internal/storage/sharding"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Storage struct {
	pg      *db.Postgres
	shards  *sharding.WeaponShardManager
	builder squirrel.StatementBuilderType
}

func New(ctx context.Context, pg *db.Postgres) (*Storage, error) {
	storage := &Storage{
		pg:      pg,
		builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}

	if err := storage.runMigrations(ctx); err != nil {
		return nil, fmt.Errorf("ошибка применения миграций: %w", err)
	}

	return storage, nil
}

func NewWithSharding(ctx context.Context, manager *sharding.WeaponShardManager) (*Storage, error) {
	if manager == nil || manager.ShardCount() == 0 {
		return nil, fmt.Errorf("шард менеджер не настроен")
	}

	storage := &Storage{
		shards:  manager,
		builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}

	if err := storage.runMigrations(ctx); err != nil {
		manager.Close()
		return nil, fmt.Errorf("ошибка применения миграций: %w", err)
	}

	return storage, nil
}

func (s *Storage) Close() {
	if s == nil {
		return
	}
	if s.pg != nil {
		s.pg.Close()
	}
	if s.shards != nil {
		s.shards.Close()
	}
}

func (s *Storage) HasSharding() bool {
	return s.shards != nil
}

func (s *Storage) GetPG() *db.Postgres {
	return s.pg
}

func (s *Storage) GetShards() *sharding.WeaponShardManager {
	return s.shards
}

func (s *Storage) execQuery(ctx context.Context, query squirrel.Sqlizer) error {
	if s.HasSharding() {
		return fmt.Errorf("execQuery not supported for sharded storage")
	}

	queryText, args, err := query.ToSql()
	if err != nil {
		return errors.Wrap(err, "generate query error")
	}

	_, err = s.pg.Pool.Exec(ctx, queryText, args...)
	if err != nil {
		return errors.Wrap(err, "exec query error")
	}

	return nil
}

func (s *Storage) query(ctx context.Context, query squirrel.Sqlizer) (pgx.Rows, error) {
	if s.HasSharding() {
		return nil, fmt.Errorf("query not supported for sharded storage")
	}

	queryText, args, err := query.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "generate query error")
	}

	rows, err := s.pg.Pool.Query(ctx, queryText, args...)
	if err != nil {
		return nil, errors.Wrap(err, "rows query error")
	}

	return rows, nil
}

func (s *Storage) HealthCheck(ctx context.Context) error {
	if s.HasSharding() {
		for i, shard := range s.shards.AllShards() {
			if shard == nil {
				return fmt.Errorf("shard %d is nil", i)
			}
			if err := shard.Ping(ctx); err != nil {
				return fmt.Errorf("shard %d health check failed: %w", i, err)
			}
		}
		return nil
	}

	if s.pg == nil || s.pg.Pool == nil {
		return fmt.Errorf("database connection is nil")
	}

	if err := s.pg.Ping(ctx); err != nil {
		return errors.Wrap(err, "health check ping failed")
	}

	return nil
}

func (s *Storage) runMigrations(ctx context.Context) error {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("чтение миграций: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	if s.HasSharding() {
		for shardIndex, pool := range s.shards.AllShards() {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				content, err := migrationFiles.ReadFile("migrations/" + entry.Name())
				if err != nil {
					return fmt.Errorf("чтение миграции %s: %w", entry.Name(), err)
				}

				_, err = pool.Exec(ctx, string(content))
				if err != nil {
					return fmt.Errorf("применение миграции %s на шарде %d: %w", entry.Name(), shardIndex, err)
				}
			}
		}
		return nil
	}

	if s.pg != nil && s.pg.Pool != nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			content, err := migrationFiles.ReadFile("migrations/" + entry.Name())
			if err != nil {
				return fmt.Errorf("чтение миграции %s: %w", entry.Name(), err)
			}

			_, err = s.pg.Pool.Exec(ctx, string(content))
			if err != nil {
				return fmt.Errorf("применение миграции %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}
