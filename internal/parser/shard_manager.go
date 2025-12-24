package parser

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kedr891/cs-parser/internal/entity"
	"github.com/kedr891/cs-parser/pkg/logger"
	"github.com/kedr891/cs-parser/pkg/postgres"
)

type shardedRepository struct {
	shardManager *postgres.ShardManager
	log          *logger.Logger
}

func NewShardedRepository(shardManager *postgres.ShardManager, log *logger.Logger) Repository {
	return &shardedRepository{
		shardManager: shardManager,
		log:          log,
	}
}

func (r *shardedRepository) GetAllSkins(ctx context.Context) ([]entity.Skin, error) {
	query := `
		SELECT 
			id, slug, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		ORDER BY updated_at ASC
	`

	var (
		allSkins []entity.Skin
		mu       sync.Mutex
		wg       sync.WaitGroup
		errChan  = make(chan error, r.shardManager.ShardsCount())
	)

	for _, shard := range r.shardManager.GetAllShards() {
		wg.Add(1)
		go func(pool *pgxpool.Pool) {
			defer wg.Done()

			rows, err := pool.Query(ctx, query)
			if err != nil {
				errChan <- fmt.Errorf("query shard: %w", err)
				return
			}
			defer rows.Close()

			var shardSkins []entity.Skin
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
					errChan <- fmt.Errorf("scan skin: %w", err)
					return
				}
				shardSkins = append(shardSkins, skin)
			}

			mu.Lock()
			allSkins = append(allSkins, shardSkins...)
			mu.Unlock()
		}(shard)
	}

	wg.Wait()
	close(errChan)

	if err := <-errChan; err != nil {
		return nil, err
	}

	return allSkins, nil
}

func (r *shardedRepository) GetSkinBySlug(ctx context.Context, slug string) (*entity.Skin, error) {
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

	var (
		result  *entity.Skin
		mu      sync.Mutex
		wg      sync.WaitGroup
		errChan = make(chan error, r.shardManager.ShardsCount())
	)

	for _, shard := range r.shardManager.GetAllShards() {
		wg.Add(1)
		go func(pool *pgxpool.Pool) {
			defer wg.Done()

			var skin entity.Skin
			err := pool.QueryRow(ctx, query, slug).Scan(
				&skin.ID, &skin.Slug, &skin.MarketHashName, &skin.Name, &skin.Weapon, &skin.Quality, &skin.Rarity,
				&skin.CurrentPrice, &skin.Currency, &skin.ImageURL, &skin.Volume24h,
				&skin.PriceChange24h, &skin.PriceChange7d,
				&skin.LowestPrice, &skin.HighestPrice,
				&skin.LastUpdated, &skin.CreatedAt, &skin.UpdatedAt,
			)

			if err == nil {
				mu.Lock()
				if result == nil {
					result = &skin
				}
				mu.Unlock()
			} else if err != pgx.ErrNoRows {
				errChan <- fmt.Errorf("query shard: %w", err)
			}
		}(shard)
	}

	wg.Wait()
	close(errChan)

	if err := <-errChan; err != nil {
		return nil, err
	}

	if result == nil {
		return nil, fmt.Errorf("skin not found")
	}

	return result, nil
}

func (r *shardedRepository) GetSkinByMarketHashName(ctx context.Context, marketHashName string) (*entity.Skin, error) {
	query := `
		SELECT 
			id, slug, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		WHERE market_hash_name = $1
	`

	var (
		result  *entity.Skin
		mu      sync.Mutex
		wg      sync.WaitGroup
		errChan = make(chan error, r.shardManager.ShardsCount())
	)

	for _, shard := range r.shardManager.GetAllShards() {
		wg.Add(1)
		go func(pool *pgxpool.Pool) {
			defer wg.Done()

			var skin entity.Skin
			err := pool.QueryRow(ctx, query, marketHashName).Scan(
				&skin.ID, &skin.Slug, &skin.MarketHashName, &skin.Name, &skin.Weapon, &skin.Quality, &skin.Rarity,
				&skin.CurrentPrice, &skin.Currency, &skin.ImageURL, &skin.Volume24h,
				&skin.PriceChange24h, &skin.PriceChange7d,
				&skin.LowestPrice, &skin.HighestPrice,
				&skin.LastUpdated, &skin.CreatedAt, &skin.UpdatedAt,
			)

			if err == nil {
				mu.Lock()
				if result == nil {
					result = &skin
				}
				mu.Unlock()
			} else if err != pgx.ErrNoRows {
				errChan <- fmt.Errorf("query shard: %w", err)
			}
		}(shard)
	}

	wg.Wait()
	close(errChan)

	if err := <-errChan; err != nil {
		return nil, err
	}

	if result == nil {
		return nil, fmt.Errorf("skin not found")
	}

	return result, nil
}

