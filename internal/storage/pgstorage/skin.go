package pgstorage

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/kedr891/cs-parser/internal/models"
)

func (s *Storage) GetSkins(ctx context.Context, filter *models.SkinFilter) ([]models.Skin, int, error) {
	if s.HasSharding() {
		return s.getSkinsSharded(ctx, filter)
	}
	return s.getSkinsSingle(ctx, filter)
}

func (s *Storage) getSkinsSingle(ctx context.Context, filter *models.SkinFilter) ([]models.Skin, int, error) {
	qb := s.builder.
		Select(
			"id", "slug", "market_hash_name", "name", "weapon", "quality", "rarity",
			"current_price", "currency", "image_url", "volume_24h",
			"price_change_24h", "price_change_7d",
			"lowest_price", "highest_price",
			"last_updated", "created_at", "updated_at",
		).
		From("skins")

	countQb := s.builder.Select("COUNT(*)").From("skins")

	// Apply filters to both queries
	if filter.Weapon != "" {
		qb = qb.Where(squirrel.Eq{"weapon": filter.Weapon})
		countQb = countQb.Where(squirrel.Eq{"weapon": filter.Weapon})
	}
	if filter.Quality != "" {
		qb = qb.Where(squirrel.Eq{"quality": filter.Quality})
		countQb = countQb.Where(squirrel.Eq{"quality": filter.Quality})
	}
	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		qb = qb.Where("name ILIKE ? OR market_hash_name ILIKE ?", searchPattern, searchPattern)
		countQb = countQb.Where("name ILIKE ? OR market_hash_name ILIKE ?", searchPattern, searchPattern)
	}
	if filter.MinPrice > 0 {
		qb = qb.Where(squirrel.GtOrEq{"current_price": filter.MinPrice})
		countQb = countQb.Where(squirrel.GtOrEq{"current_price": filter.MinPrice})
	}
	if filter.MaxPrice > 0 {
		qb = qb.Where(squirrel.LtOrEq{"current_price": filter.MaxPrice})
		countQb = countQb.Where(squirrel.LtOrEq{"current_price": filter.MaxPrice})
	}

	countQuery, countArgs, err := countQb.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count query: %w", err)
	}

	var total int
	if err := s.pg.Pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count skins: %w", err)
	}

	if total == 0 {
		return []models.Skin{}, 0, nil
	}

	// Apply sorting
	sortBy := "updated_at"
	switch filter.SortBy {
	case "price":
		sortBy = "current_price"
	case "volume":
		sortBy = "volume_24h"
	case "name":
		sortBy = "name"
	case "updated":
		sortBy = "updated_at"
	case "created":
		sortBy = "created_at"
	case "weapon":
		sortBy = "weapon"
	}

	if strings.ToUpper(filter.SortOrder) == "ASC" {
		qb = qb.OrderBy(sortBy + " ASC")
	} else {
		qb = qb.OrderBy(sortBy + " DESC")
	}

	qb = qb.Limit(uint64(filter.Limit)).Offset(uint64(filter.Offset))

	// Execute query
	rows, err := s.query(ctx, qb)
	if err != nil {
		return nil, 0, fmt.Errorf("query skins: %w", err)
	}
	defer rows.Close()

	var skins []models.Skin
	for rows.Next() {
		var skin models.Skin
		err := rows.Scan(
			&skin.ID, &skin.Slug, &skin.MarketHashName, &skin.Name, &skin.Weapon, &skin.Quality, &skin.Rarity,
			&skin.CurrentPrice, &skin.Currency, &skin.ImageURL, &skin.Volume24h,
			&skin.PriceChange24h, &skin.PriceChange7d,
			&skin.LowestPrice, &skin.HighestPrice,
			&skin.LastUpdated, &skin.CreatedAt, &skin.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan skin: %w", err)
		}
		skins = append(skins, skin)
	}

	return skins, total, nil
}

