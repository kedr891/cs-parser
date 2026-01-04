package skins_service_api

import (
	"context"

	"github.com/kedr891/cs-parser/internal/pb/skins_api"
)

func (s *SkinsServiceAPI) SearchSkins(ctx context.Context, req *skins_api.SearchSkinsRequest) (*skins_api.SearchSkinsResponse, error) {
	limit := int(req.Limit)
	if limit == 0 {
		limit = 20
	}

	skins, err := s.skinService.SearchSkins(ctx, req.Query, limit)
	if err != nil {
		return nil, err
	}

	return &skins_api.SearchSkinsResponse{
		Skins: mapSkinsToProto(skins),
	}, nil
}