func (r *shardedRepository) SkinExists(ctx context.Context, marketHashName string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM skins WHERE market_hash_name = $1)`

	for _, shard := range r.shardManager.GetAllShards() {
		var exists bool
		err := shard.QueryRow(ctx, query, marketHashName).Scan(&exists)
		if err != nil {
			return false, fmt.Errorf("check existence: %w", err)
		}
		if exists {
			return true, nil
		}
	}

	return false, nil
}

func (r *shardedRepository) CreateSkin(ctx context.Context, skin *entity.Skin) error {
	shard := r.shardManager.GetShardByWeapon(skin.Weapon)

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

	_, err := shard.Exec(ctx, query,
		skin.ID, skin.MarketHashName, skin.Name, skin.Weapon, skin.Quality, skin.Rarity,
		skin.CurrentPrice, skin.Currency, skin.ImageURL, skin.Volume24h,
		skin.PriceChange24h, skin.PriceChange7d,
		skin.LowestPrice, skin.HighestPrice,
		skin.LastUpdated, skin.CreatedAt, skin.UpdatedAt,
	)

	return err
}

func (r *shardedRepository) UpdateSkinPrice(ctx context.Context, skinID uuid.UUID, price float64, volume int) error {
	return r.shardManager.TransactionByID(ctx, skinID, func(tx pgx.Tx) error {
		var oldPrice float64
		err := tx.QueryRow(ctx, `SELECT current_price FROM skins WHERE id = $1`, skinID).Scan(&oldPrice)
		if err != nil {
			return fmt.Errorf("get old price: %w", err)
		}

		priceChange := 0.0
		if oldPrice > 0 {
			priceChange = ((price - oldPrice) / oldPrice) * 100
		}

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

		historyQuery := `
			INSERT INTO price_history (skin_id, price, source, volume, recorded_at)
			VALUES ($1, $2, $3, $4, NOW())
		`

		_, err = tx.Exec(ctx, historyQuery, skinID, price, "steam_market", volume)
		return err
	})
}

func (r *shardedRepository) SavePriceHistory(ctx context.Context, history *entity.PriceHistory) error {
	for _, shard := range r.shardManager.GetAllShards() {
		var exists bool
		err := shard.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM skins WHERE id = $1)", history.SkinID).Scan(&exists)
		if err != nil {
			continue
		}

		if exists {
			query := `
				INSERT INTO price_history (skin_id, price, currency, source, volume, recorded_at)
				VALUES ($1, $2, $3, $4, $5, $6)
			`

			_, err := shard.Exec(ctx, query,
				history.SkinID, history.Price, history.Currency,
				history.Source, history.Volume, history.RecordedAt,
			)
			return err
		}
	}

	return fmt.Errorf("skin not found in any shard")
}

func (r *shardedRepository) GetSkinsCount(ctx context.Context) (int, error) {
	var (
		totalCount int
		mu         sync.Mutex
		wg         sync.WaitGroup
		errChan    = make(chan error, r.shardManager.ShardsCount())
	)

	for _, shard := range r.shardManager.GetAllShards() {
		wg.Add(1)
		go func(pool *pgxpool.Pool) {
			defer wg.Done()

			var count int
			err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM skins`).Scan(&count)
			if err != nil {
				errChan <- err
				return
			}

			mu.Lock()
			totalCount += count
			mu.Unlock()
		}(shard)
	}

	wg.Wait()
	close(errChan)

	if err := <-errChan; err != nil {
		return 0, err
	}

	return totalCount, nil
}
