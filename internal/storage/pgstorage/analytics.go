package pgstorage

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/kedr891/cs-parser/internal/models"
)

func (s *Storage) GetTrendingSkins(ctx context.Context, period string, limit int) ([]models.Skin, error) {
	sortField := "price_change_24h"
	if period == "7d" {
		sortField = "price_change_7d"
	}

	qb := s.builder.
		Select(
			"id", "slug", "market_hash_name", "name", "weapon", "quality", "rarity",
			"current_price", "currency", "image_url", "volume_24h",
			"price_change_24h", "price_change_7d",
			"lowest_price", "highest_price",
			"last_updated", "created_at", "updated_at",
		).
		From("skins").
		Where(squirrel.NotEq{sortField: nil}).
		OrderBy(fmt.Sprintf("ABS(%s) DESC", sortField)).
		Limit(uint64(limit))

	queryText, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	if s.HasSharding() {
		var allSkins []models.Skin
		for _, shard := range s.shards.AllShards() {
			rows, err := shard.Query(ctx, queryText, args...)
			if err != nil {
				continue
			}
			skins, _ := s.scanSkins(rows)
			allSkins = append(allSkins, skins...)
		}
		if len(allSkins) > limit {
			return allSkins[:limit], nil
		}
		return allSkins, nil
	}

	rows, err := s.pg.Pool.Query(ctx, queryText, args...)
	if err != nil {
		return nil, fmt.Errorf("query trending skins: %w", err)
	}
	defer rows.Close()

	return s.scanSkins(rows)
}

func (s *Storage) GetTotalSkinsCount(ctx context.Context) (int, error) {
	qb := s.builder.Select("COUNT(*)").From("skins")
	queryText, args, err := qb.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build query: %w", err)
	}

	if s.HasSharding() {
		var total int
		for _, shard := range s.shards.AllShards() {
			var count int
			err := shard.QueryRow(ctx, queryText, args...).Scan(&count)
			if err != nil {
				continue
			}
			total += count
		}
		return total, nil
	}

	var count int
	err = s.pg.Pool.QueryRow(ctx, queryText, args...).Scan(&count)
	return count, err
}

func (s *Storage) GetAveragePrice(ctx context.Context) (float64, error) {
	if s.HasSharding() {
		qb := s.builder.
			Select("COALESCE(SUM(current_price), 0)", "COUNT(*)").
			From("skins").
			Where(squirrel.Gt{"current_price": 0})

		queryText, args, err := qb.ToSql()
		if err != nil {
			return 0, fmt.Errorf("build query: %w", err)
		}

		var totalSum float64
		var totalCount int
		for _, shard := range s.shards.AllShards() {
			var sum float64
			var count int
			err := shard.QueryRow(ctx, queryText, args...).Scan(&sum, &count)
			if err != nil {
				continue
			}
			totalSum += sum
			totalCount += count
		}
		if totalCount == 0 {
			return 0, nil
		}
		return totalSum / float64(totalCount), nil
	}

	qb := s.builder.
		Select("COALESCE(AVG(current_price), 0)").
		From("skins").
		Where(squirrel.Gt{"current_price": 0})

	queryText, args, err := qb.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build query: %w", err)
	}

	var avg float64
	err = s.pg.Pool.QueryRow(ctx, queryText, args...).Scan(&avg)
	return avg, err
}

func (s *Storage) GetTotalVolume24h(ctx context.Context) (int, error) {
	qb := s.builder.Select("COALESCE(SUM(volume_24h), 0)").From("skins")
	queryText, args, err := qb.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build query: %w", err)
	}

	if s.HasSharding() {
		var total int
		for _, shard := range s.shards.AllShards() {
			var volume int
			err := shard.QueryRow(ctx, queryText, args...).Scan(&volume)
			if err != nil {
				continue
			}
			total += volume
		}
		return total, nil
	}

	var volume int
	err = s.pg.Pool.QueryRow(ctx, queryText, args...).Scan(&volume)
	return volume, err
}

func (s *Storage) GetTopGainers(ctx context.Context, limit int) ([]models.Skin, error) {
	qb := s.builder.
		Select(
			"id", "slug", "market_hash_name", "name", "weapon", "quality", "rarity",
			"current_price", "currency", "image_url", "volume_24h",
			"price_change_24h", "price_change_7d",
			"lowest_price", "highest_price",
			"last_updated", "created_at", "updated_at",
		).
		From("skins").
		Where(squirrel.Gt{"price_change_24h": 0}).
		OrderBy("price_change_24h DESC").
		Limit(uint64(limit))

	queryText, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	if s.HasSharding() {
		var allSkins []models.Skin
		for _, shard := range s.shards.AllShards() {
			rows, err := shard.Query(ctx, queryText, args...)
			if err != nil {
				continue
			}
			skins, _ := s.scanSkins(rows)
			allSkins = append(allSkins, skins...)
		}
		if len(allSkins) > limit {
			return allSkins[:limit], nil
		}
		return allSkins, nil
	}

	rows, err := s.pg.Pool.Query(ctx, queryText, args...)
	if err != nil {
		return nil, fmt.Errorf("query top gainers: %w", err)
	}
	defer rows.Close()

	return s.scanSkins(rows)
}

