package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/cs-parser/internal/entity"
	"github.com/cs-parser/pkg/logger"
	"github.com/cs-parser/pkg/postgres"
	"github.com/cs-parser/pkg/redis"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AnalyticsHandler - обработчик для аналитики
type AnalyticsHandler struct {
	pg    *postgres.Postgres
	redis *redis.Redis
	log   *logger.Logger
}

// NewAnalyticsHandler - создать обработчик аналитики
func NewAnalyticsHandler(pg *postgres.Postgres, redis *redis.Redis, log *logger.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		pg:    pg,
		redis: redis,
		log:   log,
	}
}

// GetTrending - получить трендовые скины
// @Summary Get trending skins
// @Tags analytics
// @Param period query string false "Period: 24h or 7d"
// @Param limit query int false "Limit"
// @Success 200 {array} entity.TrendingSkin
// @Router /api/v1/analytics/trending [get]
func (h *AnalyticsHandler) GetTrending(c *gin.Context) {
	period := c.DefaultQuery("period", "24h")
	limit := 20
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 50 {
			limit = val
		}
	}

	key := "analytics:trending:24h"
	if period == "7d" {
		key = "analytics:trending:7d"
	}

	// Получить skin IDs из Redis
	skinIDs, err := h.redis.ZRevRange(c.Request.Context(), key, 0, int64(limit-1))
	if err != nil {
		h.log.Error("Failed to get trending from Redis", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get trending"})
		return
	}

	if len(skinIDs) == 0 {
		c.JSON(http.StatusOK, []entity.TrendingSkin{})
		return
	}

	// Получить детали скинов из БД
	trending := make([]entity.TrendingSkin, 0, len(skinIDs))
	for i, idStr := range skinIDs {
		skinID, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}

		skin, err := h.getSkinByID(c.Request.Context(), skinID)
		if err != nil {
			continue
		}

		// Получить score (price change) из Redis
		score, _ := h.redis.Client.ZScore(c.Request.Context(), key, idStr).Result()

		trending = append(trending, entity.TrendingSkin{
			Skin:            *skin,
			PriceChangeRate: score,
			Rank:            i + 1,
		})
	}

	c.JSON(http.StatusOK, trending)
}

// GetMarketOverview - получить обзор рынка
// @Summary Get market overview
// @Tags analytics
// @Success 200 {object} entity.MarketOverview
// @Router /api/v1/analytics/market-overview [get]
func (h *AnalyticsHandler) GetMarketOverview(c *gin.Context) {
	// Проверить кэш
	cacheKey := "analytics:market:overview"
	if cached, err := h.redis.GetCache(c.Request.Context(), cacheKey); err == nil {
		var overview entity.MarketOverview
		if err := json.Unmarshal([]byte(cached), &overview); err == nil {
			c.JSON(http.StatusOK, overview)
			return
		}
	}

	// Если кэш пуст, вернуть пустой объект или рассчитать заново
	// В реальном проекте здесь было бы обновление из БД
	c.JSON(http.StatusOK, entity.MarketOverview{
		TotalSkins:      0,
		AvgPrice:        0,
		TotalVolume24h:  0,
		TopGainers:      []entity.Skin{},
		TopLosers:       []entity.Skin{},
		MostPopular:     []entity.Skin{},
		RecentlyUpdated: []entity.Skin{},
	})
}

// GetPopularSearches - получить популярные поисковые запросы
// @Summary Get popular searches
// @Tags analytics
// @Param limit query int false "Limit"
// @Success 200 {array} string
// @Router /api/v1/analytics/popular-searches [get]
func (h *AnalyticsHandler) GetPopularSearches(c *gin.Context) {
	limit := 10
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 50 {
			limit = val
		}
	}

	searches, err := h.redis.ZRevRange(c.Request.Context(), "analytics:popular:searches", 0, int64(limit-1))
	if err != nil {
		h.log.Error("Failed to get popular searches", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get popular searches"})
		return
	}

	c.JSON(http.StatusOK, searches)
}

// Helper methods

func (h *AnalyticsHandler) getSkinByID(ctx context.Context, id uuid.UUID) (*entity.Skin, error) {
	query := `
		SELECT 
			id, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		WHERE id = $1
	`

	var skin entity.Skin
	err := h.pg.Pool.QueryRow(ctx, query, id).Scan(
		&skin.ID, &skin.MarketHashName, &skin.Name, &skin.Weapon, &skin.Quality, &skin.Rarity,
		&skin.CurrentPrice, &skin.Currency, &skin.ImageURL, &skin.Volume24h,
		&skin.PriceChange24h, &skin.PriceChange7d,
		&skin.LowestPrice, &skin.HighestPrice,
		&skin.LastUpdated, &skin.CreatedAt, &skin.UpdatedAt,
	)

	return &skin, err
}
