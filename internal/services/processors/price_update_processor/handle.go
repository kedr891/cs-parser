package priceupdateprocessor

import (
	"context"

	"github.com/kedr891/cs-parser/internal/models"
)

func (p *PriceUpdateProcessor) Handle(ctx context.Context, event *models.PriceUpdateEvent) error {
	return p.analyticsService.ProcessPriceUpdate(ctx, event)
}
