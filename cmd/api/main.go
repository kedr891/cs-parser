package main

import (
	"fmt"
	"os"

	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/internal/bootstrap"
)

func main() {
	cfg, err := config.LoadConfig(os.Getenv("configPath"))
	if err != nil {
		panic(fmt.Sprintf("ошибка парсинга конфига, %v", err))
	}

	logger := bootstrap.InitLogger(cfg)

	storage := bootstrap.InitPGStorage(cfg)
	cache, closeCache := bootstrap.InitCache(cfg)

	skinService := bootstrap.InitSkinService(storage, cache, logger)
	analyticsService := bootstrap.InitAnalyticsService(storage, logger)

	priceUpdateProcessor := bootstrap.InitPriceUpdateProcessor(analyticsService)
	priceUpdateConsumer := bootstrap.InitPriceUpdateConsumer(cfg, priceUpdateProcessor)

	skinsAPI := bootstrap.InitSkinsServiceAPI(skinService, analyticsService)

	bootstrap.AppRun(*skinsAPI, priceUpdateConsumer, storage, closeCache, logger)
}
