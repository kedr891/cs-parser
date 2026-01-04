package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kedr891/cs-parser/internal/models"
)

type WeaponType int

const (
	WeaponTypePistol WeaponType = iota
	WeaponTypeRifle
	WeaponTypeOther
)

type shardConfig struct {
	maxPoolSize  int
	connAttempts int
	connTimeout  time.Duration
}

type ShardOption func(*shardConfig)

const (
	_defaultShardMaxPoolSize  = 10
	_defaultShardConnAttempts = 3
	_defaultShardConnTimeout  = 5 * time.Second
)

func WithShardMaxPoolSize(size int) ShardOption {
	return func(c *shardConfig) {
		c.maxPoolSize = size
	}
}

func WithShardConnAttempts(attempts int) ShardOption {
	return func(c *shardConfig) {
		c.connAttempts = attempts
	}
}

func WithShardConnTimeout(timeout time.Duration) ShardOption {
	return func(c *shardConfig) {
		c.connTimeout = timeout
	}
}

type ShardManager struct {
	shards map[WeaponType]*pgxpool.Pool
}

func NewShardManager(urls []string, opts ...ShardOption) (*ShardManager, error) {
	if len(urls) != 3 {
		return nil, fmt.Errorf("expected 3 shard URLs (pistols, rifles, other), got %d", len(urls))
	}

	cfg := &shardConfig{
		maxPoolSize:  _defaultShardMaxPoolSize,
		connAttempts: _defaultShardConnAttempts,
		connTimeout:  _defaultShardConnTimeout,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	shards := make(map[WeaponType]*pgxpool.Pool)
	weaponTypes := []WeaponType{WeaponTypePistol, WeaponTypeRifle, WeaponTypeOther}
	shardNames := []string{"pistols", "rifles", "other"}

	for i, url := range urls {
		poolConfig, err := pgxpool.ParseConfig(url)
		if err != nil {
			return nil, fmt.Errorf("parse %s shard config: %w", shardNames[i], err)
		}
		poolConfig.MaxConns = int32(cfg.maxPoolSize)

		var pool *pgxpool.Pool
		for attempt := cfg.connAttempts; attempt > 0; attempt-- {
			pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
			if err == nil {
				break
			}
			time.Sleep(cfg.connTimeout)
		}
		if err != nil {
			return nil, fmt.Errorf("connect to %s shard after %d attempts: %w", shardNames[i], cfg.connAttempts, err)
		}
		if err := pool.Ping(context.Background()); err != nil {
			pool.Close()
			return nil, fmt.Errorf("ping %s shard: %w", shardNames[i], err)
		}
		shards[weaponTypes[i]] = pool
	}

	return &ShardManager{shards: shards}, nil
}

func (sm *ShardManager) GetShardByWeapon(weapon string) *pgxpool.Pool {
	weaponType := sm.determineWeaponType(weapon)
	return sm.shards[weaponType]
}

func (sm *ShardManager) GetShardByType(wt WeaponType) *pgxpool.Pool {
	return sm.shards[wt]
}

func (sm *ShardManager) GetAllShards() []*pgxpool.Pool {
	return []*pgxpool.Pool{
		sm.shards[WeaponTypePistol],
		sm.shards[WeaponTypeRifle],
		sm.shards[WeaponTypeOther],
	}
}

func (sm *ShardManager) ShardsCount() int {
	return 3
}

func (sm *ShardManager) CreateSkin(ctx context.Context, skin *models.Skin) error {
	shard := sm.GetShardByWeapon(skin.Weapon)
	query := `
		INSERT INTO skins (
			market_hash_name, slug, name, weapon, quality, rarity,
			image_url, current_price, volume_24h, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`
	return shard.QueryRow(ctx, query,
		skin.MarketHashName,
		skin.Slug,
		skin.Name,
		skin.Weapon,
		skin.Quality,
		skin.Rarity,
		skin.ImageURL,
		skin.CurrentPrice,
		skin.Volume24h,
		skin.CreatedAt,
		skin.UpdatedAt,
	).Scan(&skin.ID)
}

func (sm *ShardManager) GetSkinByID(ctx context.Context, skinID uuid.UUID) (*models.Skin, error) {
	query := `SELECT id, market_hash_name, slug, name, weapon, quality, rarity,
	          image_url, current_price, volume_24h, created_at, updated_at
	          FROM skins WHERE id = $1`

	for _, shard := range sm.GetAllShards() {
		var skin models.Skin
		err := shard.QueryRow(ctx, query, skinID).Scan(
			&skin.ID,
			&skin.MarketHashName,
			&skin.Slug,
			&skin.Name,
			&skin.Weapon,
			&skin.Quality,
			&skin.Rarity,
			&skin.ImageURL,
			&skin.CurrentPrice,
			&skin.Volume24h,
			&skin.CreatedAt,
			&skin.UpdatedAt,
		)
		if err == nil {
			return &skin, nil
		}
		if err != pgx.ErrNoRows {
			return nil, fmt.Errorf("query shard: %w", err)
		}
	}
	return nil, pgx.ErrNoRows
}

func (sm *ShardManager) GetSkinByMarketHashName(ctx context.Context, marketHashName string) (*models.Skin, error) {
	query := `SELECT id, market_hash_name, slug, name, weapon, quality, rarity,
	          image_url, current_price, volume_24h, created_at, updated_at
	          FROM skins WHERE market_hash_name = $1`

	for _, shard := range sm.GetAllShards() {
		var skin models.Skin
		err := shard.QueryRow(ctx, query, marketHashName).Scan(
			&skin.ID,
			&skin.MarketHashName,
			&skin.Slug,
			&skin.Name,
			&skin.Weapon,
			&skin.Quality,
			&skin.Rarity,
			&skin.ImageURL,
			&skin.CurrentPrice,
			&skin.Volume24h,
			&skin.CreatedAt,
			&skin.UpdatedAt,
		)
		if err == nil {
			return &skin, nil
		}
		if err != pgx.ErrNoRows {
			return nil, fmt.Errorf("query shard: %w", err)
		}
	}
	return nil, pgx.ErrNoRows
}

func (sm *ShardManager) UpdateSkinPrice(ctx context.Context, skinID uuid.UUID, price float64, volume int) error {
	query := `UPDATE skins SET current_price = $1, volume_24h = $2, updated_at = $3 WHERE id = $4`
	for _, shard := range sm.GetAllShards() {
		result, err := shard.Exec(ctx, query, price, volume, time.Now(), skinID)
		if err != nil {
			return fmt.Errorf("update in shard: %w", err)
		}
		if result.RowsAffected() > 0 {
			return nil
		}
	}
	return fmt.Errorf("skin not found in any shard")
}

func (sm *ShardManager) GetAllSkins(ctx context.Context) ([]models.Skin, error) {
	query := `SELECT id, market_hash_name, slug, name, weapon, quality, rarity,
	          image_url, current_price, volume_24h, created_at, updated_at
	          FROM skins ORDER BY created_at DESC`

	var allSkins []models.Skin
	for _, shard := range sm.GetAllShards() {
		rows, err := shard.Query(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("query shard: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var skin models.Skin
			err := rows.Scan(
				&skin.ID,
				&skin.MarketHashName,
				&skin.Slug,
				&skin.Name,
				&skin.Weapon,
				&skin.Quality,
				&skin.Rarity,
				&skin.ImageURL,
				&skin.CurrentPrice,
				&skin.Volume24h,
				&skin.CreatedAt,
				&skin.UpdatedAt,
			)
			if err != nil {
				return nil, fmt.Errorf("scan row: %w", err)
			}
			allSkins = append(allSkins, skin)
		}
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("rows error: %w", err)
		}
	}
	return allSkins, nil
}

func (sm *ShardManager) ExecuteOnAllShards(ctx context.Context, query string, args ...interface{}) error {
	for wt, shard := range sm.shards {
		if _, err := shard.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("execute on %s shard: %w", sm.getShardName(wt), err)
		}
	}
	return nil
}

