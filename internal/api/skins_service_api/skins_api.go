package skins_service_api

import (
	"context"

	"github.com/kedr891/cs-parser/internal/models"
	"github.com/kedr891/cs-parser/internal/pb/skins_api"
)

type skinService interface {
	GetSkins(ctx context.Context, filter *models.SkinFilter) (*models.SkinListResponse, error)
	GetSkinBySlug(ctx context.Context, slug string, period models.PriceStatsPeriod) (*models.SkinDetailResponse, error)
	SearchSkins(ctx context.Context, query string, limit int) ([]models.Skin, error)
	GetPopularSkins(ctx context.Context, limit int) ([]models.Skin, error)
	GetPriceChart(ctx context.Context, slug string, period models.PriceStatsPeriod) (*models.PriceChartResponse, error)
	CreateSkin(ctx context.Context, skin *models.Skin) error
}

type analyticsService interface {
	GetTrending(ctx context.Context, period string, limit int) ([]models.Skin, error)
	GetMarketOverview(ctx context.Context) (*models.MarketOverview, error)
	GetTopGainers(ctx context.Context, limit int) ([]models.Skin, error)
	GetTopLosers(ctx context.Context, limit int) ([]models.Skin, error)
}

type SkinsServiceAPI struct {
	skins_api.UnimplementedSkinsServiceServer
	skinService      skinService
	analyticsService analyticsService
}

func NewSkinsServiceAPI(skinService skinService, analyticsService analyticsService) *SkinsServiceAPI {
	return &SkinsServiceAPI{
		skinService:      skinService,
		analyticsService: analyticsService,
	}
}
