package sharding

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WeaponType int

const (
	WeaponTypePistol WeaponType = iota
	WeaponTypeRifle
	WeaponTypeOther
)

type WeaponShardManager struct {
	shards map[WeaponType]*pgxpool.Pool
}

func NewWeaponShardManagerFromURLs(ctx context.Context, urls []string) (*WeaponShardManager, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("шарды не настроены")
	}

	if len(urls) != 3 {
		return nil, fmt.Errorf("expected 3 shard URLs (pistols, rifles, other), got %d", len(urls))
	}

	manager := &WeaponShardManager{
		shards: make(map[WeaponType]*pgxpool.Pool),
	}

	weaponTypes := []WeaponType{WeaponTypePistol, WeaponTypeRifle, WeaponTypeOther}
	shardNames := []string{"pistols", "rifles", "other"}

	for i, url := range urls {
		pool, err := newShardPool(ctx, url)
		if err != nil {
			manager.closeUntil(i)
			return nil, fmt.Errorf("создание шарда %s: %w", shardNames[i], err)
		}
		manager.shards[weaponTypes[i]] = pool
	}

	return manager, nil
}

func newShardPool(ctx context.Context, connURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connURL)
	if err != nil {
		return nil, fmt.Errorf("парсинг конфигурации postgres: %w", err)
	}

	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("создание пула postgres: %w", err)
	}
	return pool, nil
}

func (m *WeaponShardManager) ShardCount() int {
	return len(m.shards)
}

func (m *WeaponShardManager) GetShardByWeapon(weapon string) *pgxpool.Pool {
	weaponType := m.determineWeaponType(weapon)
	return m.shards[weaponType]
}

func (m *WeaponShardManager) GetShardByType(wt WeaponType) *pgxpool.Pool {
	return m.shards[wt]
}

func (m *WeaponShardManager) Primary() *pgxpool.Pool {
	return m.GetShardByType(WeaponTypePistol)
}

func (m *WeaponShardManager) AllShards() []*pgxpool.Pool {
	result := make([]*pgxpool.Pool, 3)
	result[0] = m.shards[WeaponTypePistol]
	result[1] = m.shards[WeaponTypeRifle]
	result[2] = m.shards[WeaponTypeOther]
	return result
}

func (m *WeaponShardManager) Close() {
	for _, pool := range m.shards {
		if pool != nil {
			pool.Close()
		}
	}
}

func (m *WeaponShardManager) closeUntil(idx int) {
	weaponTypes := []WeaponType{WeaponTypePistol, WeaponTypeRifle, WeaponTypeOther}
	for i := 0; i < idx && i < len(weaponTypes); i++ {
		if m.shards[weaponTypes[i]] != nil {
			m.shards[weaponTypes[i]].Close()
		}
	}
}

func (m *WeaponShardManager) determineWeaponType(weapon string) WeaponType {
	weapon = strings.ToLower(weapon)

	pistols := []string{
		"glock-18", "glock", "usp-s", "usp", "p2000", "p250",
		"five-seven", "fiveseven", "tec-9", "tec9", "cz75-auto", "cz75",
		"desert eagle", "deagle", "dual berettas", "dualies", "r8 revolver", "r8",
	}

	for _, pistol := range pistols {
		if strings.Contains(weapon, pistol) {
			return WeaponTypePistol
		}
	}

	rifles := []string{
		"ak-47", "ak47", "m4a4", "m4a1-s", "m4a1", "awp",
		"ssg 08", "ssg08", "scout", "scar-20", "scar20", "g3sg1",
		"aug", "sg 553", "sg553", "famas", "galil ar", "galil",
	}

	for _, rifle := range rifles {
		if strings.Contains(weapon, rifle) {
			return WeaponTypeRifle
		}
	}

	return WeaponTypeOther
}

func (m *WeaponShardManager) Transaction(ctx context.Context, weapon string, fn func(pgx.Tx) error) error {
	shard := m.GetShardByWeapon(weapon)
	if shard == nil {
		return fmt.Errorf("шард не найден для оружия %s", weapon)
	}

	tx, err := shard.Begin(ctx)
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
