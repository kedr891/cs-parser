package models

import (
	"time"

	"github.com/google/uuid"
)

type PriceHistory struct {
	ID         uuid.UUID `json:"id" db:"id"`
	SkinID     uuid.UUID `json:"skin_id" db:"skin_id"`
	Price      float64   `json:"price" db:"price"`
	Currency   string    `json:"currency" db:"currency"`
	Source     string    `json:"source" db:"source"`
	Volume     int       `json:"volume" db:"volume"`
	RecordedAt time.Time `json:"recorded_at" db:"recorded_at"`
}

type PriceSource string

const (
	SourceSteamMarket PriceSource = "steam_market"
	SourceCSMoney     PriceSource = "csmoney"
	SourceSkinport    PriceSource = "skinport"
	SourceBuffMarket  PriceSource = "buff_market"
	SourceManual      PriceSource = "manual"
)

func NewPriceHistory(skinID uuid.UUID, price float64, source string, volume int) *PriceHistory {
	return &PriceHistory{
		SkinID:     skinID,
		Price:      price,
		Currency:   "USD",
		Source:     source,
		Volume:     volume,
		RecordedAt: time.Now(),
	}
}

type PriceUpdateEvent struct {
	SkinID         uuid.UUID `json:"skin_id"`
	Slug           string    `json:"slug"`
	MarketHashName string    `json:"market_hash_name"`
	Source         string    `json:"source"`
	OldPrice       float64   `json:"old_price"`
	NewPrice       float64   `json:"new_price"`
	Currency       string    `json:"currency"`
	Volume24h      int       `json:"volume_24h"`
	PriceChange    float64   `json:"price_change"`
	Timestamp      time.Time `json:"timestamp"`
}

func NewPriceUpdateEvent(skinID uuid.UUID, slug, marketHashName, source string, oldPrice, newPrice float64, volume int) *PriceUpdateEvent {
	priceChange := 0.0
	if oldPrice > 0 {
		priceChange = ((newPrice - oldPrice) / oldPrice) * 100
	}

	return &PriceUpdateEvent{
		SkinID:         skinID,
		Slug:           slug,
		MarketHashName: marketHashName,
		Source:         source,
		OldPrice:       oldPrice,
		NewPrice:       newPrice,
		Currency:       "USD",
		Volume24h:      volume,
		PriceChange:    priceChange,
		Timestamp:      time.Now(),
	}
}

func (e *PriceUpdateEvent) IsSignificantChange() bool {
	return e.PriceChange >= 5.0 || e.PriceChange <= -5.0
}

func (e *PriceUpdateEvent) IsPriceDrop() bool {
	return e.NewPrice < e.OldPrice
}

func (e *PriceUpdateEvent) IsPriceIncrease() bool {
	return e.NewPrice > e.OldPrice
}

type SkinDiscoveredEvent struct {
	MarketHashName string    `json:"market_hash_name"`
	Name           string    `json:"name"`
	Weapon         string    `json:"weapon"`
	Quality        string    `json:"quality"`
	Rarity         string    `json:"rarity"`
	InitialPrice   float64   `json:"initial_price"`
	Currency       string    `json:"currency"`
	Source         string    `json:"source"`
	ImageURL       string    `json:"image_url"`
	Timestamp      time.Time `json:"timestamp"`
}

func NewSkinDiscoveredEvent(marketHashName, name, weapon, quality, rarity string, price float64, source, imageURL string) *SkinDiscoveredEvent {
	return &SkinDiscoveredEvent{
		MarketHashName: marketHashName,
		Name:           name,
		Weapon:         weapon,
		Quality:        quality,
		Rarity:         rarity,
		InitialPrice:   price,
		Currency:       "USD",
		Source:         source,
		ImageURL:       imageURL,
		Timestamp:      time.Now(),
	}
}

type PriceChartData struct {
	Timestamp time.Time `json:"timestamp"`
	Price     float64   `json:"price"`
	Volume    int       `json:"volume"`
}

type PriceChartResponse struct {
	SkinID      uuid.UUID        `json:"skin_id"`
	Period      string           `json:"period"` // 24h, 7d, 30d, 90d, 1y, all
	DataPoints  []PriceChartData `json:"data_points"`
	MinPrice    float64          `json:"min_price"`
	MaxPrice    float64          `json:"max_price"`
	AvgPrice    float64          `json:"avg_price"`
	TotalVolume int              `json:"total_volume"`
}

type PriceStatsPeriod string

const (
	Period24h PriceStatsPeriod = "24h"
	Period7d  PriceStatsPeriod = "7d"
	Period30d PriceStatsPeriod = "30d"
	Period90d PriceStatsPeriod = "90d"
	Period1y  PriceStatsPeriod = "1y"
	PeriodAll PriceStatsPeriod = "all"
)

func (p PriceStatsPeriod) GetDuration() time.Duration {
	switch p {
	case Period24h:
		return 24 * time.Hour
	case Period7d:
		return 7 * 24 * time.Hour
	case Period30d:
		return 30 * 24 * time.Hour
	case Period90d:
		return 90 * 24 * time.Hour
	case Period1y:
		return 365 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
}

type PriceComparison struct {
	SkinID         uuid.UUID          `json:"skin_id"`
	MarketHashName string             `json:"market_hash_name"`
	Prices         map[string]float64 `json:"prices"`
	BestPrice      float64            `json:"best_price"`
	BestSource     string             `json:"best_source"`
	PriceDiff      float64            `json:"price_diff"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

type MarketOverview struct {
	TotalSkins      int       `json:"total_skins"`
	AvgPrice        float64   `json:"avg_price"`
	TotalVolume24h  int       `json:"total_volume_24h"`
	TopGainers      []Skin    `json:"top_gainers"`
	TopLosers       []Skin    `json:"top_losers"`
	MostPopular     []Skin    `json:"most_popular"`
	RecentlyUpdated []Skin    `json:"recently_updated"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type PriceAlert struct {
	TargetPrice  float64 `json:"target_price"`
	CurrentPrice float64 `json:"current_price"`
	Condition    string  `json:"condition"`
}

func (a *PriceAlert) ShouldTrigger() bool {
	switch a.Condition {
	case "below":
		return a.CurrentPrice <= a.TargetPrice
	case "above":
		return a.CurrentPrice >= a.TargetPrice
	default:
		return false
	}
}

