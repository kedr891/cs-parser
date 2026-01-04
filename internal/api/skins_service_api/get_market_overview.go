package skins_service_api

import (
	"context"

	proto_models "github.com/kedr891/cs-parser/internal/pb/models"
	"github.com/kedr891/cs-parser/internal/pb/skins_api"
)

func (s *SkinsServiceAPI) GetMarketOverview(ctx context.Context, req *skins_api.GetMarketOverviewRequest) (*skins_api.GetMarketOverviewResponse, error) {
	overview, err := s.analyticsService.GetMarketOverview(ctx)
	if err != nil {
		return nil, err
	}

	return &skins_api.GetMarketOverviewResponse{
		Overview: &proto_models.MarketOverviewModel{
			TotalSkins:      int32(overview.TotalSkins),
			AvgPrice:        overview.AvgPrice,
			TotalVolume_24H: int32(overview.TotalVolume24h),
		},
	}, nil
}
