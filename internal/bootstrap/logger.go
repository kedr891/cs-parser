package bootstrap

import (
	"github.com/kedr891/cs-parser/config"
	"github.com/kedr891/cs-parser/pkg/logger"
)

func InitLogger(cfg *config.Config) *logger.Logger {
	return logger.New(cfg.Log.Level)
}