func (sm *ShardManager) Transaction(ctx context.Context, weapon string, fn func(pgx.Tx) error) error {
	shard := sm.GetShardByWeapon(weapon)
	return pgxBegin(ctx, shard, fn)
}

func (sm *ShardManager) TransactionByID(ctx context.Context, skinID uuid.UUID, fn func(pgx.Tx) error) error {
	for _, shard := range sm.GetAllShards() {
		var exists bool
		err := shard.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM skins WHERE id = $1)", skinID).Scan(&exists)
		if err != nil {
			continue
		}
		if exists {
			return pgxBegin(ctx, shard, fn)
		}
	}
	return fmt.Errorf("skin not found in any shard")
}

func (sm *ShardManager) Close() {
	for _, shard := range sm.shards {
		shard.Close()
	}
}

func (sm *ShardManager) determineWeaponType(weapon string) WeaponType {
	weapon = strings.ToLower(weapon)

	pistols := []string{
		"glock-18", "glock",
		"usp-s", "usp",
		"p2000",
		"p250",
		"five-seven", "fiveseven",
		"tec-9", "tec9",
		"cz75-auto", "cz75",
		"desert eagle", "deagle",
		"dual berettas", "dualies",
		"r8 revolver", "r8",
	}
	for _, pistol := range pistols {
		if strings.Contains(weapon, pistol) {
			return WeaponTypePistol
		}
	}

	rifles := []string{
		"ak-47", "ak47",
		"m4a4",
		"m4a1-s", "m4a1",
		"awp",
		"ssg 08", "ssg08", "scout",
		"scar-20", "scar20",
		"g3sg1",
		"aug",
		"sg 553", "sg553",
		"famas",
		"galil ar", "galil",
	}
	for _, rifle := range rifles {
		if strings.Contains(weapon, rifle) {
			return WeaponTypeRifle
		}
	}

	return WeaponTypeOther
}

func (sm *ShardManager) getShardName(wt WeaponType) string {
	switch wt {
	case WeaponTypePistol:
		return "pistols"
	case WeaponTypeRifle:
		return "rifles"
	case WeaponTypeOther:
		return "other"
	default:
		return "unknown"
	}
}

func (sm *ShardManager) GetShardStats(ctx context.Context) (map[string]int, error) {
	stats := make(map[string]int)

	for wt, shard := range sm.shards {
		var count int
		err := shard.QueryRow(ctx, "SELECT COUNT(*) FROM skins").Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("get %s shard stats: %w", sm.getShardName(wt), err)
		}
		stats[sm.getShardName(wt)] = count
	}
	return stats, nil
}

func pgxBegin(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
