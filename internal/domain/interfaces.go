package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kedr891/cs-parser/internal/entity"
)

type SkinRepository interface {
	GetAllSkins(ctx context.Context) ([]entity.Skin, error)
	GetSkinBySlug(ctx context.Context, slug string) (*entity.Skin, error)
	GetSkinByMarketHashName(ctx context.Context, marketHashName string) (*entity.Skin, error)
	SkinExists(ctx context.Context, marketHashName string) (bool, error)
	CreateSkin(ctx context.Context, skin *entity.Skin) error
	UpdateSkinPrice(ctx context.Context, skinID uuid.UUID, price float64, volume int) error
	SavePriceHistory(ctx context.Context, history *entity.PriceHistory) error
	GetSkinsCount(ctx context.Context) (int, error)
}

type MarketClient interface {
	GetItemPrice(ctx context.Context, marketHashName string) (*PriceData, error)
	SearchItems(ctx context.Context, query string) ([]MarketItem, error)
}

type PriceData struct {
	Price  float64
	Volume int
}

type MarketItem struct {
	MarketHashName string
	Name           string
	Weapon         string
	Quality        string
	Rarity         string
	Price          float64
	ImageURL       string
}

type CacheStorage interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	IncrementRateLimit(ctx context.Context, key string, ttl time.Duration) (int64, error)
	GetRateLimit(ctx context.Context, key string) (int64, error)

	Increment(ctx context.Context, key string) (int64, error)

	SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	GetJSON(ctx context.Context, key string, dest interface{}) error

	ZAdd(ctx context.Context, key string, score float64, member string) error
	ZIncrBy(ctx context.Context, key string, increment float64, member string) (float64, error)
	ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]ZMember, error)

	HSet(ctx context.Context, key, field string, value interface{}) error
	HGet(ctx context.Context, key, field string) (string, error)
	HGetAll(ctx context.Context, key string) (map[string]string, error)
}

type ZMember struct {
	Member string
	Score  float64
}

type MessageProducer interface {
	WriteMessage(ctx context.Context, key string, value interface{}) error
	Close() error
}

type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})
}
