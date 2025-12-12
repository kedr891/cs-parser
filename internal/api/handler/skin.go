package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kedr891/cs-parser/internal/entity"
	"github.com/kedr891/cs-parser/pkg/logger"
	"github.com/kedr891/cs-parser/pkg/postgres"
	"github.com/kedr891/cs-parser/pkg/redis"
)

// SkinHandler - обработчик для скинов
type SkinHandler struct {
	pg    *postgres.Postgres
	redis *redis.Redis
	log   *logger.Logger
}

// NewSkinHandler - создать обработчик скинов
func NewSkinHandler(pg *postgres.Postgres, redis *redis.Redis, log *logger.Logger) *SkinHandler {
	return &SkinHandler{
		pg:    pg,
		redis: redis,
		log:   log,
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
	// Парсинг параметров фильтра
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

	// Проверить кэш
	cacheKey := h.generateCacheKey(filter)
	if cached, err := h.redis.GetCache(c.Request.Context(), cacheKey); err == nil {
		var response entity.SkinListResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			h.log.Debug("Skins loaded from cache")
			c.JSON(http.StatusOK, response)
			return
		}
	}

	// Получить скины из БД
	skins, total, err := h.getSkinsFromDB(c.Request.Context(), filter)
	if err != nil {
		h.log.Error("Failed to get skins", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get skins"})
		return
	}

	totalPages := (total + pageSize - 1) / pageSize

	response := entity.SkinListResponse{
		Skins:      skins,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	// Сохранить в кэш
	if data, err := json.Marshal(response); err == nil {
		_ = h.redis.SetCache(c.Request.Context(), cacheKey, string(data), 2*time.Minute)
	}

	c.JSON(http.StatusOK, response)
}

// GetSkinByID - получить скин по ID
// @Summary Get skin by ID
// @Tags skins
// @Param id path string true "Skin ID"
// @Success 200 {object} entity.SkinDetailResponse
// @Router /api/v1/skins/{id} [get]
func (h *SkinHandler) GetSkinByID(c *gin.Context) {
	idStr := c.Param("id")
	skinID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid skin ID"})
		return
	}

	// Проверить кэш
	cacheKey := fmt.Sprintf("skin:detail:%s", skinID.String())
	if cached, err := h.redis.GetCache(c.Request.Context(), cacheKey); err == nil {
		var response entity.SkinDetailResponse
		if err := json.Unmarshal([]byte(cached), &response); err == nil {
			// Инкремент просмотров
			_ = h.redis.IncrementCounter(c.Request.Context(), fmt.Sprintf("analytics:views:%s", skinID.String()))
			c.JSON(http.StatusOK, response)
			return
		}
	}

	// Получить скин
	skin, err := h.getSkinByID(c.Request.Context(), skinID)
	if err != nil {
		h.log.Error("Failed to get skin", "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Skin not found"})
		return
	}

	// Получить историю цен
	period := entity.Period7d
	if p := c.Query("period"); p != "" {
		period = entity.PriceStatsPeriod(p)
	}

	priceHistory, err := h.getPriceHistory(c.Request.Context(), skinID, period)
	if err != nil {
		h.log.Warn("Failed to get price history", "error", err)
		priceHistory = []entity.PriceHistory{}
	}

	// Получить статистику
	stats, err := h.getSkinStatistics(c.Request.Context(), skinID)
	if err != nil {
		h.log.Warn("Failed to get statistics", "error", err)
		stats = entity.SkinStatistics{}
	}

	response := entity.SkinDetailResponse{
		Skin:         *skin,
		PriceHistory: priceHistory,
		Statistics:   stats,
	}

	// Сохранить в кэш
	if data, err := json.Marshal(response); err == nil {
		_ = h.redis.SetCache(c.Request.Context(), cacheKey, string(data), 5*time.Minute)
	}

	// Инкремент просмотров
	_ = h.redis.IncrementCounter(c.Request.Context(), fmt.Sprintf("analytics:views:%s", skinID.String()))

	c.JSON(http.StatusOK, response)
}

