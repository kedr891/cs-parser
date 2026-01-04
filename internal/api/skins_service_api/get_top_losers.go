package skins_service_api

import (
	"context"

	"github.com/kedr891/cs-parser/internal/pb/skins_api"
)

func (s *SkinsServiceAPI) GetTopLosers(ctx context.Context, req *skins_api.GetTopLosersRequest) (*skins_api.GetTopLosersResponse, error) {
	limit := int(req.Limit)
	if limit == 0 {
		limit = 10
	}

	skins, err := s.analyticsService.GetTopLosers(ctx, limit)
	if err != nil {
		return nil, err
	}

	return &skins_api.GetTopLosersResponse{
		Skins: mapSkinsToProto(skins),
	}, nil
}
