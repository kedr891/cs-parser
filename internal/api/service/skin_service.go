package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kedr891/cs-parser/internal/api/repository"
	"github.com/kedr891/cs-parser/internal/domain"
	"github.com/kedr891/cs-parser/internal/entity"
)

// SkinService - сервис для работы со скинами
type SkinService struct {
	repo  repository.SkinRepository
	cache domain.CacheStorage
	log   domain.Logger
}

// NewSkinService - создать сервис скинов
func NewSkinService(
	repo repository.SkinRepository,
	cache domain.CacheStorage,
	log domain.Logger,
) *SkinService {
	return &SkinService{
		repo:  repo,
		cache: cache,
		log:   log,
	}
}

// GetSkins - получить список скинов с кэшированием
func (s *SkinService) GetSkins(ctx context.Context, filter *entity.SkinFilter) (*entity.SkinListResponse, error) {
	// Генерация ключа кэша
	cacheKey := s.generateCacheKey(filter)

	// Проверить кэш
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var response entity.SkinListResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			s.log.Debug("Skins loaded from cache")
			return &response, nil
		}
	}

	// Получить из БД
	skins, total, err := s.repo.GetSkins(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("get skins from repo: %w", err)
	}

	pageSize := filter.Limit
	page := (filter.Offset / pageSize) + 1
	totalPages := (total + pageSize - 1) / pageSize

	response := &entity.SkinListResponse{
		Skins:      skins,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	// Сохранить в кэш
	if data, err := json.Marshal(response); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), 2*time.Minute)
	}

	return response, nil
}

// GetSkinBySlug - получить скин по slug с кэшированием
func (s *SkinService) GetSkinBySlug(ctx context.Context, slug string, period entity.PriceStatsPeriod) (*entity.SkinDetailResponse, error) {
	// Проверить кэш
	cacheKey := fmt.Sprintf("skin:detail:slug:%s", slug)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var response entity.SkinDetailResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			// Инкремент просмотров
			s.incrementViewCount(ctx, response.Skin.ID)
			return &response, nil
		}
	}

	// Получить скин по slug
	skin, err := s.repo.GetSkinBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get skin by slug: %w", err)
	}

	// Получить историю цен
	priceHistory, err := s.repo.GetPriceHistory(ctx, skin.ID, period)
	if err != nil {
		s.log.Warn("Failed to get price history", "error", err)
		priceHistory = []entity.PriceHistory{}
	}

	// Получить статистику
	stats, err := s.repo.GetSkinStatistics(ctx, skin.ID)
	if err != nil {
		s.log.Warn("Failed to get statistics", "error", err)
		stats = &entity.SkinStatistics{}
	}

	// Получить view count из кэша
	viewCount, _ := s.getViewCount(ctx, skin.ID)
	stats.ViewCount = viewCount

	response := &entity.SkinDetailResponse{
		Skin:         *skin,
		PriceHistory: priceHistory,
		Statistics:   *stats,
	}

	// Сохранить в кэш
	if data, err := json.Marshal(response); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), 5*time.Minute)
	}

	// Инкремент просмотров
	s.incrementViewCount(ctx, skin.ID)

	return response, nil
}

// GetPriceChart - получить данные графика цен
func (s *SkinService) GetPriceChart(
	ctx context.Context,
	slug string,
	period entity.PriceStatsPeriod,
) (*entity.PriceChartResponse, error) {

	// 1. Получаем skin по slug
	skin, err := s.repo.GetSkinBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get skin by slug: %w", err)
	}

	// 2. Получаем историю цен по ID
	history, err := s.repo.GetPriceHistory(ctx, skin.ID, period)
	if err != nil {
		return nil, fmt.Errorf("get price history: %w", err)
	}

	// 3. Если истории нет
	if len(history) == 0 {
		return &entity.PriceChartResponse{
			SkinID:     skin.ID,
			Period:     string(period),
			DataPoints: []entity.PriceChartData{},
		}, nil
	}

	// 4. Агрегация
	dataPoints := make([]entity.PriceChartData, len(history))

	minPrice := history[0].Price
	maxPrice := history[0].Price

	var sumPrice float64
	var totalVolume int

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

	return &entity.PriceChartResponse{
		SkinID:      skin.ID,
		Period:      string(period),
		DataPoints:  dataPoints,
		MinPrice:    minPrice,
		MaxPrice:    maxPrice,
		AvgPrice:    sumPrice / float64(len(history)),
		TotalVolume: totalVolume,
	}, nil
}

// SearchSkins - поиск скинов
func (s *SkinService) SearchSkins(ctx context.Context, query string, limit int) ([]entity.Skin, error) {
	skins, err := s.repo.SearchSkins(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search skins: %w", err)
	}

	// Записать популярный запрос (используем rate limit как счетчик)
	searchKey := fmt.Sprintf("analytics:popular:search:%s", query)
	_, _ = s.cache.IncrementRateLimit(ctx, searchKey, 24*time.Hour)

	return skins, nil
}

// GetPopularSkins - получить популярные скины
func (s *SkinService) GetPopularSkins(ctx context.Context, limit int) ([]entity.Skin, error) {
	// Проверить кэш
	cacheKey := fmt.Sprintf("skins:popular:%d", limit)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var skins []entity.Skin
		if err := json.Unmarshal([]byte(cached), &skins); err == nil {
			return skins, nil
		}
	}

	skins, err := s.repo.GetPopularSkins(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("get popular skins: %w", err)
	}

	// Сохранить в кэш
	if data, err := json.Marshal(skins); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), 5*time.Minute)
	}

	return skins, nil
}

// Helper methods

func (s *SkinService) generateCacheKey(filter *entity.SkinFilter) string {
	return fmt.Sprintf("skins:list:%s:%s:%.2f-%.2f:%s:%s:%d:%d",
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

func (s *SkinService) incrementViewCount(ctx context.Context, skinID uuid.UUID) {
	key := fmt.Sprintf("analytics:views:%s", skinID.String())
	_, _ = s.cache.IncrementRateLimit(ctx, key, 24*time.Hour)
}

func (s *SkinService) getViewCount(ctx context.Context, skinID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("analytics:views:%s", skinID.String())
	return s.cache.GetRateLimit(ctx, key)
}

// InvalidateSkinCache - инвалидировать кэш скина
func (s *SkinService) InvalidateSkinCache(ctx context.Context, skinID uuid.UUID) error {
	cacheKey := fmt.Sprintf("skin:detail:%s", skinID.String())
	return s.cache.Delete(ctx, cacheKey)
}