func (s *Storage) GetTopLosers(ctx context.Context, limit int) ([]models.Skin, error) {
	qb := s.builder.
		Select(
			"id", "slug", "market_hash_name", "name", "weapon", "quality", "rarity",
			"current_price", "currency", "image_url", "volume_24h",
			"price_change_24h", "price_change_7d",
			"lowest_price", "highest_price",
			"last_updated", "created_at", "updated_at",
		).
		From("skins").
		Where(squirrel.Lt{"price_change_24h": 0}).
		OrderBy("price_change_24h ASC").
		Limit(uint64(limit))

	queryText, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	if s.HasSharding() {
		var allSkins []models.Skin
		for _, shard := range s.shards.AllShards() {
			rows, err := shard.Query(ctx, queryText, args...)
			if err != nil {
				continue
			}
			skins, _ := s.scanSkins(rows)
			allSkins = append(allSkins, skins...)
		}
		if len(allSkins) > limit {
			return allSkins[:limit], nil
		}
		return allSkins, nil
	}

	rows, err := s.pg.Pool.Query(ctx, queryText, args...)
	if err != nil {
		return nil, fmt.Errorf("query top losers: %w", err)
	}
	defer rows.Close()

	return s.scanSkins(rows)
}

func (s *Storage) GetMostPopularSkins(ctx context.Context, limit int) ([]models.Skin, error) {
	qb := s.builder.
		Select(
			"id", "slug", "market_hash_name", "name", "weapon", "quality", "rarity",
			"current_price", "currency", "image_url", "volume_24h",
			"price_change_24h", "price_change_7d",
			"lowest_price", "highest_price",
			"last_updated", "created_at", "updated_at",
		).
		From("skins").
		Where(squirrel.Gt{"volume_24h": 0}).
		OrderBy("volume_24h DESC").
		Limit(uint64(limit))

	queryText, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	if s.HasSharding() {
		var allSkins []models.Skin
		for _, shard := range s.shards.AllShards() {
			rows, err := shard.Query(ctx, queryText, args...)
			if err != nil {
				continue
			}
			skins, _ := s.scanSkins(rows)
			allSkins = append(allSkins, skins...)
		}
		if len(allSkins) > limit {
			return allSkins[:limit], nil
		}
		return allSkins, nil
	}

	rows, err := s.pg.Pool.Query(ctx, queryText, args...)
	if err != nil {
		return nil, fmt.Errorf("query popular skins: %w", err)
	}
	defer rows.Close()

	return s.scanSkins(rows)
}

func (s *Storage) GetRecentlyUpdatedSkins(ctx context.Context, limit int) ([]models.Skin, error) {
	qb := s.builder.
		Select(
			"id", "slug", "market_hash_name", "name", "weapon", "quality", "rarity",
			"current_price", "currency", "image_url", "volume_24h",
			"price_change_24h", "price_change_7d",
			"lowest_price", "highest_price",
			"last_updated", "created_at", "updated_at",
		).
		From("skins").
		Where(squirrel.NotEq{"last_updated": nil}).
		OrderBy("last_updated DESC").
		Limit(uint64(limit))

	queryText, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	if s.HasSharding() {
		var allSkins []models.Skin
		for _, shard := range s.shards.AllShards() {
			rows, err := shard.Query(ctx, queryText, args...)
			if err != nil {
				continue
			}
			skins, _ := s.scanSkins(rows)
			allSkins = append(allSkins, skins...)
		}
		if len(allSkins) > limit {
			return allSkins[:limit], nil
		}
		return allSkins, nil
	}

	rows, err := s.pg.Pool.Query(ctx, queryText, args...)
	if err != nil {
		return nil, fmt.Errorf("query recently updated: %w", err)
	}
	defer rows.Close()

	return s.scanSkins(rows)
}

func (s *Storage) GetPriceStatsByPeriod(ctx context.Context, skinID uuid.UUID, period models.PriceStatsPeriod) (*models.SkinStatistics, error) {
	qb := s.builder.
		Select(
			"COALESCE(AVG(CASE WHEN recorded_at >= NOW() - INTERVAL '7 days' THEN price END), 0) as avg_price_7d",
			"COALESCE(AVG(CASE WHEN recorded_at >= NOW() - INTERVAL '30 days' THEN price END), 0) as avg_price_30d",
			"COALESCE(SUM(CASE WHEN recorded_at >= NOW() - INTERVAL '7 days' THEN volume END), 0) as total_volume_7d",
			"COALESCE(STDDEV(price), 0) as price_volatility",
		).
		From("price_history").
		Where(squirrel.Eq{"skin_id": skinID})

	queryText, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	if s.HasSharding() {
		for _, shard := range s.shards.AllShards() {
			var stats models.SkinStatistics
			err := shard.QueryRow(ctx, queryText, args...).Scan(
				&stats.AvgPrice7d,
				&stats.AvgPrice30d,
				&stats.TotalVolume7d,
				&stats.PriceVolatility,
			)
			if err == nil {
				return &stats, nil
			}
		}
		return &models.SkinStatistics{}, nil
	}

	var stats models.SkinStatistics
	err = s.pg.Pool.QueryRow(ctx, queryText, args...).Scan(
		&stats.AvgPrice7d,
		&stats.AvgPrice30d,
		&stats.TotalVolume7d,
		&stats.PriceVolatility,
	)

	if err != nil {
		return nil, fmt.Errorf("query price stats: %w", err)
	}

	return &stats, nil
}
