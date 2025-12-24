package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kedr891/cs-parser/internal/api/service"
	"github.com/kedr891/cs-parser/internal/domain"
	"github.com/kedr891/cs-parser/internal/entity"
)

type SkinHandler struct {
	service *service.SkinService
	log     domain.Logger
}

func NewSkinHandler(service *service.SkinService, log domain.Logger) *SkinHandler {
	return &SkinHandler{
		service: service,
		log:     log,
	}
}

// GetSkins - получить список скинов с фильтрацией
// @Summary Get skins list
// @Tags skins
// @Param weapon query string false "Weapon filter"
// @Param quality query string false "Quality filter"
// @Param min_price query number false "Min price"
// @Param max_price query number false "Max price"
// @Param search query string false "Search query"
// @Param sort_by query string false "Sort by: price, volume, name, updated"
// @Param sort_order query string false "Sort order: asc, desc"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Success 200 {object} entity.SkinListResponse
// @Router /api/v1/skins [get]
func (h *SkinHandler) GetSkins(c *gin.Context) {
	filter := entity.NewSkinFilter()
	filter.Weapon = c.Query("weapon")
	filter.Quality = c.Query("quality")
	filter.Search = c.Query("search")
	filter.SortBy = c.DefaultQuery("sort_by", "updated")
	filter.SortOrder = c.DefaultQuery("sort_order", "desc")

	if minPrice := c.Query("min_price"); minPrice != "" {
		if price, err := strconv.ParseFloat(minPrice, 64); err == nil {
			filter.MinPrice = price
		}
	}
	if maxPrice := c.Query("max_price"); maxPrice != "" {
		if price, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			filter.MaxPrice = price
		}
	}

	page := 1
	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}

	pageSize := 50
	if ps := c.Query("page_size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 && val <= 100 {
			pageSize = val
		}
	}

	filter.Limit = pageSize
	filter.Offset = (page - 1) * pageSize

	response, err := h.service.GetSkins(c.Request.Context(), filter)
	if err != nil {
		h.log.Error("Failed to get skins", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get skins",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetSkinBySlug - получить скин по slug
// @Summary Get skin by slug
// @Tags skins
// @Param slug path string true "Skin slug (e.g., awp_acheron_ft)"
// @Param period query string false "Period: 24h, 7d, 30d, 90d, 1y, all"
// @Success 200 {object} entity.SkinDetailResponse
// @Router /api/v1/skins/{slug} [get]
func (h *SkinHandler) GetSkinBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Slug is required"})
		return
	}

	period := entity.Period7d
	if p := c.Query("period"); p != "" {
		period = entity.PriceStatsPeriod(p)
	}

	response, err := h.service.GetSkinBySlug(c.Request.Context(), slug, period)
	if err != nil {
		h.log.Error("Failed to get skin by slug", "error", err, "slug", slug)
		c.JSON(http.StatusNotFound, gin.H{"error": "Skin not found"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetPriceChart - получить данные для графика цен
// @Summary Get price chart
// @Tags skins
// @Param slug path string true "Skin slug"
// @Param period query string false "Period: 24h, 7d, 30d, 90d, 1y, all"
// @Success 200 {object} entity.PriceChartResponse
// @Router /api/v1/skins/chart/{slug} [get]
func (h *SkinHandler) GetPriceChart(c *gin.Context) {
	slug := c.Param("slug")

	period := entity.Period7d
	if p := c.Query("period"); p != "" {
		period = entity.PriceStatsPeriod(p)
	}

	chartData, err := h.service.GetPriceChart(
		c.Request.Context(),
		slug,
		period,
	)
	if err != nil {
		h.log.Error("Failed to get chart data", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chartData)
}

// SearchSkins - поиск скинов
// @Summary Search skins
// @Tags skins
// @Param q query string true "Search query"
// @Param limit query int false "Limit"
// @Success 200 {array} entity.Skin
// @Router /api/v1/skins/search [get]
func (h *SkinHandler) SearchSkins(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}

	// Вызов сервиса
	skins, err := h.service.SearchSkins(c.Request.Context(), query, limit)
	if err != nil {
		h.log.Error("Failed to search skins", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search skins"})
		return
	}

	c.JSON(http.StatusOK, skins)
}

// GetPopularSkins - получить популярные скины
// @Summary Get popular skins
// @Tags skins
// @Param limit query int false "Limit"
// @Success 200 {array} entity.Skin
// @Router /api/v1/skins/popular [get]
func (h *SkinHandler) GetPopularSkins(c *gin.Context) {
	limit := 10
	if l := c.Query("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 50 {
			limit = val
		}
	}

	skins, err := h.service.GetPopularSkins(c.Request.Context(), limit)
	if err != nil {
		h.log.Error("Failed to get popular skins", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get popular skins"})
		return
	}

	c.JSON(http.StatusOK, skins)
}
