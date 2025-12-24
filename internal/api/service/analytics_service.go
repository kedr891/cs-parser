package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kedr891/cs-parser/internal/api/repository"
	"github.com/kedr891/cs-parser/internal/domain"
	"github.com/kedr891/cs-parser/internal/entity"
)

type AnalyticsService struct {
	repo  repository.AnalyticsRepository
	cache domain.CacheStorage
	log   domain.Logger
}

func NewAnalyticsService(
	repo repository.AnalyticsRepository,
	cache domain.CacheStorage,
	log domain.Logger,
) *AnalyticsService {
	return &AnalyticsService{
		repo:  repo,
		cache: cache,
		log:   log,
	}
}

func (s *AnalyticsService) GetTrendingSkins(ctx context.Context, period string, limit int) ([]entity.Skin, error) {
	cacheKey := fmt.Sprintf("analytics:trending:%s:%d", period, limit)
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var skins []entity.Skin
		if err := json.Unmarshal([]byte(cached), &skins); err == nil {
			return skins, nil
		}
	}

	skins, err := s.repo.GetTrendingSkins(ctx, period, limit)
	if err != nil {
		return nil, fmt.Errorf("get trending skins: %w", err)
	}

	if data, err := json.Marshal(skins); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), 5*time.Minute)
	}

	return skins, nil
}

func (s *AnalyticsService) GetMarketOverview(ctx context.Context) (*entity.MarketOverview, error) {
	cacheKey := "analytics:market:overview"
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var overview entity.MarketOverview
		if err := json.Unmarshal([]byte(cached), &overview); err == nil {
			return &overview, nil
		}
	}

	totalSkins, err := s.repo.GetTotalSkinsCount(ctx)
	if err != nil {
		s.log.Warn("Failed to get total skins count", "error", err)
		totalSkins = 0
	}

	avgPrice, err := s.repo.GetAveragePrice(ctx)
	if err != nil {
		s.log.Warn("Failed to get average price", "error", err)
		avgPrice = 0
	}

	totalVolume, err := s.repo.GetTotalVolume24h(ctx)
	if err != nil {
		s.log.Warn("Failed to get total volume", "error", err)
		totalVolume = 0
	}

	topGainers, err := s.repo.GetTopGainers(ctx, 5)
	if err != nil {
		s.log.Warn("Failed to get top gainers", "error", err)
		topGainers = []entity.Skin{}
	}

	topLosers, err := s.repo.GetTopLosers(ctx, 5)
	if err != nil {
		s.log.Warn("Failed to get top losers", "error", err)
		topLosers = []entity.Skin{}
	}

	mostPopular, err := s.repo.GetMostPopularSkins(ctx, 5)
	if err != nil {
		s.log.Warn("Failed to get most popular", "error", err)
		mostPopular = []entity.Skin{}
	}

	recentlyUpdated, err := s.repo.GetRecentlyUpdatedSkins(ctx, 5)
	if err != nil {
		s.log.Warn("Failed to get recently updated", "error", err)
		recentlyUpdated = []entity.Skin{}
	}

	overview := &entity.MarketOverview{
		TotalSkins:      totalSkins,
		AvgPrice:        avgPrice,
		TotalVolume24h:  totalVolume,
		TopGainers:      topGainers,
		TopLosers:       topLosers,
		MostPopular:     mostPopular,
		RecentlyUpdated: recentlyUpdated,
	}

	if data, err := json.Marshal(overview); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(data), 10*time.Minute)
	}

	return overview, nil
}

func (s *AnalyticsService) GetPopularSearches(ctx context.Context, limit int) ([]string, error) {
	cacheKey := "analytics:popular:searches"

	searches, err := s.cache.ZRevRange(ctx, cacheKey, 0, int64(limit-1))
	if err != nil {
		s.log.Warn("Failed to get popular searches from Redis", "error", err)
		return []string{}, nil
	}

	return searches, nil
}
