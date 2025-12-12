package parser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kedr891/cs-parser/internal/entity"
	"github.com/kedr891/cs-parser/pkg/kafka"
	"github.com/kedr891/cs-parser/pkg/logger"
	"github.com/kedr891/cs-parser/pkg/redis"
	"golang.org/x/sync/errgroup"
)

const (
	_defaultBatchSize = 50
	_defaultWorkers   = 10
	_rateLimitKey     = "rate_limit:parser:steam_market"
	_rateLimitTTL     = time.Minute
)

// Service - сервис парсинга цен
type Service struct {
	repo              Repository
	steamClient       *SteamClient
	priceProducer     *kafka.Producer
	discoveryProducer *kafka.Producer
	redis             *redis.Redis
	log               *logger.Logger
}

// NewService - создать сервис парсинга
func NewService(
	repo Repository,
	steamClient *SteamClient,
	priceProducer *kafka.Producer,
	discoveryProducer *kafka.Producer,
	redis *redis.Redis,
	log *logger.Logger,
) *Service {
	return &Service{
		repo:              repo,
		steamClient:       steamClient,
		priceProducer:     priceProducer,
		discoveryProducer: discoveryProducer,
		redis:             redis,
		log:               log,
	}
}

// ParseAllSkins - парсинг всех скинов из БД
func (s *Service) ParseAllSkins(ctx context.Context) error {
	s.log.Info("Starting full skin parsing cycle")
	startTime := time.Now()

	// Получить все скины из БД
	skins, err := s.repo.GetAllSkins(ctx)
	if err != nil {
		return fmt.Errorf("get all skins: %w", err)
	}

	totalSkins := len(skins)
	s.log.Info("Found skins to parse", "total", totalSkins)

	if totalSkins == 0 {
		s.log.Warn("No skins found in database")
		return nil
	}

	// Парсинг с использованием worker pool
	successCount, errorCount := s.parseSkinsWithWorkers(ctx, skins)

	duration := time.Since(startTime)
	s.log.Info("Parsing cycle completed",
		"total", totalSkins,
		"success", successCount,
		"errors", errorCount,
		"duration", duration.String(),
	)

	return nil
}

