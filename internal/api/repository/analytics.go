package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kedr891/cs-parser/internal/entity"
	"github.com/kedr891/cs-parser/pkg/postgres"
)

// AnalyticsRepository - интерфейс репозитория аналитики
type AnalyticsRepository interface {
	// Trending
	GetTrendingSkins(ctx context.Context, period string, limit int) ([]entity.Skin, error)

	// Market Overview
	GetTotalSkinsCount(ctx context.Context) (int, error)
	GetAveragePrice(ctx context.Context) (float64, error)
	GetTotalVolume24h(ctx context.Context) (int, error)
	GetTopGainers(ctx context.Context, limit int) ([]entity.Skin, error)
	GetTopLosers(ctx context.Context, limit int) ([]entity.Skin, error)
	GetMostPopularSkins(ctx context.Context, limit int) ([]entity.Skin, error)
	GetRecentlyUpdatedSkins(ctx context.Context, limit int) ([]entity.Skin, error)

	// Price Statistics
	GetPriceStatsByPeriod(ctx context.Context, skinID uuid.UUID, period entity.PriceStatsPeriod) (*entity.SkinStatistics, error)
}

// analyticsRepository - реализация репозитория
type analyticsRepository struct {
	pg *postgres.Postgres
}

// NewAnalyticsRepository - создать репозиторий аналитики
func NewAnalyticsRepository(pg *postgres.Postgres) AnalyticsRepository {
	return &analyticsRepository{
		pg: pg,
	}
}

// GetTrendingSkins - получить трендовые скины
func (r *analyticsRepository) GetTrendingSkins(ctx context.Context, period string, limit int) ([]entity.Skin, error) {
	// Определить поле для сортировки по периоду
	sortField := "price_change_24h"
	if period == "7d" {
		sortField = "price_change_7d"
	}

	query := fmt.Sprintf(`
		SELECT 
			id, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		WHERE %s IS NOT NULL
		ORDER BY ABS(%s) DESC
		LIMIT $1
	`, sortField, sortField)

	rows, err := r.pg.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query trending skins: %w", err)
	}
	defer rows.Close()

	var skins []entity.Skin
	for rows.Next() {
		var skin entity.Skin
		err := rows.Scan(
			&skin.ID, &skin.MarketHashName, &skin.Name, &skin.Weapon, &skin.Quality, &skin.Rarity,
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

// GetTotalSkinsCount - получить общее количество скинов
func (r *analyticsRepository) GetTotalSkinsCount(ctx context.Context) (int, error) {
	var count int
	err := r.pg.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM skins`).Scan(&count)
	return count, err
}

// GetAveragePrice - получить среднюю цену всех скинов
func (r *analyticsRepository) GetAveragePrice(ctx context.Context) (float64, error) {
	var avg float64
	err := r.pg.Pool.QueryRow(ctx, `SELECT COALESCE(AVG(current_price), 0) FROM skins WHERE current_price > 0`).Scan(&avg)
	return avg, err
}

// GetTotalVolume24h - получить общий объём торгов за 24ч
func (r *analyticsRepository) GetTotalVolume24h(ctx context.Context) (int, error) {
	var volume int
	err := r.pg.Pool.QueryRow(ctx, `SELECT COALESCE(SUM(volume_24h), 0) FROM skins`).Scan(&volume)
	return volume, err
}

// GetTopGainers - получить топ растущих скинов
func (r *analyticsRepository) GetTopGainers(ctx context.Context, limit int) ([]entity.Skin, error) {
	query := `
		SELECT 
			id, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		WHERE price_change_24h > 0
		ORDER BY price_change_24h DESC
		LIMIT $1
	`

	rows, err := r.pg.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query top gainers: %w", err)
	}
	defer rows.Close()

	return r.scanSkins(rows)
}

// GetTopLosers - получить топ падающих скинов
func (r *analyticsRepository) GetTopLosers(ctx context.Context, limit int) ([]entity.Skin, error) {
	query := `
		SELECT 
			id, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		WHERE price_change_24h < 0
		ORDER BY price_change_24h ASC
		LIMIT $1
	`

	rows, err := r.pg.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query top losers: %w", err)
	}
	defer rows.Close()

	return r.scanSkins(rows)
}

// GetMostPopularSkins - получить самые популярные скины (по объёму торгов)
func (r *analyticsRepository) GetMostPopularSkins(ctx context.Context, limit int) ([]entity.Skin, error) {
	query := `
		SELECT 
			id, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		WHERE volume_24h > 0
		ORDER BY volume_24h DESC
		LIMIT $1
	`

	rows, err := r.pg.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query popular skins: %w", err)
	}
	defer rows.Close()

	return r.scanSkins(rows)
}

// GetRecentlyUpdatedSkins - получить недавно обновленные скины
func (r *analyticsRepository) GetRecentlyUpdatedSkins(ctx context.Context, limit int) ([]entity.Skin, error) {
	query := `
		SELECT 
			id, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		WHERE last_updated IS NOT NULL
		ORDER BY last_updated DESC
		LIMIT $1
	`

	rows, err := r.pg.Pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query recently updated: %w", err)
	}
	defer rows.Close()

	return r.scanSkins(rows)
}

// GetPriceStatsByPeriod - получить статистику цен за период (метод оставлен для совместимости, но не используется)
func (r *analyticsRepository) GetPriceStatsByPeriod(ctx context.Context, skinID uuid.UUID, period entity.PriceStatsPeriod) (*entity.SkinStatistics, error) {
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

	if err != nil {
		return nil, fmt.Errorf("query price stats: %w", err)
	}

	return &stats, nil
}

// Helper method для сканирования скинов
func (r *analyticsRepository) scanSkins(rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}) ([]entity.Skin, error) {
	var skins []entity.Skin
	for rows.Next() {
		var skin entity.Skin
		err := rows.Scan(
			&skin.ID, &skin.MarketHashName, &skin.Name, &skin.Weapon, &skin.Quality, &skin.Rarity,
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
