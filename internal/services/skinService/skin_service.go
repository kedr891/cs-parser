package skinservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kedr891/cs-parser/internal/models"
)

type SkinStorage interface {
	GetSkins(ctx context.Context, filter *models.SkinFilter) ([]models.Skin, int, error)
	GetSkinBySlug(ctx context.Context, slug string) (*models.Skin, error)
	GetPriceHistory(ctx context.Context, skinID uuid.UUID, period models.PriceStatsPeriod) ([]models.PriceHistory, error)
	GetSkinStatistics(ctx context.Context, skinID uuid.UUID) (*models.SkinStatistics, error)
	SearchSkins(ctx context.Context, query string, limit int) ([]models.Skin, error)
	GetPopularSkins(ctx context.Context, limit int) ([]models.Skin, error)
	CreateSkin(ctx context.Context, skin *models.Skin) error
}

type SkinCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	GetSkinList(ctx context.Context, cacheKey string) (*models.SkinListResponse, bool)
	SetSkinList(ctx context.Context, cacheKey string, response *models.SkinListResponse, ttl time.Duration) error
	GetSkinDetail(ctx context.Context, slug string) (*models.SkinDetailResponse, bool)
	SetSkinDetail(ctx context.Context, slug string, response *models.SkinDetailResponse, ttl time.Duration) error
}

type Service struct {
	storage SkinStorage
	cache   SkinCache
	log     *slog.Logger
}

func New(storage SkinStorage, cache SkinCache, log *slog.Logger) *Service {
	return &Service{
		storage: storage,
		cache:   cache,
		log:     log,
	}
}

func (s *Service) GetSkins(ctx context.Context, filter *models.SkinFilter) (*models.SkinListResponse, error) {
	cacheKey := s.generateCacheKey(filter)

	if cached, ok := s.cache.GetSkinList(ctx, cacheKey); ok {
		s.log.Debug("skins loaded from cache", "cache_key", cacheKey)
		return cached, nil
	}

	skins, total, err := s.storage.GetSkins(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("get skins from storage: %w", err)
	}

	pageSize := filter.Limit
	page := (filter.Offset / pageSize) + 1
	totalPages := (total + pageSize - 1) / pageSize

	response := &models.SkinListResponse{
		Skins:      skins,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	_ = s.cache.SetSkinList(ctx, cacheKey, response, 2*time.Minute)

	return response, nil
}

func (s *Service) GetSkinBySlug(ctx context.Context, slug string, period models.PriceStatsPeriod) (*models.SkinDetailResponse, error) {
	if cached, ok := s.cache.GetSkinDetail(ctx, slug); ok {
		s.incrementViewCount(ctx, cached.Skin.ID)
		return cached, nil
	}

	skin, err := s.storage.GetSkinBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get skin by slug: %w", err)
	}

	priceHistory, err := s.storage.GetPriceHistory(ctx, skin.ID, period)
	if err != nil {
		s.log.Warn("failed to get price history", "slug", slug, "error", err)
		priceHistory = []models.PriceHistory{}
	}

	stats, err := s.storage.GetSkinStatistics(ctx, skin.ID)
	if err != nil {
		s.log.Warn("failed to get skin statistics", "skin_id", skin.ID, "error", err)
		stats = &models.SkinStatistics{}
	}

	viewCount, _ := s.getViewCount(ctx, skin.ID)
	stats.ViewCount = viewCount

	response := &models.SkinDetailResponse{
		Skin:         *skin,
		PriceHistory: priceHistory,
		Statistics:   *stats,
	}

	_ = s.cache.SetSkinDetail(ctx, slug, response, 5*time.Minute)

	s.incrementViewCount(ctx, skin.ID)

	return response, nil
}

func (s *Service) GetPriceChart(ctx context.Context, slug string, period models.PriceStatsPeriod) (*models.PriceChartResponse, error) {
	skin, err := s.storage.GetSkinBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get skin by slug: %w", err)
	}

	history, err := s.storage.GetPriceHistory(ctx, skin.ID, period)
	if err != nil {
		return nil, fmt.Errorf("get price history: %w", err)
	}

	if len(history) == 0 {
		return &models.PriceChartResponse{
			SkinID:     skin.ID,
			Period:     string(period),
			DataPoints: []models.PriceChartData{},
		}, nil
	}

	dataPoints := make([]models.PriceChartData, len(history))
	minPrice := history[0].Price
	maxPrice := history[0].Price

	var sumPrice float64
	var totalVolume int

	for i, h := range history {
		dataPoints[i] = models.PriceChartData{
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

	return &models.PriceChartResponse{
		SkinID:      skin.ID,
		Period:      string(period),
		DataPoints:  dataPoints,
		MinPrice:    minPrice,
		MaxPrice:    maxPrice,
		AvgPrice:    sumPrice / float64(len(history)),
		TotalVolume: totalVolume,
	}, nil
}

func (s *Service) SearchSkins(ctx context.Context, query string, limit int) ([]models.Skin, error) {
	skins, err := s.storage.SearchSkins(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search skins: %w", err)
	}

	key := fmt.Sprintf("analytics:popular:search:%s", query)
	_ = s.cache.Set(ctx, key, "1", 24*time.Hour)

	return skins, nil
}

func (s *Service) GetPopularSkins(ctx context.Context, limit int) ([]models.Skin, error) {
	cacheKey := fmt.Sprintf("skins:popular:%d", limit)

	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var skins []models.Skin
		if err := json.Unmarshal([]byte(cached), &skins); err == nil {
			return skins, nil
		}
	}

	skins, err := s.storage.GetPopularSkins(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("get popular skins: %w", err)
	}

	if data, err := json.Marshal(skins); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), 5*time.Minute)
	}

	return skins, nil
}

func (s *Service) generateCacheKey(filter *models.SkinFilter) string {
	return fmt.Sprintf(
		"skins:list:%s:%s:%.2f-%.2f:%s:%s:%d:%d",
		filter.Weapon,
		filter.Quality,
		filter.MinPrice,
		filter.MaxPrice,
		filter.SortBy,
		filter.SortOrder,
		filter.Limit,
		filter.Offset,
	)
}

func (s *Service) incrementViewCount(ctx context.Context, skinID uuid.UUID) {
	key := fmt.Sprintf("analytics:views:%s", skinID.String())
	_ = s.cache.Set(ctx, key, "1", 24*time.Hour)
}

func (s *Service) getViewCount(ctx context.Context, skinID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("analytics:views:%s", skinID.String())
	_, err := s.cache.Get(ctx, key)
	if err != nil {
		return 0, nil
	}
	return 1, nil
}

func (s *Service) InvalidateSkinCache(ctx context.Context, skinID uuid.UUID) error {
	return s.cache.Delete(ctx, fmt.Sprintf("skin:detail:%s", skinID.String()))
}

func (s *Service) CreateSkin(ctx context.Context, skin *models.Skin) error {
	if err := s.storage.CreateSkin(ctx, skin); err != nil {
		return fmt.Errorf("create skin: %w", err)
	}

	_ = s.cache.Delete(ctx, "skins:list:*")

	return nil
}