// parseSkinsWithWorkers - параллельный парсинг с worker pool
func (s *Service) parseSkinsWithWorkers(ctx context.Context, skins []entity.Skin) (success, errors int) {
	var (
		successCount int
		errorCount   int
		mu           sync.Mutex
	)

	// Worker pool
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(_defaultWorkers)

	for _, skin := range skins {
		skin := skin // capture loop variable

		g.Go(func() error {
			if err := s.parseSingleSkin(ctx, &skin); err != nil {
				s.log.Error("Failed to parse skin",
					"skin_id", skin.ID,
					"market_hash_name", skin.MarketHashName,
					"error", err,
				)
				mu.Lock()
				errorCount++
				mu.Unlock()
			} else {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
			return nil // продолжаем даже при ошибках
		})
	}

	// Ждём завершения всех воркеров
	if err := g.Wait(); err != nil {
		s.log.Error("Worker pool error", "error", err)
	}

	return successCount, errorCount
}

// parseSingleSkin - парсинг одного скина
func (s *Service) parseSingleSkin(ctx context.Context, skin *entity.Skin) error {
	// Проверка rate limit
	if err := s.checkRateLimit(ctx); err != nil {
		return fmt.Errorf("rate limit exceeded: %w", err)
	}

	// Получить цену из Steam Market
	priceData, err := s.steamClient.GetItemPrice(ctx, skin.MarketHashName)
	if err != nil {
		return fmt.Errorf("get steam price: %w", err)
	}

	// Сохранить старую цену для сравнения
	oldPrice := skin.CurrentPrice

	// Обновить скин в БД
	if err := s.repo.UpdateSkinPrice(ctx, skin.ID, priceData.Price, priceData.Volume); err != nil {
		return fmt.Errorf("update skin price: %w", err)
	}

	// Инвалидировать кэш
	cacheKey := fmt.Sprintf("skin:price:%s", skin.ID.String())
	if err := s.redis.DeleteCache(ctx, cacheKey); err != nil {
		s.log.Warn("Failed to invalidate cache", "key", cacheKey, "error", err)
	}

	// Создать событие обновления цены
	priceEvent := entity.NewPriceUpdateEvent(
		skin.ID,
		skin.MarketHashName,
		string(entity.SourceSteamMarket),
		oldPrice,
		priceData.Price,
		priceData.Volume,
	)

	// Отправить в Kafka
	if err := s.priceProducer.WriteMessage(ctx, skin.ID.String(), priceEvent); err != nil {
		s.log.Error("Failed to send price event to Kafka",
			"skin_id", skin.ID,
			"error", err,
		)
		// Не возвращаем ошибку, так как цена уже обновлена в БД
	}

	s.log.Debug("Skin parsed successfully",
		"skin_id", skin.ID,
		"old_price", oldPrice,
		"new_price", priceData.Price,
		"change", priceEvent.PriceChange,
	)

	return nil
}

// DiscoverNewSkins - поиск новых скинов
func (s *Service) DiscoverNewSkins(ctx context.Context, searchQuery string) error {
	s.log.Info("Starting skin discovery", "query", searchQuery)

	// Поиск скинов через Steam API
	items, err := s.steamClient.SearchItems(ctx, searchQuery)
	if err != nil {
		return fmt.Errorf("search items: %w", err)
	}

	s.log.Info("Found items", "count", len(items))

	var discoveredCount int
	for _, item := range items {
		// Проверить, существует ли уже в БД
		exists, err := s.repo.SkinExists(ctx, item.MarketHashName)
		if err != nil {
			s.log.Error("Failed to check skin existence", "error", err)
			continue
		}

		if exists {
			continue
		}

		// Создать новый скин
		skin := entity.NewSkin(
			item.MarketHashName,
			item.Name,
			item.Weapon,
			item.Quality,
		)
		skin.ImageURL = item.ImageURL
		skin.Rarity = item.Rarity
		skin.CurrentPrice = item.Price
		skin.Currency = "USD"

		// Сохранить в БД
		if err := s.repo.CreateSkin(ctx, skin); err != nil {
			s.log.Error("Failed to create skin", "error", err)
			continue
		}

		// Создать событие обнаружения
		discoveryEvent := entity.NewSkinDiscoveredEvent(
			skin.MarketHashName,
			skin.Name,
			skin.Weapon,
			skin.Quality,
			skin.Rarity,
			skin.CurrentPrice,
			string(entity.SourceSteamMarket),
			skin.ImageURL,
		)

		// Отправить в Kafka
		if err := s.discoveryProducer.WriteMessage(ctx, skin.ID.String(), discoveryEvent); err != nil {
			s.log.Error("Failed to send discovery event", "error", err)
		}

		discoveredCount++
		s.log.Info("New skin discovered",
			"skin_id", skin.ID,
			"name", skin.MarketHashName,
			"price", skin.CurrentPrice,
		)
	}

	s.log.Info("Skin discovery completed", "discovered", discoveredCount)
	return nil
}

// ParseSpecificSkins - парсинг конкретных скинов по ID
func (s *Service) ParseSpecificSkins(ctx context.Context, skinIDs []uuid.UUID) error {
	s.log.Info("Starting specific skins parsing", "count", len(skinIDs))

	var successCount, errorCount int
	for _, skinID := range skinIDs {
		skin, err := s.repo.GetSkinByID(ctx, skinID)
		if err != nil {
			s.log.Error("Failed to get skin", "skin_id", skinID, "error", err)
			errorCount++
			continue
		}

		if err := s.parseSingleSkin(ctx, skin); err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	s.log.Info("Specific skins parsing completed",
		"success", successCount,
		"errors", errorCount,
	)

	return nil
}

// checkRateLimit - проверка rate limit
func (s *Service) checkRateLimit(ctx context.Context) error {
	count, err := s.redis.IncrementRateLimit(ctx, _rateLimitKey, _rateLimitTTL)
	if err != nil {
		return fmt.Errorf("increment rate limit: %w", err)
	}

	// Максимум 60 запросов в минуту
	if count > 60 {
		return fmt.Errorf("rate limit exceeded: %d requests in last minute", count)
	}

	return nil
}

// GetStats - получить статистику парсера
func (s *Service) GetStats(ctx context.Context) (*ParserStats, error) {
	totalSkins, err := s.repo.GetSkinsCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("get skins count: %w", err)
	}

	// Получить rate limit
	rateLimit, err := s.redis.GetRateLimit(ctx, _rateLimitKey)
	if err != nil {
		rateLimit = 0
	}

	return &ParserStats{
		TotalSkins:      totalSkins,
		RequestsLastMin: int(rateLimit),
		LastParseTime:   time.Now(), // TODO: сохранять в Redis
	}, nil
}

// ParserStats - статистика парсера
type ParserStats struct {
	TotalSkins      int       `json:"total_skins"`
	RequestsLastMin int       `json:"requests_last_min"`
	LastParseTime   time.Time `json:"last_parse_time"`
}
