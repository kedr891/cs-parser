package price

import (
	"context"
	"fmt"
	"time"

	"github.com/cs-parser/internal/entity"
	"github.com/cs-parser/pkg/logger"
	"github.com/cs-parser/pkg/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Repository - интерфейс репозитория для price consumer
type Repository interface {
	SavePriceHistory(ctx context.Context, history *entity.PriceHistory) error
	GetPriceHistory(ctx context.Context, skinID uuid.UUID, period entity.PriceStatsPeriod) ([]entity.PriceHistory, error)
	GetWatchlistsBySkinID(ctx context.Context, skinID uuid.UUID) ([]entity.Watchlist, error)
	CalculatePriceStats(ctx context.Context, skinID uuid.UUID, period entity.PriceStatsPeriod) (*entity.SkinStatistics, error)
	GetTrendingSkins(ctx context.Context, limit int, period entity.PriceStatsPeriod) ([]entity.TrendingSkin, error)
	UpdateSkinPriceChange(ctx context.Context, skinID uuid.UUID, change24h, change7d float64) error
}

// repository - реализация репозитория
type repository struct {
	pg  *postgres.Postgres
	log *logger.Logger
}

// NewRepository - создать репозиторий
func NewRepository(pg *postgres.Postgres, log *logger.Logger) Repository {
	return &repository{
		pg:  pg,
		log: log,
	}
}

// SavePriceHistory - сохранить историю цены
func (r *repository) SavePriceHistory(ctx context.Context, history *entity.PriceHistory) error {
	query := `
		INSERT INTO price_history (skin_id, price, currency, source, volume, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	err := r.pg.Pool.QueryRow(ctx, query,
		history.SkinID,
		history.Price,
		history.Currency,
		history.Source,
		history.Volume,
		history.RecordedAt,
	).Scan(&history.ID)

	if err != nil {
		return fmt.Errorf("insert price history: %w", err)
	}

	return nil
}

// GetPriceHistory - получить историю цен за период
func (r *repository) GetPriceHistory(ctx context.Context, skinID uuid.UUID, period entity.PriceStatsPeriod) ([]entity.PriceHistory, error) {
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
		err := rows.Scan(&h.ID, &h.SkinID, &h.Price, &h.Currency, &h.Source, &h.Volume, &h.RecordedAt)
		if err != nil {
			return nil, fmt.Errorf("scan price history: %w", err)
		}
		history = append(history, h)
	}

	return history, nil
}

// GetWatchlistsBySkinID - получить все watchlist для скина
func (r *repository) GetWatchlistsBySkinID(ctx context.Context, skinID uuid.UUID) ([]entity.Watchlist, error) {
	query := `
		SELECT 
			id, user_id, skin_id, target_price, 
			notify_on_drop, notify_on_price, is_active,
			added_at, updated_at
		FROM watchlist
		WHERE skin_id = $1 AND is_active = true
	`

	rows, err := r.pg.Pool.Query(ctx, query, skinID)
	if err != nil {
		return nil, fmt.Errorf("query watchlists: %w", err)
	}
	defer rows.Close()

	var watchlists []entity.Watchlist
	for rows.Next() {
		var wl entity.Watchlist
		err := rows.Scan(
			&wl.ID,
			&wl.UserID,
			&wl.SkinID,
			&wl.TargetPrice,
			&wl.NotifyOnDrop,
			&wl.NotifyOnPrice,
			&wl.IsActive,
			&wl.AddedAt,
			&wl.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan watchlist: %w", err)
		}
		watchlists = append(watchlists, wl)
	}

	return watchlists, nil
}

// CalculatePriceStats - рассчитать статистику по ценам
func (r *repository) CalculatePriceStats(ctx context.Context, skinID uuid.UUID, period entity.PriceStatsPeriod) (*entity.SkinStatistics, error) {
	since := time.Now().Add(-period.GetDuration())

	query := `
		SELECT 
			COALESCE(AVG(price), 0) as avg_price,
			COALESCE(SUM(volume), 0) as total_volume,
			COALESCE(STDDEV(price), 0) as price_volatility
		FROM price_history
		WHERE skin_id = $1 AND recorded_at >= $2
	`

	var stats entity.SkinStatistics
	err := r.pg.Pool.QueryRow(ctx, query, skinID, since).Scan(
		&stats.AvgPrice7d,
		&stats.TotalVolume7d,
		&stats.PriceVolatility,
	)

	if err == pgx.ErrNoRows {
		return &entity.SkinStatistics{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("calculate price stats: %w", err)
	}

	// Получить количество просмотров (если есть в БД или Redis)
	// stats.ViewCount = ... (можно добавить позже)

	return &stats, nil
}

// GetTrendingSkins - получить трендовые скины
func (r *repository) GetTrendingSkins(ctx context.Context, limit int, period entity.PriceStatsPeriod) ([]entity.TrendingSkin, error) {
	since := time.Now().Add(-period.GetDuration())

	query := `
		WITH price_changes AS (
			SELECT 
				s.id,
				s.market_hash_name,
				s.name,
				s.weapon,
				s.quality,
				s.rarity,
				s.current_price,
				s.currency,
				s.image_url,
				s.volume_24h,
				FIRST_VALUE(ph.price) OVER (PARTITION BY s.id ORDER BY ph.recorded_at ASC) as first_price,
				LAST_VALUE(ph.price) OVER (PARTITION BY s.id ORDER BY ph.recorded_at ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as last_price
			FROM skins s
			LEFT JOIN price_history ph ON s.id = ph.skin_id AND ph.recorded_at >= $1
			WHERE s.volume_24h > 0
		)
		SELECT DISTINCT
			id, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			CASE 
				WHEN first_price > 0 THEN ((last_price - first_price) / first_price) * 100
				ELSE 0
			END as price_change_rate
		FROM price_changes
		WHERE first_price > 0 AND last_price > 0
		ORDER BY price_change_rate DESC
		LIMIT $2
	`

	rows, err := r.pg.Pool.Query(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("query trending skins: %w", err)
	}
	defer rows.Close()

	var trending []entity.TrendingSkin
	rank := 1
	for rows.Next() {
		var ts entity.TrendingSkin
		err := rows.Scan(
			&ts.Skin.ID,
			&ts.Skin.MarketHashName,
			&ts.Skin.Name,
			&ts.Skin.Weapon,
			&ts.Skin.Quality,
			&ts.Skin.Rarity,
			&ts.Skin.CurrentPrice,
			&ts.Skin.Currency,
			&ts.Skin.ImageURL,
			&ts.Skin.Volume24h,
			&ts.PriceChangeRate,
		)
		if err != nil {
			return nil, fmt.Errorf("scan trending skin: %w", err)
		}
		ts.Rank = rank
		rank++
		trending = append(trending, ts)
	}

	return trending, nil
}

// UpdateSkinPriceChange - обновить процент изменения цены для скина
func (r *repository) UpdateSkinPriceChange(ctx context.Context, skinID uuid.UUID, change24h, change7d float64) error {
	query := `
		UPDATE skins
		SET 
			price_change_24h = $1,
			price_change_7d = $2,
			updated_at = NOW()
		WHERE id = $3
	`

	_, err := r.pg.Pool.Exec(ctx, query, change24h, change7d, skinID)
	if err != nil {
		return fmt.Errorf("update skin price change: %w", err)
	}

	return nil
}

// GetPriceChartData - получить данные для графика цен
func (r *repository) GetPriceChartData(ctx context.Context, skinID uuid.UUID, period entity.PriceStatsPeriod) (*entity.PriceChartResponse, error) {
	history, err := r.GetPriceHistory(ctx, skinID, period)
	if err != nil {
		return nil, fmt.Errorf("get price history: %w", err)
	}

	if len(history) == 0 {
		return &entity.PriceChartResponse{
			SkinID:     skinID,
			Period:     string(period),
			DataPoints: []entity.PriceChartData{},
		}, nil
	}

	// Конвертация в формат графика
	dataPoints := make([]entity.PriceChartData, len(history))
	var minPrice, maxPrice, sumPrice float64
	var totalVolume int

	minPrice = history[0].Price
	maxPrice = history[0].Price

	for i, h := range history {
		dataPoints[i] = entity.PriceChartData{
			Timestamp: h.RecordedAt,
			Price:     h.Price,
			Volume:    h.Volume,
		}

		if h.Price < minPrice {
			minPrice = h.Price
		}
		if h.Price > maxPrice {
			maxPrice = h.Price
		}
		sumPrice += h.Price
		totalVolume += h.Volume
	}

	avgPrice := sumPrice / float64(len(history))

	return &entity.PriceChartResponse{
		SkinID:      skinID,
		Period:      string(period),
		DataPoints:  dataPoints,
		MinPrice:    minPrice,
		MaxPrice:    maxPrice,
		AvgPrice:    avgPrice,
		TotalVolume: totalVolume,
	}, nil
}
