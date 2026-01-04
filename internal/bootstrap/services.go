package bootstrap

import (
	"log/slog"
	"os"

	"github.com/kedr891/cs-parser/config"
	analyticsservice "github.com/kedr891/cs-parser/internal/services/analyticsService"
	skinservice "github.com/kedr891/cs-parser/internal/services/skinService"
	"github.com/kedr891/cs-parser/internal/storage/pgstorage"
)

func InitLogger(cfg *config.Config) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func InitSkinService(storage *pgstorage.Storage, cache *skinservice.SkinsCache, log *slog.Logger) *skinservice.Service {
	return skinservice.New(storage, cache, log)
}

func InitAnalyticsService(
	storage *pgstorage.Storage,
	log *slog.Logger,
) *analyticsservice.Service {
	return analyticsservice.New(storage, nil, nil, log)
}
