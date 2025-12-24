package entity

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Skin struct {
	ID             uuid.UUID `json:"id" db:"id"`
	Slug           string    `json:"slug" db:"slug"`
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

// Пример: "AWP | Acheron (Field-Tested)" -> "awp_acheron_ft"
func GenerateSlug(marketHashName string) string {
	slug := strings.ToLower(marketHashName)

	qualityReplacements := map[string]string{
		"(factory new)":    "fn",
		"(minimal wear)":   "mw",
		"(field-tested)":   "ft",
		"(well-worn)":      "ww",
		"(battle-scarred)": "bs",
	}

	for full, abbr := range qualityReplacements {
		slug = strings.ReplaceAll(slug, full, abbr)
	}

	slug = regexp.MustCompile(`[|()\[\]{}]`).ReplaceAllString(slug, "")

	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "_")

	slug = strings.Trim(slug, "_")
	slug = regexp.MustCompile(`_+`).ReplaceAllString(slug, "_")

	return slug
}

type SkinQuality string

const (
	QualityFactoryNew    SkinQuality = "Factory New"
	QualityMinimalWear   SkinQuality = "Minimal Wear"
	QualityFieldTested   SkinQuality = "Field-Tested"
	QualityWellWorn      SkinQuality = "Well-Worn"
	QualityBattleScarred SkinQuality = "Battle-Scarred"
	QualityNotPainted    SkinQuality = "Not Painted"
)

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

func NewSkin(marketHashName, name, weapon, quality string) *Skin {
	now := time.Now()
	return &Skin{
		ID:             uuid.New(),
		Slug:           GenerateSlug(marketHashName),
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

func (s *Skin) UpdatePrice(newPrice float64, volume int) {
	s.CurrentPrice = newPrice
	s.Volume24h = volume
	s.LastUpdated = time.Now()
	s.UpdatedAt = time.Now()
}

func (s *Skin) CalculatePriceChange(oldPrice float64) float64 {
	if oldPrice == 0 {
		return 0
	}
	return ((s.CurrentPrice - oldPrice) / oldPrice) * 100
}

func (s *Skin) IsPopular() bool {
	return s.Volume24h > 100
}

func (s *Skin) IsTrending() bool {
	return s.PriceChange24h > 5.0
}

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

	if len(s.Weapon) > 6 && s.Weapon[:6] == "Gloves" {
		return CategoryGloves
	}

	if len(s.Weapon) > 5 && (s.Weapon[len(s.Weapon)-5:] == "Knife" || s.Weapon[:5] == "Knife") {
		return CategoryKnife
	}

	return CategoryRifle
}

type SkinFilter struct {
	Weapon    string
	Quality   string
	Rarity    string
	MinPrice  float64
	MaxPrice  float64
	Search    string
	SortBy    string
	SortOrder string
	Limit     int
	Offset    int
}

func NewSkinFilter() *SkinFilter {
	return &SkinFilter{
		SortBy:    "updated",
		SortOrder: "desc",
		Limit:     50,
		Offset:    0,
	}
}

type SkinListResponse struct {
	Skins      []Skin `json:"skins"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int    `json:"total_pages"`
}

type SkinDetailResponse struct {
	Skin         Skin           `json:"skin"`
	PriceHistory []PriceHistory `json:"price_history"`
	Statistics   SkinStatistics `json:"statistics"`
}

type SkinStatistics struct {
	AvgPrice7d      float64 `json:"avg_price_7d"`
	AvgPrice30d     float64 `json:"avg_price_30d"`
	TotalVolume7d   int     `json:"total_volume_7d"`
	PriceVolatility float64 `json:"price_volatility"`
	ViewCount       int64   `json:"view_count"`
}

type TrendingSkin struct {
	Skin            Skin    `json:"skin"`
	PriceChangeRate float64 `json:"price_change_rate"`
	Rank            int     `json:"rank"`
}

type SkinSlugResponse struct {
	ID             uuid.UUID `json:"id"`
	Slug           string    `json:"slug"`
	MarketHashName string    `json:"market_hash_name"`
	Name           string    `json:"name"`
	CurrentPrice   float64   `json:"current_price"`
	ImageURL       string    `json:"image_url"`
}

func (s *Skin) ToSlugResponse() SkinSlugResponse {
	return SkinSlugResponse{
		ID:             s.ID,
		Slug:           s.Slug,
		MarketHashName: s.MarketHashName,
		Name:           s.Name,
		CurrentPrice:   s.CurrentPrice,
		ImageURL:       s.ImageURL,
	}
}