func (s *Storage) getSkinsSharded(ctx context.Context, filter *models.SkinFilter) ([]models.Skin, int, error) {
	var allSkins []models.Skin
	var totalCount int

	for _, shard := range s.shards.AllShards() {
		// Build query using squirrel
		qb := s.builder.
			Select(
				"id", "slug", "market_hash_name", "name", "weapon", "quality", "rarity",
				"current_price", "currency", "image_url", "volume_24h",
				"price_change_24h", "price_change_7d",
				"lowest_price", "highest_price",
				"last_updated", "created_at", "updated_at",
			).
			From("skins")

		countQb := s.builder.Select("COUNT(*)").From("skins")

		// Apply filters
		if filter.Weapon != "" {
			qb = qb.Where(squirrel.Eq{"weapon": filter.Weapon})
			countQb = countQb.Where(squirrel.Eq{"weapon": filter.Weapon})
		}
		if filter.Quality != "" {
			qb = qb.Where(squirrel.Eq{"quality": filter.Quality})
			countQb = countQb.Where(squirrel.Eq{"quality": filter.Quality})
		}
		if filter.Search != "" {
			searchPattern := "%" + filter.Search + "%"
			qb = qb.Where("name ILIKE ? OR market_hash_name ILIKE ?", searchPattern, searchPattern)
			countQb = countQb.Where("name ILIKE ? OR market_hash_name ILIKE ?", searchPattern, searchPattern)
		}
		if filter.MinPrice > 0 {
			qb = qb.Where(squirrel.GtOrEq{"current_price": filter.MinPrice})
			countQb = countQb.Where(squirrel.GtOrEq{"current_price": filter.MinPrice})
		}
		if filter.MaxPrice > 0 {
			qb = qb.Where(squirrel.LtOrEq{"current_price": filter.MaxPrice})
			countQb = countQb.Where(squirrel.LtOrEq{"current_price": filter.MaxPrice})
		}

		// Count on this shard
		countQuery, countArgs, err := countQb.ToSql()
		if err != nil {
			continue
		}

		var shardCount int
		if err := shard.QueryRow(ctx, countQuery, countArgs...).Scan(&shardCount); err != nil {
			continue
		}
		totalCount += shardCount

		// Apply sorting
		sortBy := "updated_at"
		switch filter.SortBy {
		case "price":
			sortBy = "current_price"
		case "volume":
			sortBy = "volume_24h"
		case "name":
			sortBy = "name"
		case "updated":
			sortBy = "updated_at"
		case "created":
			sortBy = "created_at"
		case "weapon":
			sortBy = "weapon"
		}

		if strings.ToUpper(filter.SortOrder) == "ASC" {
			qb = qb.OrderBy(sortBy + " ASC")
		} else {
			qb = qb.OrderBy(sortBy + " DESC")
		}

		qb = qb.Limit(uint64(filter.Limit)).Offset(uint64(filter.Offset))

		// Execute query
		queryText, args, err := qb.ToSql()
		if err != nil {
			continue
		}

		rows, err := shard.Query(ctx, queryText, args...)
		if err != nil {
			continue
		}

		for rows.Next() {
			var skin models.Skin
			err := rows.Scan(
				&skin.ID, &skin.Slug, &skin.MarketHashName, &skin.Name, &skin.Weapon, &skin.Quality, &skin.Rarity,
				&skin.CurrentPrice, &skin.Currency, &skin.ImageURL, &skin.Volume24h,
				&skin.PriceChange24h, &skin.PriceChange7d,
				&skin.LowestPrice, &skin.HighestPrice,
				&skin.LastUpdated, &skin.CreatedAt, &skin.UpdatedAt,
			)
			if err != nil {
				rows.Close()
				continue
			}
			allSkins = append(allSkins, skin)
		}
		rows.Close()
	}

	return allSkins, totalCount, nil
}

func (s *Storage) GetSkinBySlug(ctx context.Context, slug string) (*models.Skin, error) {
	if s.HasSharding() {
		return s.getSkinBySlugSharded(ctx, slug)
	}
	return s.getSkinBySlugSingle(ctx, slug)
}

