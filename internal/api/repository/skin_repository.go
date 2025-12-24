package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/kedr891/cs-parser/internal/entity"
	"github.com/kedr891/cs-parser/pkg/postgres"
)

type SkinRepository interface {
	GetSkins(ctx context.Context, filter *entity.SkinFilter) ([]entity.Skin, int, error)
	GetSkinBySlug(ctx context.Context, slug string) (*entity.Skin, error)
	GetPriceHistory(ctx context.Context, skinID uuid.UUID, period entity.PriceStatsPeriod) ([]entity.PriceHistory, error)
	GetSkinStatistics(ctx context.Context, skinID uuid.UUID) (*entity.SkinStatistics, error)
	SearchSkins(ctx context.Context, query string, limit int) ([]entity.Skin, error)
	GetPopularSkins(ctx context.Context, limit int) ([]entity.Skin, error)
}

type skinRepository struct {
	pg *postgres.Postgres
}

func NewSkinRepository(pg *postgres.Postgres) SkinRepository {
	return &skinRepository{
		pg: pg,
	}
}

func (r *skinRepository) GetSkins(ctx context.Context, filter *entity.SkinFilter) ([]entity.Skin, int, error) {
	baseQuery := `
		SELECT 
			id, slug, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
	`

	countQuery := `SELECT COUNT(*) FROM skins`

	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if filter.Weapon != "" {
		conditions = append(conditions, fmt.Sprintf("weapon = $%d", argIndex))
		args = append(args, filter.Weapon)
		argIndex++
	}

	if filter.Quality != "" {
		conditions = append(conditions, fmt.Sprintf("quality = $%d", argIndex))
		args = append(args, filter.Quality)
		argIndex++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR market_hash_name ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	if filter.MinPrice > 0 {
		conditions = append(conditions, fmt.Sprintf("current_price >= $%d", argIndex))
		args = append(args, filter.MinPrice)
		argIndex++
	}

	if filter.MaxPrice > 0 {
		conditions = append(conditions, fmt.Sprintf("current_price <= $%d", argIndex))
		args = append(args, filter.MaxPrice)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQueryFull := countQuery + whereClause
	if err := r.pg.Pool.QueryRow(ctx, countQueryFull, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count skins: %w", err)
	}

	if total == 0 {
		return []entity.Skin{}, 0, nil
	}

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

	sortOrder := "DESC"
	if strings.ToUpper(filter.SortOrder) == "ASC" {
		sortOrder = "ASC"
	}

	fullQuery := baseQuery + whereClause +
		fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d", sortBy, sortOrder, argIndex, argIndex+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pg.Pool.Query(ctx, fullQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query skins: %w", err)
	}
	defer rows.Close()

	var skins []entity.Skin
	for rows.Next() {
		var skin entity.Skin
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

func (r *skinRepository) GetSkinBySlug(ctx context.Context, slug string) (*entity.Skin, error) {
	query := `
		SELECT 
			id, slug, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		WHERE slug = $1
	`

	var skin entity.Skin
	err := r.pg.Pool.QueryRow(ctx, query, slug).Scan(
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

func (r *skinRepository) GetPriceHistory(ctx context.Context, skinID uuid.UUID, period entity.PriceStatsPeriod) ([]entity.PriceHistory, error) {
	since := time.Now().Add(-period.GetDuration())

	query := `
		SELECT id, skin_id, price, currency, source, volume, recorded_at
		FROM price_history
		WHERE skin_id = $1 AND recorded_at >= $2
		ORDER BY recorded_at ASC
	`

	rows, err := r.pg.Pool.Query(ctx, query, skinID, since)
	if err != nil {
		return nil, fmt.Errorf("query price history: %w", err)
	}
	defer rows.Close()

	var history []entity.PriceHistory
	for rows.Next() {
		var h entity.PriceHistory
		if err := rows.Scan(&h.ID, &h.SkinID, &h.Price, &h.Currency, &h.Source, &h.Volume, &h.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan price history: %w", err)
		}
		history = append(history, h)
	}

	return history, nil
}

func (r *skinRepository) GetSkinStatistics(ctx context.Context, skinID uuid.UUID) (*entity.SkinStatistics, error) {
	query := `
		SELECT 
			COALESCE(AVG(CASE WHEN recorded_at >= NOW() - INTERVAL '7 days' THEN price END), 0) as avg_price_7d,
			COALESCE(AVG(CASE WHEN recorded_at >= NOW() - INTERVAL '30 days' THEN price END), 0) as avg_price_30d,
			COALESCE(SUM(CASE WHEN recorded_at >= NOW() - INTERVAL '7 days' THEN volume END), 0) as total_volume_7d,
			COALESCE(STDDEV(price), 0) as price_volatility
		FROM price_history
		WHERE skin_id = $1
	`

	var stats entity.SkinStatistics
	err := r.pg.Pool.QueryRow(ctx, query, skinID).Scan(
		&stats.AvgPrice7d,
		&stats.AvgPrice30d,
		&stats.TotalVolume7d,
		&stats.PriceVolatility,
	)

	if err == pgx.ErrNoRows {
		return &entity.SkinStatistics{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query statistics: %w", err)
	}

	return &stats, nil
}

func (r *skinRepository) SearchSkins(ctx context.Context, query string, limit int) ([]entity.Skin, error) {
	sql := `
		SELECT 
			id, slug, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		WHERE name ILIKE $1 OR market_hash_name ILIKE $1
		ORDER BY volume_24h DESC
		LIMIT $2
	`

	rows, err := r.pg.Pool.Query(ctx, sql, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("query search: %w", err)
	}
	defer rows.Close()

	var skins []entity.Skin
	for rows.Next() {
		var skin entity.Skin
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

	return skins, nil
}

func (r *skinRepository) GetPopularSkins(ctx context.Context, limit int) ([]entity.Skin, error) {
	query := `
		SELECT 
			id, slug, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		ORDER BY volume_24h DESC
		LIMIT $1
	`

	rows, err := r.pg.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query popular skins: %w", err)
	}
	defer rows.Close()

	var skins []entity.Skin
	for rows.Next() {
		var skin entity.Skin
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

	return skins, nil
}