// GetPriceChart - получить данные для графика цен
// @Summary Get price chart
// @Tags skins
// @Param id path string true "Skin ID"
// @Param period query string false "Period: 24h, 7d, 30d, 90d, 1y, all"
// @Success 200 {object} entity.PriceChartResponse
// @Router /api/v1/skins/{id}/chart [get]
func (h *SkinHandler) GetPriceChart(c *gin.Context) {
	idStr := c.Param("id")
	skinID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid skin ID"})
		return
	}

	period := entity.Period7d
	if p := c.Query("period"); p != "" {
		period = entity.PriceStatsPeriod(p)
	}

	chartData, err := h.getPriceChartData(c.Request.Context(), skinID, period)
	if err != nil {
		h.log.Error("Failed to get chart data", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chart data"})
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

	skins, err := h.searchSkins(c.Request.Context(), query, limit)
	if err != nil {
		h.log.Error("Failed to search skins", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search skins"})
		return
	}

	// Записать популярный запрос
	_ = h.redis.Client.ZIncrBy(c.Request.Context(), "analytics:popular:searches", 1, query)

	c.JSON(http.StatusOK, skins)
}

// Helper methods

func (h *SkinHandler) generateCacheKey(filter *entity.SkinFilter) string {
	return fmt.Sprintf("skins:list:%s:%s:%.2f-%.2f:%s:%s:%d:%d",
		filter.Weapon,
		filter.Quality,
		filter.MinPrice,
		filter.MaxPrice,
		filter.SortBy,
		filter.SortOrder,
		filter.Limit,
		filter.Offset,
	)
}

func (h *SkinHandler) getSkinsFromDB(ctx context.Context, filter *entity.SkinFilter) ([]entity.Skin, int, error) {
	// Simplified query - в реальном проекте используй query builder (squirrel)
	query := `
		SELECT 
			id, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		WHERE 1=1
	`

	countQuery := `SELECT COUNT(*) FROM skins WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	// Добавить фильтры
	if filter.Weapon != "" {
		query += fmt.Sprintf(" AND weapon = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND weapon = $%d", argIndex)
		args = append(args, filter.Weapon)
		argIndex++
	}

	if filter.Quality != "" {
		query += fmt.Sprintf(" AND quality = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND quality = $%d", argIndex)
		args = append(args, filter.Quality)
		argIndex++
	}

	if filter.Search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR market_hash_name ILIKE $%d)", argIndex, argIndex)
		countQuery += fmt.Sprintf(" AND (name ILIKE $%d OR market_hash_name ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	// Получить total count
	var total int
	if err := h.pg.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count skins: %w", err)
	}

	// Добавить сортировку и пагинацию
	query += fmt.Sprintf(" ORDER BY %s %s LIMIT $%d OFFSET $%d", filter.SortBy, filter.SortOrder, argIndex, argIndex+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := h.pg.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query skins: %w", err)
	}
	defer rows.Close()

	var skins []entity.Skin
	for rows.Next() {
		var skin entity.Skin
		err := rows.Scan(
			&skin.ID, &skin.MarketHashName, &skin.Name, &skin.Weapon, &skin.Quality, &skin.Rarity,
			&skin.CurrentPrice, &skin.Currency, &skin.ImageURL, &skin.Volume24h,
			&skin.PriceChange24h, &skin.PriceChange7d,
			&skin.LowestPrice, &skin.HighestPrice,
			&skin.LastUpdated, &skin.CreatedAt, &skin.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan skin: %w", err)
		}
		skins = append(skins, skin)
	}

	return skins, total, nil
}

func (h *SkinHandler) getSkinByID(ctx context.Context, id uuid.UUID) (*entity.Skin, error) {
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

	if err != nil {
		return nil, err
	}

	return &skin, nil
}

func (h *SkinHandler) getPriceHistory(ctx context.Context, skinID uuid.UUID, period entity.PriceStatsPeriod) ([]entity.PriceHistory, error) {
	since := time.Now().Add(-period.GetDuration())

	query := `
		SELECT id, skin_id, price, currency, source, volume, recorded_at
		FROM price_history
		WHERE skin_id = $1 AND recorded_at >= $2
		ORDER BY recorded_at ASC
	`

	rows, err := h.pg.Pool.Query(ctx, query, skinID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []entity.PriceHistory
	for rows.Next() {
		var h entity.PriceHistory
		if err := rows.Scan(&h.ID, &h.SkinID, &h.Price, &h.Currency, &h.Source, &h.Volume, &h.RecordedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, nil
}

func (h *SkinHandler) getSkinStatistics(ctx context.Context, skinID uuid.UUID) (entity.SkinStatistics, error) {
	// Получить view count из Redis
	viewCount, _ := h.redis.GetCounter(ctx, fmt.Sprintf("analytics:views:%s", skinID.String()))

	// Рассчитать статистику из БД
	query := `
		SELECT 
			COALESCE(AVG(price), 0) as avg_price_7d,
			COALESCE(SUM(volume), 0) as total_volume_7d
		FROM price_history
		WHERE skin_id = $1 AND recorded_at >= NOW() - INTERVAL '7 days'
	`

	var stats entity.SkinStatistics
	err := h.pg.Pool.QueryRow(ctx, query, skinID).Scan(&stats.AvgPrice7d, &stats.TotalVolume7d)
	if err != nil {
		return stats, err
	}

	stats.ViewCount = viewCount
	return stats, nil
}

func (h *SkinHandler) getPriceChartData(ctx context.Context, skinID uuid.UUID, period entity.PriceStatsPeriod) (*entity.PriceChartResponse, error) {
	history, err := h.getPriceHistory(ctx, skinID, period)
	if err != nil {
		return nil, err
	}

	if len(history) == 0 {
		return &entity.PriceChartResponse{
			SkinID:     skinID,
			Period:     string(period),
			DataPoints: []entity.PriceChartData{},
		}, nil
	}

	dataPoints := make([]entity.PriceChartData, len(history))
	var minPrice, maxPrice, sumPrice float64
	var totalVolume int

	minPrice = history[0].Price
	maxPrice = history[0].Price

	for i, h := range history {
		dataPoints[i] = entity.PriceChartData{
			Timestamp: h.RecordedAt,
			Price:     h.Price,
			Volume:    h.Volume,
		}

		if h.Price < minPrice {
			minPrice = h.Price
		}
		if h.Price > maxPrice {
			maxPrice = h.Price
		}
		sumPrice += h.Price
		totalVolume += h.Volume
	}

	return &entity.PriceChartResponse{
		SkinID:      skinID,
		Period:      string(period),
		DataPoints:  dataPoints,
		MinPrice:    minPrice,
		MaxPrice:    maxPrice,
		AvgPrice:    sumPrice / float64(len(history)),
		TotalVolume: totalVolume,
	}, nil
}

func (h *SkinHandler) searchSkins(ctx context.Context, query string, limit int) ([]entity.Skin, error) {
	sql := `
		SELECT 
			id, market_hash_name, name, weapon, quality, rarity,
			current_price, currency, image_url, volume_24h,
			price_change_24h, price_change_7d,
			lowest_price, highest_price,
			last_updated, created_at, updated_at
		FROM skins
		WHERE name ILIKE $1 OR market_hash_name ILIKE $1
		ORDER BY volume_24h DESC
		LIMIT $2
	`

	rows, err := h.pg.Pool.Query(ctx, sql, "%"+query+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skins []entity.Skin
	for rows.Next() {
		var skin entity.Skin
		err := rows.Scan(
			&skin.ID, &skin.MarketHashName, &skin.Name, &skin.Weapon, &skin.Quality, &skin.Rarity,
			&skin.CurrentPrice, &skin.Currency, &skin.ImageURL, &skin.Volume24h,
			&skin.PriceChange24h, &skin.PriceChange7d,
			&skin.LowestPrice, &skin.HighestPrice,
			&skin.LastUpdated, &skin.CreatedAt, &skin.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		skins = append(skins, skin)
	}

	return skins, nil
}