func (s *Storage) getSkinBySlugSingle(ctx context.Context, slug string) (*models.Skin, error) {
	qb := s.builder.
		Select(
			"id", "slug", "market_hash_name", "name", "weapon", "quality", "rarity",
			"current_price", "currency", "image_url", "volume_24h",
			"price_change_24h", "price_change_7d",
			"lowest_price", "highest_price",
			"last_updated", "created_at", "updated_at",
		).
		From("skins").
		Where(squirrel.Eq{"slug": slug})

	queryText, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var skin models.Skin
	err = s.pg.Pool.QueryRow(ctx, queryText, args...).Scan(
		&skin.ID, &skin.Slug, &skin.MarketHashName, &skin.Name, &skin.Weapon, &skin.Quality, &skin.Rarity,
		&skin.CurrentPrice, &skin.Currency, &skin.ImageURL, &skin.Volume24h,
		&skin.PriceChange24h, &skin.PriceChange7d,
		&skin.LowestPrice, &skin.HighestPrice,
		&skin.LastUpdated, &skin.CreatedAt, &skin.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("skin not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query skin: %w", err)
	}

	return &skin, nil
}

func (s *Storage) getSkinBySlugSharded(ctx context.Context, slug string) (*models.Skin, error) {
	qb := s.builder.
		Select(
			"id", "slug", "market_hash_name", "name", "weapon", "quality", "rarity",
			"current_price", "currency", "image_url", "volume_24h",
			"price_change_24h", "price_change_7d",
			"lowest_price", "highest_price",
			"last_updated", "created_at", "updated_at",
		).
		From("skins").
		Where(squirrel.Eq{"slug": slug})

	queryText, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	for _, shard := range s.shards.AllShards() {
		var skin models.Skin
		err := shard.QueryRow(ctx, queryText, args...).Scan(
			&skin.ID, &skin.Slug, &skin.MarketHashName, &skin.Name, &skin.Weapon, &skin.Quality, &skin.Rarity,
			&skin.CurrentPrice, &skin.Currency, &skin.ImageURL, &skin.Volume24h,
			&skin.PriceChange24h, &skin.PriceChange7d,
			&skin.LowestPrice, &skin.HighestPrice,
			&skin.LastUpdated, &skin.CreatedAt, &skin.UpdatedAt,
		)

		if err == nil {
			return &skin, nil
		}
		if err != pgx.ErrNoRows {
			return nil, fmt.Errorf("query shard: %w", err)
		}
	}

	return nil, fmt.Errorf("skin not found")
}

func (s *Storage) GetPriceHistory(ctx context.Context, skinID uuid.UUID, period models.PriceStatsPeriod) ([]models.PriceHistory, error) {
	qb := s.builder.
		Select("id", "skin_id", "price", "currency", "source", "volume", "recorded_at").
		From("price_history").
		Where(squirrel.Eq{"skin_id": skinID}).
		Where("recorded_at >= NOW() - ?::interval", period.GetDuration()).
		OrderBy("recorded_at ASC")

	queryText, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	if s.HasSharding() {
		var allHistory []models.PriceHistory
		for _, shard := range s.shards.AllShards() {
			rows, err := shard.Query(ctx, queryText, args...)
			if err != nil {
				continue
			}
			for rows.Next() {
				var h models.PriceHistory
				if err := rows.Scan(&h.ID, &h.SkinID, &h.Price, &h.Currency, &h.Source, &h.Volume, &h.RecordedAt); err != nil {
					rows.Close()
					continue
				}
				allHistory = append(allHistory, h)
			}
			rows.Close()
		}
		return allHistory, nil
	}

	rows, err := s.pg.Pool.Query(ctx, queryText, args...)
	if err != nil {
		return nil, fmt.Errorf("query price history: %w", err)
	}
	defer rows.Close()

	var history []models.PriceHistory
	for rows.Next() {
		var h models.PriceHistory
		if err := rows.Scan(&h.ID, &h.SkinID, &h.Price, &h.Currency, &h.Source, &h.Volume, &h.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan price history: %w", err)
		}
		history = append(history, h)
	}

	return history, nil
}

