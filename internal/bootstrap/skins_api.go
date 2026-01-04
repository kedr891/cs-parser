package bootstrap

import (
	skins_service_api "github.com/kedr891/cs-parser/internal/api/skins_service_api"
	analyticsservice "github.com/kedr891/cs-parser/internal/services/analyticsService"
	skinservice "github.com/kedr891/cs-parser/internal/services/skinService"
)

func InitSkinsServiceAPI(skinService *skinservice.Service, analyticsService *analyticsservice.Service) *skins_service_api.SkinsServiceAPI {
	return skins_service_api.NewSkinsServiceAPI(skinService, analyticsService)
}
