package skins_service_api

import (
	"context"

	proto_models "github.com/kedr891/cs-parser/internal/pb/models"
	"github.com/kedr891/cs-parser/internal/pb/skins_api"
)

func (s *SkinsServiceAPI) GetTrending(ctx context.Context, req *skins_api.GetTrendingRequest) (*skins_api.GetTrendingResponse, error) {
	period := req.Period
	if period == "" {
		period = "24h"
	}

	limit := int(req.Limit)
	if limit == 0 {
		limit = 20
	}

	skins, err := s.analyticsService.GetTrending(ctx, period, limit)
	if err != nil {
		return nil, err
	}

	trendingSkins := make([]*proto_models.TrendingSkinModel, len(skins))
	for i, skin := range skins {
		trendingSkins[i] = &proto_models.TrendingSkinModel{
			Rank: int32(i + 1),
			Skin: &proto_models.SkinModel{
				Id:              skin.ID.String(),
				MarketHashName:  skin.MarketHashName,
				Name:            skin.Name,
				Weapon:          skin.Weapon,
				Quality:         skin.Quality,
				Rarity:          skin.Rarity,
				CurrentPrice:    skin.CurrentPrice,
				Currency:        skin.Currency,
				ImageUrl:        skin.ImageURL,
				Volume_24H:      int32(skin.Volume24h),
				PriceChange_24H: skin.PriceChange24h,
				PriceChange_7D:  skin.PriceChange7d,
				Slug:            skin.Slug,
				CreatedAt:       skin.CreatedAt.Format("2006-01-02T15:04:05Z"),
				UpdatedAt:       skin.UpdatedAt.Format("2006-01-02T15:04:05Z"),
			},
			PriceChangeRate: skin.PriceChange24h,
		}
	}

	return &skins_api.GetTrendingResponse{
		TrendingSkins: trendingSkins,
	}, nil
}
