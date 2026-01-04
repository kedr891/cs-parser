package skins_service_api

import (
	"context"

	"github.com/kedr891/cs-parser/internal/models"
	proto_models "github.com/kedr891/cs-parser/internal/pb/models"
	"github.com/kedr891/cs-parser/internal/pb/skins_api"
)

func (s *SkinsServiceAPI) GetSkins(ctx context.Context, req *skins_api.GetSkinsRequest) (*skins_api.GetSkinsResponse, error) {
	filter := &models.SkinFilter{
		Weapon:    req.Weapon,
		Quality:   req.Quality,
		MinPrice:  req.MinPrice,
		MaxPrice:  req.MaxPrice,
		Search:    req.Search,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Limit:     int(req.PageSize),
		Offset:    (int(req.Page) - 1) * int(req.PageSize),
	}

	if filter.Limit == 0 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	response, err := s.skinService.GetSkins(ctx, filter)
	if err != nil {
		return nil, err
	}

	return &skins_api.GetSkinsResponse{
		Skins:      mapSkinsToProto(response.Skins),
		Total:      int32(response.Total),
		Page:       int32(response.Page),
		PageSize:   int32(response.PageSize),
		TotalPages: int32(response.TotalPages),
	}, nil
}

func mapSkinsToProto(skins []models.Skin) []*proto_models.SkinModel {
	result := make([]*proto_models.SkinModel, len(skins))
	for i, skin := range skins {
		result[i] = &proto_models.SkinModel{
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
		}
	}
	return result
}
