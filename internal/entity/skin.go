package entity

import (
	"time"

	"github.com/google/uuid"
)

// Skin - модель скина CS2
type Skin struct {
	ID             uuid.UUID `json:"id" db:"id"`
	MarketHashName string    `json:"market_hash_name" db:"market_hash_name"`
	Name           string    `json:"name" db:"name"`
	Weapon         string    `json:"weapon" db:"weapon"`
	Quality        string    `json:"quality" db:"quality"`
	Rarity         string    `json:"rarity" db:"rarity"`
	CurrentPrice   float64   `json:"current_price" db:"current_price"`
	Currency       string    `json:"currency" db:"currency"`
	ImageURL       string    `json:"image_url" db:"image_url"`
	Volume24h      int       `json:"volume_24h" db:"volume_24h"`
	PriceChange24h float64   `json:"price_change_24h" db:"price_change_24h"`
	PriceChange7d  float64   `json:"price_change_7d" db:"price_change_7d"`
	LowestPrice    float64   `json:"lowest_price" db:"lowest_price"`
	HighestPrice   float64   `json:"highest_price" db:"highest_price"`
	LastUpdated    time.Time `json:"last_updated" db:"last_updated"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// SkinQuality - качество скина
type SkinQuality string

const (
	QualityFactoryNew    SkinQuality = "Factory New"
	QualityMinimalWear   SkinQuality = "Minimal Wear"
	QualityFieldTested   SkinQuality = "Field-Tested"
	QualityWellWorn      SkinQuality = "Well-Worn"
	QualityBattleScarred SkinQuality = "Battle-Scarred"
	QualityNotPainted    SkinQuality = "Not Painted"
)

// SkinRarity - редкость скина
type SkinRarity string

const (
	RarityConsumerGrade   SkinRarity = "Consumer Grade"
	RarityIndustrialGrade SkinRarity = "Industrial Grade"
	RarityMilSpecGrade    SkinRarity = "Mil-Spec Grade"
	RarityRestricted      SkinRarity = "Restricted"
	RarityClassified      SkinRarity = "Classified"
	RarityCovert          SkinRarity = "Covert"
	RarityContraband      SkinRarity = "Contraband"
)

// SkinCategory - категория оружия
type SkinCategory string

const (
	CategoryRifle      SkinCategory = "Rifle"
	CategoryPistol     SkinCategory = "Pistol"
	CategorySMG        SkinCategory = "SMG"
	CategorySniper     SkinCategory = "Sniper"
	CategoryShotgun    SkinCategory = "Shotgun"
	CategoryMachinegun SkinCategory = "Machinegun"
	CategoryKnife      SkinCategory = "Knife"
	CategoryGloves     SkinCategory = "Gloves"
)

// NewSkin - создать новый скин
func NewSkin(marketHashName, name, weapon, quality string) *Skin {
	now := time.Now()
	return &Skin{
		ID:             uuid.New(),
		MarketHashName: marketHashName,
		Name:           name,
		Weapon:         weapon,
		Quality:        quality,
		Currency:       "USD",
		LastUpdated:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// UpdatePrice - обновить цену скина
func (s *Skin) UpdatePrice(newPrice float64, volume int) {
	s.CurrentPrice = newPrice
	s.Volume24h = volume
	s.LastUpdated = time.Now()
	s.UpdatedAt = time.Now()
}

// CalculatePriceChange - рассчитать изменение цены
func (s *Skin) CalculatePriceChange(oldPrice float64) float64 {
	if oldPrice == 0 {
		return 0
	}
	return ((s.CurrentPrice - oldPrice) / oldPrice) * 100
}

// IsPopular - популярный ли скин (высокий объём торгов)
func (s *Skin) IsPopular() bool {
	return s.Volume24h > 100
}

// IsTrending - в тренде ли скин (рост цены)
func (s *Skin) IsTrending() bool {
	return s.PriceChange24h > 5.0
}

// GetCategory - получить категорию оружия
func (s *Skin) GetCategory() SkinCategory {
	weapons := map[string]SkinCategory{
		"AK-47":           CategoryRifle,
		"M4A4":            CategoryRifle,
		"M4A1-S":          CategoryRifle,
		"AWP":             CategorySniper,
		"Desert Eagle":    CategoryPistol,
		"Glock-18":        CategoryPistol,
		"USP-S":           CategoryPistol,
		"P250":            CategoryPistol,
		"MP9":             CategorySMG,
		"MAC-10":          CategorySMG,
		"UMP-45":          CategorySMG,
		"Nova":            CategoryShotgun,
		"XM1014":          CategoryShotgun,
		"Negev":           CategoryMachinegun,
		"M249":            CategoryMachinegun,
		"Karambit":        CategoryKnife,
		"Bayonet":         CategoryKnife,
		"Butterfly Knife": CategoryKnife,
	}

	if category, ok := weapons[s.Weapon]; ok {
		return category
	}

	// Проверка на перчатки
	if len(s.Weapon) > 6 && s.Weapon[:6] == "Gloves" {
		return CategoryGloves
	}

	// Проверка на ножи (содержит "Knife")
	if len(s.Weapon) > 5 && (s.Weapon[len(s.Weapon)-5:] == "Knife" || s.Weapon[:5] == "Knife") {
		return CategoryKnife
	}

	return CategoryRifle // по умолчанию
}

// SkinFilter - фильтр для поиска скинов
type SkinFilter struct {
	Weapon    string
	Quality   string
	Rarity    string
	MinPrice  float64
	MaxPrice  float64
	Search    string
	SortBy    string // price, volume, name, updated
	SortOrder string // asc, desc
	Limit     int
	Offset    int
}

// NewSkinFilter - создать фильтр по умолчанию
func NewSkinFilter() *SkinFilter {
	return &SkinFilter{
		SortBy:    "updated",
		SortOrder: "desc",
		Limit:     50,
		Offset:    0,
	}
}

// SkinListResponse - ответ со списком скинов
type SkinListResponse struct {
	Skins      []Skin `json:"skins"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int    `json:"total_pages"`
}

// SkinDetailResponse - детальная информация о скине
type SkinDetailResponse struct {
	Skin         Skin           `json:"skin"`
	PriceHistory []PriceHistory `json:"price_history"`
	Statistics   SkinStatistics `json:"statistics"`
}

// SkinStatistics - статистика по скину
type SkinStatistics struct {
	AvgPrice7d      float64 `json:"avg_price_7d"`
	AvgPrice30d     float64 `json:"avg_price_30d"`
	TotalVolume7d   int     `json:"total_volume_7d"`
	PriceVolatility float64 `json:"price_volatility"`
	ViewCount       int64   `json:"view_count"`
}

// TrendingSkin - скин в трендах
type TrendingSkin struct {
	Skin            Skin    `json:"skin"`
	PriceChangeRate float64 `json:"price_change_rate"`
	Rank            int     `json:"rank"`
}
