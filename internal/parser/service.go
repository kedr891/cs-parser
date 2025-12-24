package parser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kedr891/cs-parser/internal/domain"
	"github.com/kedr891/cs-parser/internal/entity"
	"golang.org/x/sync/errgroup"
)

const (
	_defaultWorkers = 10
	_rateLimitKey   = "rate_limit:parser:steam_market"
	_rateLimitTTL   = time.Minute
)

type Service struct {
	repo              domain.SkinRepository
	marketClient      domain.MarketClient
	priceProducer     domain.MessageProducer
	discoveryProducer domain.MessageProducer
	cache             domain.CacheStorage
	log               domain.Logger
}

func NewService(
	repo domain.SkinRepository,
	marketClient domain.MarketClient,
	priceProducer domain.MessageProducer,
	discoveryProducer domain.MessageProducer,
	cache domain.CacheStorage,
	log domain.Logger,
) *Service {
	return &Service{
		repo:              repo,
		marketClient:      marketClient,
		priceProducer:     priceProducer,
		discoveryProducer: discoveryProducer,
		cache:             cache,
		log:               log,
	}
}

func (s *Service) ParseAllSkins(ctx context.Context) error {
	s.log.Info("Starting full skin parsing cycle")
	startTime := time.Now()

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

func (s *Service) parseSkinsWithWorkers(ctx context.Context, skins []entity.Skin) (success, errors int) {
	var (
		successCount int
		errorCount   int
		mu           sync.Mutex
	)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(_defaultWorkers)

	for _, skin := range skins {
		skin := skin

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
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		s.log.Error("Worker pool error", "error", err)
	}

	return successCount, errorCount
}

func (s *Service) parseSingleSkin(ctx context.Context, skin *entity.Skin) error {
	if err := s.checkRateLimit(ctx); err != nil {
		return fmt.Errorf("rate limit exceeded: %w", err)
	}

	priceData, err := s.marketClient.GetItemPrice(ctx, skin.MarketHashName)
	if err != nil {
		return fmt.Errorf("get steam price: %w", err)
	}

	oldPrice := skin.CurrentPrice

	if err := s.repo.UpdateSkinPrice(ctx, skin.ID, priceData.Price, priceData.Volume); err != nil {
		return fmt.Errorf("update skin price: %w", err)
	}

	cacheKey := fmt.Sprintf("skin:price:%s", skin.ID.String())
	if err := s.cache.Delete(ctx, cacheKey); err != nil {
		s.log.Warn("Failed to invalidate cache", "key", cacheKey, "error", err)
	}

	priceEvent := entity.NewPriceUpdateEvent(
		skin.ID,
		skin.Slug,
		skin.MarketHashName,
		string(entity.SourceSteamMarket),
		oldPrice,
		priceData.Price,
		priceData.Volume,
	)

	if err := s.priceProducer.WriteMessage(ctx, skin.ID.String(), priceEvent); err != nil {
		s.log.Error("Failed to send price event to Kafka",
			"skin_id", skin.ID,
			"error", err,
		)
	}

	s.log.Debug("Skin parsed successfully",
		"skin_id", skin.ID,
		"old_price", oldPrice,
		"new_price", priceData.Price,
		"change", priceEvent.PriceChange,
	)

	return nil
}

func (s *Service) DiscoverNewSkins(ctx context.Context, searchQuery string) error {
	s.log.Info("Starting skin discovery", "query", searchQuery)

	items, err := s.marketClient.SearchItems(ctx, searchQuery)
	if err != nil {
		return fmt.Errorf("search items: %w", err)
	}

	s.log.Info("Found items", "count", len(items))

	var discoveredCount int
	for _, item := range items {
		exists, err := s.repo.SkinExists(ctx, item.MarketHashName)
		if err != nil {
			s.log.Error("Failed to check skin existence", "error", err)
			continue
		}

		if exists {
			continue
		}

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

		if err := s.repo.CreateSkin(ctx, skin); err != nil {
			s.log.Error("Failed to create skin", "error", err)
			continue
		}

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

func (s *Service) ParseSpecificSkins(ctx context.Context, slugs []string) error {
	s.log.Info("Starting specific skins parsing", "count", len(slugs))

	var successCount, errorCount int
	for _, slug := range slugs {
		skin, err := s.repo.GetSkinBySlug(ctx, slug)
		if err != nil {
			s.log.Error("Failed to get skin", "slug", slug, "error", err)
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

func (s *Service) checkRateLimit(ctx context.Context) error {
	count, err := s.cache.IncrementRateLimit(ctx, _rateLimitKey, _rateLimitTTL)
	if err != nil {
		return fmt.Errorf("increment rate limit: %w", err)
	}

	if count > 60 {
		return fmt.Errorf("rate limit exceeded: %d requests in last minute", count)
	}

	return nil
}

func (s *Service) GetStats(ctx context.Context) (*ParserStats, error) {
	totalSkins, err := s.repo.GetSkinsCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("get skins count: %w", err)
	}

	rateLimit, err := s.cache.GetRateLimit(ctx, _rateLimitKey)
	if err != nil {
		rateLimit = 0
	}

	return &ParserStats{
		TotalSkins:      totalSkins,
		RequestsLastMin: int(rateLimit),
		LastParseTime:   time.Now(),
	}, nil
}

type ParserStats struct {
	TotalSkins      int       `json:"total_skins"`
	RequestsLastMin int       `json:"requests_last_min"`
	LastParseTime   time.Time `json:"last_parse_time"`
}
