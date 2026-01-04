package priceupdateprocessor

import (
	"context"

	"github.com/kedr891/cs-parser/internal/models"
)

type analyticsService interface {
	ProcessPriceUpdate(ctx context.Context, event *models.PriceUpdateEvent) error
}

type PriceUpdateProcessor struct {
	analyticsService analyticsService
}

func NewPriceUpdateProcessor(analyticsService analyticsService) *PriceUpdateProcessor {
	return &PriceUpdateProcessor{
		analyticsService: analyticsService,
	}
}
