package pgstorage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kedr891/cs-parser/internal/models"
)

func (s *Storage) CreateSkin(ctx context.Context, skin *models.Skin) error {
	qb := s.builder.
		Insert("skins").
		Columns(
			"id", "slug", "market_hash_name", "name", "weapon", "quality", "rarity",
			"current_price", "currency", "image_url", "volume_24h",
			"price_change_24h", "price_change_7d",
			"lowest_price", "highest_price",
			"last_updated", "created_at", "updated_at",
		).
		Values(
			skin.ID, skin.Slug, skin.MarketHashName, skin.Name, skin.Weapon, skin.Quality, skin.Rarity,
			skin.CurrentPrice, skin.Currency, skin.ImageURL, skin.Volume24h,
			skin.PriceChange24h, skin.PriceChange7d,
			skin.LowestPrice, skin.HighestPrice,
			skin.LastUpdated, skin.CreatedAt, skin.UpdatedAt,
		)

	queryText, args, err := qb.ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	if s.HasSharding() {
		shard := s.shards.GetShardByWeapon(skin.Weapon)
		_, err := shard.Exec(ctx, queryText, args...)
		if err != nil {
			if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
				existingSkin, findErr := s.GetSkinBySlug(ctx, skin.Slug)
				if findErr == nil && existingSkin != nil {
					skin.ID = existingSkin.ID
					return s.UpdateSkin(ctx, skin)
				}
				return s.UpdateSkin(ctx, skin)
			}
			return fmt.Errorf("create skin in shard: %w", err)
		}
		return nil
	}

	_, err = s.pg.Pool.Exec(ctx, queryText, args...)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			existingSkin, findErr := s.GetSkinBySlug(ctx, skin.Slug)
			if findErr == nil && existingSkin != nil {
				skin.ID = existingSkin.ID
				return s.UpdateSkin(ctx, skin)
			}
			return s.UpdateSkin(ctx, skin)
		}
		return fmt.Errorf("create skin: %w", err)
	}

	return nil
}

func (s *Storage) UpdateSkin(ctx context.Context, skin *models.Skin) error {
	whereClause := squirrel.Eq{"slug": skin.Slug}
	if skin.ID != uuid.Nil {
		whereClause = squirrel.Eq{"id": skin.ID}
	}

	qb := s.builder.
		Update("skins").
		Set("market_hash_name", skin.MarketHashName).
		Set("name", skin.Name).
		Set("weapon", skin.Weapon).
		Set("quality", skin.Quality).
		Set("rarity", skin.Rarity).
		Set("current_price", skin.CurrentPrice).
		Set("currency", skin.Currency).
		Set("image_url", skin.ImageURL).
		Set("volume_24h", skin.Volume24h).
		Set("price_change_24h", skin.PriceChange24h).
		Set("price_change_7d", skin.PriceChange7d).
		Set("lowest_price", skin.LowestPrice).
		Set("highest_price", skin.HighestPrice).
		Set("last_updated", skin.LastUpdated).
		Set("updated_at", skin.UpdatedAt).
		Where(whereClause)

	queryText, args, err := qb.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if s.HasSharding() {
		allShards := s.shards.AllShards()
		for _, shard := range allShards {
			if shard == nil {
				continue
			}
			result, err := shard.Exec(ctx, queryText, args...)
			if err != nil {
				continue
			}
			rowsAffected := result.RowsAffected()
			if rowsAffected > 0 {
				return nil
			}
		}
		return fmt.Errorf("skin not found for update")
	}

	result, err := s.pg.Pool.Exec(ctx, queryText, args...)
	if err != nil {
		return fmt.Errorf("update skin: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
