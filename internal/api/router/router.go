package router

import (
	"github.com/gin-gonic/gin"
	_ "github.com/kedr891/cs-parser/docs"
	"github.com/kedr891/cs-parser/internal/api/handler"
	"github.com/kedr891/cs-parser/internal/api/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes(
	r *gin.Engine,
	skinHandler *handler.SkinHandler,
	userHandler *handler.UserHandler,
	analyticsHandler *handler.AnalyticsHandler,
	authMiddleware *middleware.AuthMiddleware,
) {
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		skins := v1.Group("/skins")
		{
			skins.GET("", skinHandler.GetSkins)
			skins.GET("/search", skinHandler.SearchSkins)
			skins.GET("/popular", skinHandler.GetPopularSkins)
			skins.GET("/:slug", skinHandler.GetSkinBySlug)
			skins.GET("/chart/:slug", skinHandler.GetPriceChart)
		}

		analytics := v1.Group("/analytics")
		{
			analytics.GET("/trending", analyticsHandler.GetTrending)
			analytics.GET("/market-overview", analyticsHandler.GetMarketOverview)
			analytics.GET("/popular-searches", analyticsHandler.GetPopularSearches)
		}

		auth := v1.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
		}

		users := v1.Group("/users")
		users.Use(authMiddleware.RequireAuth())
		{
			users.GET("/me", userHandler.GetProfile)
			users.GET("/me/watchlist", userHandler.GetWatchlist)
			users.POST("/me/watchlist/:skin_id", userHandler.AddToWatchlist)
			users.DELETE("/me/watchlist/:skin_id", userHandler.RemoveFromWatchlist)
			users.GET("/me/notifications", userHandler.GetNotifications)
			users.POST("/me/notifications/read", userHandler.MarkNotificationsRead)
		}

		admin := v1.Group("/admin")
		admin.Use(authMiddleware.RequireAuth())
		admin.Use(authMiddleware.RequireAdmin())
		{
			// admin.GET("/stats", adminHandler.GetStats)
			// admin.POST("/parser/trigger", adminHandler.TriggerParser)
		}
	}
}
