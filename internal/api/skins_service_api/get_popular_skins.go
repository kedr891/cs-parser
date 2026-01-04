package skins_service_api

import (
	"context"

	"github.com/kedr891/cs-parser/internal/pb/skins_api"
)

func (s *SkinsServiceAPI) GetPopularSkins(ctx context.Context, req *skins_api.GetPopularSkinsRequest) (*skins_api.GetPopularSkinsResponse, error) {
	limit := int(req.Limit)
	if limit == 0 {
		limit = 10
	}

	skins, err := s.skinService.GetPopularSkins(ctx, limit)
	if err != nil {
		return nil, err
	}

	return &skins_api.GetPopularSkinsResponse{
		Skins: mapSkinsToProto(skins),
	}, nil
}
