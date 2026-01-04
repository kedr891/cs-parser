package analyticsservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kedr891/cs-parser/internal/models"
)

type AnalyticsStorage interface {
	GetTrendingSkins(ctx context.Context, period string, limit int) ([]models.Skin, error)
	GetTotalSkinsCount(ctx context.Context) (int, error)
	GetAveragePrice(ctx context.Context) (float64, error)
	GetTotalVolume24h(ctx context.Context) (int, error)
	GetTopGainers(ctx context.Context, limit int) ([]models.Skin, error)
	GetTopLosers(ctx context.Context, limit int) ([]models.Skin, error)
	GetMostPopularSkins(ctx context.Context, limit int) ([]models.Skin, error)
	GetRecentlyUpdatedSkins(ctx context.Context, limit int) ([]models.Skin, error)
	GetPriceStatsByPeriod(ctx context.Context, skinID uuid.UUID, period models.PriceStatsPeriod) (*models.SkinStatistics, error)
}

type PriceAnalytics interface {
	UpdateTrending(ctx context.Context, event *models.PriceUpdateEvent) error
	InvalidateMarketOverview(ctx context.Context) error
}

type CacheStorage interface {
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type Service struct {
	storage        AnalyticsStorage
	cache          CacheStorage
	priceAnalytics PriceAnalytics
	log            *slog.Logger
}

func New(
	storage AnalyticsStorage,
	cache CacheStorage,
	priceAnalytics PriceAnalytics,
	log *slog.Logger,
) *Service {
	return &Service{
		storage:        storage,
		cache:          cache,
		priceAnalytics: priceAnalytics,
		log:            log,
	}
}

func (s *Service) GetTrending(ctx context.Context, period string, limit int) ([]models.Skin, error) {
	skins, err := s.storage.GetTrendingSkins(ctx, period, limit)
	if err != nil {
		return nil, fmt.Errorf("get trending skins: %w", err)
	}
	return skins, nil
}

func (s *Service) GetMarketOverview(ctx context.Context) (*models.MarketOverview, error) {
	totalSkins, err := s.storage.GetTotalSkinsCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("get total skins count: %w", err)
	}

	avgPrice, err := s.storage.GetAveragePrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("get average price: %w", err)
	}

	totalVolume, err := s.storage.GetTotalVolume24h(ctx)
	if err != nil {
		return nil, fmt.Errorf("get total volume: %w", err)
	}

	return &models.MarketOverview{
		TotalSkins:     totalSkins,
		AvgPrice:       avgPrice,
		TotalVolume24h: totalVolume,
	}, nil
}

func (s *Service) GetPopularSearches(ctx context.Context, limit int) ([]models.Skin, error) {
	skins, err := s.storage.GetMostPopularSkins(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("get popular skins: %w", err)
	}
	return skins, nil
}

func (s *Service) GetTopGainers(ctx context.Context, limit int) ([]models.Skin, error) {
	return s.storage.GetTopGainers(ctx, limit)
}

func (s *Service) GetTopLosers(ctx context.Context, limit int) ([]models.Skin, error) {
	return s.storage.GetTopLosers(ctx, limit)
}

func (s *Service) GetRecentlyUpdated(ctx context.Context, limit int) ([]models.Skin, error) {
	return s.storage.GetRecentlyUpdatedSkins(ctx, limit)
}
