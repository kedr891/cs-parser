package skins_service_api

import (
	"context"

	"github.com/kedr891/cs-parser/internal/models"
	proto_models "github.com/kedr891/cs-parser/internal/pb/models"
	"github.com/kedr891/cs-parser/internal/pb/skins_api"
)

func (s *SkinsServiceAPI) GetSkinBySlug(ctx context.Context, req *skins_api.GetSkinBySlugRequest) (*skins_api.GetSkinBySlugResponse, error) {
	period := models.Period7d
	if req.Period != "" {
		period = models.PriceStatsPeriod(req.Period)
	}

	response, err := s.skinService.GetSkinBySlug(ctx, req.Slug, period)
	if err != nil {
		return nil, err
	}

	return &skins_api.GetSkinBySlugResponse{
		Skin: mapSkinDetailToProto(response),
	}, nil
}

func mapSkinDetailToProto(detail *models.SkinDetailResponse) *proto_models.SkinDetailModel {
	priceHistory := make([]*proto_models.PriceHistoryModel, len(detail.PriceHistory))
	for i, ph := range detail.PriceHistory {
		priceHistory[i] = &proto_models.PriceHistoryModel{
			Id:         ph.ID.String(),
			SkinId:     ph.SkinID.String(),
			Price:      ph.Price,
			Currency:   ph.Currency,
			Source:     ph.Source,
			Volume:     int32(ph.Volume),
			RecordedAt: ph.RecordedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return &proto_models.SkinDetailModel{
		Skin: &proto_models.SkinModel{
			Id:              detail.Skin.ID.String(),
			MarketHashName:  detail.Skin.MarketHashName,
			Name:            detail.Skin.Name,
			Weapon:          detail.Skin.Weapon,
			Quality:         detail.Skin.Quality,
			Rarity:          detail.Skin.Rarity,
			CurrentPrice:    detail.Skin.CurrentPrice,
			Currency:        detail.Skin.Currency,
			ImageUrl:        detail.Skin.ImageURL,
			Volume_24H:      int32(detail.Skin.Volume24h),
			PriceChange_24H: detail.Skin.PriceChange24h,
			PriceChange_7D:  detail.Skin.PriceChange7d,
			Slug:            detail.Skin.Slug,
			CreatedAt:       detail.Skin.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:       detail.Skin.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		},
		Statistics: &proto_models.SkinStatisticsModel{
			AvgPrice_7D:     detail.Statistics.AvgPrice7d,
			AvgPrice_30D:    detail.Statistics.AvgPrice30d,
			TotalVolume_7D:  int32(detail.Statistics.TotalVolume7d),
			TotalVolume_30D: 0,
			PriceVolatility: detail.Statistics.PriceVolatility,
			MinPrice_7D:     0,
			MaxPrice_7D:     0,
		},
		PriceHistory: priceHistory,
	}
}
