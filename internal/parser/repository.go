package parser

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/kedr891/cs-parser/internal/entity"
	"github.com/kedr891/cs-parser/pkg/logger"
	"github.com/kedr891/cs-parser/pkg/postgres"
)

// Repository - интерфейс репозитория для парсера
type Repository interface {
	GetAllSkins(ctx context.Context) ([]entity.Skin, error)
	GetSkinByID(ctx context.Context, id uuid.UUID) (*entity.Skin, error)
	GetSkinByMarketHashName(ctx context.Context, marketHashName string) (*entity.Skin, error)
	SkinExists(ctx context.Context, marketHashName string) (bool, error)
	CreateSkin(ctx context.Context, skin *entity.Skin) error
	UpdateSkinPrice(ctx context.Context, skinID uuid.UUID, price float64, volume int) error
	SavePriceHistory(ctx context.Context, history *entity.PriceHistory) error
	GetSkinsCount(ctx context.Context) (int, error)
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

// GetAllSkins - получить все скины
func (r *repository) GetAllSkins(ctx context.Context) ([]entity.Skin, error) {
	query := `
		SELECT 
			id, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		ORDER BY updated_at ASC
	`

	rows, err := r.pg.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query skins: %w", err)
	}
	defer rows.Close()

	var skins []entity.Skin
	for rows.Next() {
		var skin entity.Skin
		err := rows.Scan(
			&skin.ID,
			&skin.MarketHashName,
			&skin.Name,
			&skin.Weapon,
			&skin.Quality,
			&skin.Rarity,
			&skin.CurrentPrice,
			&skin.Currency,
			&skin.ImageURL,
			&skin.Volume24h,
			&skin.PriceChange24h,
			&skin.PriceChange7d,
			&skin.LowestPrice,
			&skin.HighestPrice,
			&skin.LastUpdated,
			&skin.CreatedAt,
			&skin.UpdatedAt,
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

// GetSkinByID - получить скин по ID
func (r *repository) GetSkinByID(ctx context.Context, id uuid.UUID) (*entity.Skin, error) {
	query := `
		SELECT 
			id, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		WHERE id = $1
	`

	var skin entity.Skin
	err := r.pg.Pool.QueryRow(ctx, query, id).Scan(
		&skin.ID,
		&skin.MarketHashName,
		&skin.Name,
		&skin.Weapon,
		&skin.Quality,
		&skin.Rarity,
		&skin.CurrentPrice,
		&skin.Currency,
		&skin.ImageURL,
		&skin.Volume24h,
		&skin.PriceChange24h,
		&skin.PriceChange7d,
		&skin.LowestPrice,
		&skin.HighestPrice,
		&skin.LastUpdated,
		&skin.CreatedAt,
		&skin.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("skin not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("query skin: %w", err)
	}

	return &skin, nil
}

// GetSkinByMarketHashName - получить скин по market_hash_name
func (r *repository) GetSkinByMarketHashName(ctx context.Context, marketHashName string) (*entity.Skin, error) {
	query := `
		SELECT 
			id, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		WHERE market_hash_name = $1
	`

	var skin entity.Skin
	err := r.pg.Pool.QueryRow(ctx, query, marketHashName).Scan(
		&skin.ID,
		&skin.MarketHashName,
		&skin.Name,
		&skin.Weapon,
		&skin.Quality,
		&skin.Rarity,
		&skin.CurrentPrice,
		&skin.Currency,
		&skin.ImageURL,
		&skin.Volume24h,
		&skin.PriceChange24h,
		&skin.PriceChange7d,
		&skin.LowestPrice,
		&skin.HighestPrice,
		&skin.LastUpdated,
		&skin.CreatedAt,
		&skin.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("skin not found: %s", marketHashName)
	}
	if err != nil {
		return nil, fmt.Errorf("query skin: %w", err)
	}

	return &skin, nil
}

// SkinExists - проверить существование скина
func (r *repository) SkinExists(ctx context.Context, marketHashName string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM skins WHERE market_hash_name = $1)`

	var exists bool
	err := r.pg.Pool.QueryRow(ctx, query, marketHashName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check skin existence: %w", err)
	}

	return exists, nil
}

// CreateSkin - создать новый скин
func (r *repository) CreateSkin(ctx context.Context, skin *entity.Skin) error {
	query := `
		INSERT INTO skins (
			id, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
	`

	_, err := r.pg.Pool.Exec(ctx, query,
		skin.ID,
		skin.MarketHashName,
		skin.Name,
		skin.Weapon,
		skin.Quality,
		skin.Rarity,
		skin.CurrentPrice,
		skin.Currency,
		skin.ImageURL,
		skin.Volume24h,
		skin.PriceChange24h,
		skin.PriceChange7d,
		skin.LowestPrice,
		skin.HighestPrice,
		skin.LastUpdated,
		skin.CreatedAt,
		skin.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("insert skin: %w", err)
	}

	return nil
}

// UpdateSkinPrice - обновить цену скина
func (r *repository) UpdateSkinPrice(ctx context.Context, skinID uuid.UUID, price float64, volume int) error {
	// Используем транзакцию для атомарного обновления
	return r.pg.Transaction(ctx, func(tx pgx.Tx) error {
		// Получить старую цену для расчёта изменений
		var oldPrice float64
		err := tx.QueryRow(ctx, `SELECT current_price FROM skins WHERE id = $1`, skinID).Scan(&oldPrice)
		if err != nil {
			return fmt.Errorf("get old price: %w", err)
		}

		// Рассчитать изменение цены
		priceChange := 0.0
		if oldPrice > 0 {
			priceChange = ((price - oldPrice) / oldPrice) * 100
		}

		// Обновить скин
		query := `
			UPDATE skins
			SET 
				current_price = $1,
				volume_24h = $2,
				price_change_24h = $3,
				lowest_price = CASE 
					WHEN lowest_price = 0 OR $1 < lowest_price THEN $1 
					ELSE lowest_price 
				END,
				highest_price = CASE 
					WHEN $1 > highest_price THEN $1 
					ELSE highest_price 
				END,
				last_updated = NOW(),
				updated_at = NOW()
			WHERE id = $4
		`

		_, err = tx.Exec(ctx, query, price, volume, priceChange, skinID)
		if err != nil {
			return fmt.Errorf("update skin: %w", err)
		}

		// Сохранить историю цены
		historyQuery := `
			INSERT INTO price_history (skin_id, price, source, volume, recorded_at)
			VALUES ($1, $2, $3, $4, NOW())
		`

		_, err = tx.Exec(ctx, historyQuery, skinID, price, "steam_market", volume)
		if err != nil {
			return fmt.Errorf("insert price history: %w", err)
		}

		return nil
	})
}

// SavePriceHistory - сохранить запись истории цены
func (r *repository) SavePriceHistory(ctx context.Context, history *entity.PriceHistory) error {
	query := `
		INSERT INTO price_history (skin_id, price, currency, source, volume, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.pg.Pool.Exec(ctx, query,
		history.SkinID,
		history.Price,
		history.Currency,
		history.Source,
		history.Volume,
		history.RecordedAt,
	)

	if err != nil {
		return fmt.Errorf("insert price history: %w", err)
	}

	return nil
}

// GetSkinsCount - получить количество скинов
func (r *repository) GetSkinsCount(ctx context.Context) (int, error) {
	var count int
	err := r.pg.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM skins`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count skins: %w", err)
	}
	return count, nil
}
