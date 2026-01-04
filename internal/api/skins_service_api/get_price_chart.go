package skins_service_api

import (
	"context"

	"github.com/kedr891/cs-parser/internal/models"
	proto_models "github.com/kedr891/cs-parser/internal/pb/models"
	"github.com/kedr891/cs-parser/internal/pb/skins_api"
)

func (s *SkinsServiceAPI) GetPriceChart(ctx context.Context, req *skins_api.GetPriceChartRequest) (*skins_api.GetPriceChartResponse, error) {
	period := models.Period7d
	if req.Period != "" {
		period = models.PriceStatsPeriod(req.Period)
	}

	chartData, err := s.skinService.GetPriceChart(ctx, req.Slug, period)
	if err != nil {
		return nil, err
	}

	dataPoints := make([]*proto_models.PriceChartDataModel, len(chartData.DataPoints))
	for i, dp := range chartData.DataPoints {
		dataPoints[i] = &proto_models.PriceChartDataModel{
			Timestamp: dp.Timestamp.Format("2006-01-02T15:04:05Z"),
			Price:     dp.Price,
			Volume:    int32(dp.Volume),
		}
	}

	return &skins_api.GetPriceChartResponse{
		SkinId:      chartData.SkinID.String(),
		Period:      chartData.Period,
		DataPoints:  dataPoints,
		MinPrice:    chartData.MinPrice,
		MaxPrice:    chartData.MaxPrice,
		AvgPrice:    chartData.AvgPrice,
		TotalVolume: int32(chartData.TotalVolume),
	}, nil
}
