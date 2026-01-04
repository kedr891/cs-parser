package analyticsservice

import (
	"context"

	"github.com/kedr891/cs-parser/internal/models"
)

func (s *Service) ProcessPriceUpdate(ctx context.Context, event *models.PriceUpdateEvent) error {
	s.log.Debug("Processing price update",
		"skin_id", event.SkinID,
		"old_price", event.OldPrice,
		"new_price", event.NewPrice,
		"change", event.PriceChange,
	)

	if err := s.priceAnalytics.UpdateTrending(ctx, event); err != nil {
		s.log.Warn("Failed to update trending", "error", err)
	}

	if event.IsSignificantChange() {
		if err := s.priceAnalytics.InvalidateMarketOverview(ctx); err != nil {
			s.log.Warn("Failed to invalidate market overview", "error", err)
		}
	}

	s.log.Info("Price update processed successfully",
		"skin_id", event.SkinID,
		"price_change", event.PriceChange,
	)

	return nil
}