func (s *Storage) GetSkinStatistics(ctx context.Context, skinID uuid.UUID) (*models.SkinStatistics, error) {
	query := `
		SELECT 
			COALESCE(AVG(CASE WHEN recorded_at >= NOW() - INTERVAL '7 days' THEN price END), 0) as avg_price_7d,
			COALESCE(AVG(CASE WHEN recorded_at >= NOW() - INTERVAL '30 days' THEN price END), 0) as avg_price_30d,
			COALESCE(SUM(CASE WHEN recorded_at >= NOW() - INTERVAL '7 days' THEN volume END), 0) as total_volume_7d,
			COALESCE(STDDEV(price), 0) as price_volatility
		FROM price_history
		WHERE skin_id = $1
	`

	if s.HasSharding() {
		// Query all shards
		for _, shard := range s.shards.AllShards() {
			var stats models.SkinStatistics
			err := shard.QueryRow(ctx, query, skinID).Scan(
				&stats.AvgPrice7d,
				&stats.AvgPrice30d,
				&stats.TotalVolume7d,
				&stats.PriceVolatility,
			)
			if err == nil {
				return &stats, nil
			}
			if err != pgx.ErrNoRows {
				return nil, fmt.Errorf("query statistics: %w", err)
			}
		}
		return &models.SkinStatistics{}, nil
	}

	var stats models.SkinStatistics
	err := s.pg.Pool.QueryRow(ctx, query, skinID).Scan(
		&stats.AvgPrice7d,
		&stats.AvgPrice30d,
		&stats.TotalVolume7d,
		&stats.PriceVolatility,
	)

	if err == pgx.ErrNoRows {
		return &models.SkinStatistics{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query statistics: %w", err)
	}

	return &stats, nil
}

func (s *Storage) SearchSkins(ctx context.Context, query string, limit int) ([]models.Skin, error) {
	searchPattern := "%" + query + "%"
	qb := s.builder.
		Select(
			"id", "slug", "market_hash_name", "name", "weapon", "quality", "rarity",
			"current_price", "currency", "image_url", "volume_24h",
			"price_change_24h", "price_change_7d",
			"lowest_price", "highest_price",
			"last_updated", "created_at", "updated_at",
		).
		From("skins").
		Where("name ILIKE ? OR market_hash_name ILIKE ?", searchPattern, searchPattern).
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
			allSkins = allSkins[:limit]
		}
		return allSkins, nil
	}

	rows, err := s.pg.Pool.Query(ctx, queryText, args...)
	if err != nil {
		return nil, fmt.Errorf("query search: %w", err)
	}
	defer rows.Close()

	return s.scanSkins(rows)
}

func (s *Storage) GetPopularSkins(ctx context.Context, limit int) ([]models.Skin, error) {
	qb := s.builder.
		Select(
			"id", "slug", "market_hash_name", "name", "weapon", "quality", "rarity",
			"current_price", "currency", "image_url", "volume_24h",
			"price_change_24h", "price_change_7d",
			"lowest_price", "highest_price",
			"last_updated", "created_at", "updated_at",
		).
		From("skins").
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
			allSkins = allSkins[:limit]
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

func (s *Storage) scanSkins(rows pgx.Rows) ([]models.Skin, error) {
	defer rows.Close()
	var skins []models.Skin
	for rows.Next() {
		var skin models.Skin
		err := rows.Scan(
			&skin.ID, &skin.Slug, &skin.MarketHashName, &skin.Name, &skin.Weapon, &skin.Quality, &skin.Rarity,
			&skin.CurrentPrice, &skin.Currency, &skin.ImageURL, &skin.Volume24h,
			&skin.PriceChange24h, &skin.PriceChange7d,
			&skin.LowestPrice, &skin.HighestPrice,
			&skin.LastUpdated, &skin.CreatedAt, &skin.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan skin: %w", err)
		}
		skins = append(skins, skin)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return skins, nil
}
