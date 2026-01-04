package skins_service_api

import (
	"context"

	"github.com/kedr891/cs-parser/internal/pb/skins_api"
)

func (s *SkinsServiceAPI) GetTopGainers(ctx context.Context, req *skins_api.GetTopGainersRequest) (*skins_api.GetTopGainersResponse, error) {
	limit := int(req.Limit)
	if limit == 0 {
		limit = 10
	}

	skins, err := s.analyticsService.GetTopGainers(ctx, limit)
	if err != nil {
		return nil, err
	}

	return &skins_api.GetTopGainersResponse{
		Skins: mapSkinsToProto(skins),
	}, nil
}
