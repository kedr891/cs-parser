package bootstrap

import (
	analyticsservice "github.com/kedr891/cs-parser/internal/services/analyticsService"
	priceupdateprocessor "github.com/kedr891/cs-parser/internal/services/processors/price_update_processor"
)

func InitPriceUpdateProcessor(analyticsService *analyticsservice.Service) *priceupdateprocessor.PriceUpdateProcessor {
	return priceupdateprocessor.NewPriceUpdateProcessor(analyticsService)
}
