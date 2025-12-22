package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kedr891/cs-parser/internal/api/service"
	"github.com/kedr891/cs-parser/internal/domain"
)

// AnalyticsHandler - обработчик для аналитики
type AnalyticsHandler struct {
	service *service.AnalyticsService
	log     domain.Logger
}

// NewAnalyticsHandler - создать обработчик аналитики
func NewAnalyticsHandler(service *service.AnalyticsService, log domain.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		service: service,
		log:     log,
	}
}

// GetTrending - получить трендовые скины
// @Summary Get trending skins
// @Tags analytics
// @Param period query string false "Period: 24h or 7d"
// @Param limit query int false "Limit"
// @Success 200 {array} entity.Skin
// @Router /api/v1/analytics/trending [get]
func (h *AnalyticsHandler) GetTrending(c *gin.Context) {
	period := c.DefaultQuery("period", "24h")
	limit := 20
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 50 {
			limit = val
		}
	}

	skins, err := h.service.GetTrendingSkins(c.Request.Context(), period, limit)
	if err != nil {
		h.log.Error("Failed to get trending skins", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get trending"})
		return
	}

	c.JSON(http.StatusOK, skins)
}

// GetMarketOverview - получить обзор рынка
// @Summary Get market overview
// @Tags analytics
// @Success 200 {object} entity.MarketOverview
// @Router /api/v1/analytics/market-overview [get]
func (h *AnalyticsHandler) GetMarketOverview(c *gin.Context) {
	overview, err := h.service.GetMarketOverview(c.Request.Context())
	if err != nil {
		h.log.Error("Failed to get market overview", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get market overview"})
		return
	}

	c.JSON(http.StatusOK, overview)
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

	searches, err := h.service.GetPopularSearches(c.Request.Context(), limit)
	if err != nil {
		h.log.Error("Failed to get popular searches", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get popular searches"})
		return
	}

	c.JSON(http.StatusOK, searches)
}
