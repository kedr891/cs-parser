package skinservice

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kedr891/cs-parser/internal/models"
	"github.com/kedr891/cs-parser/internal/services/skinService/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type SkinServiceSuite struct {
	suite.Suite
	ctx         context.Context
	service     *Service
	mockStorage *mocks.MockSkinStorage
	mockCache   *mocks.MockSkinCache
	log         *slog.Logger
}

func (suite *SkinServiceSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.mockStorage = mocks.NewMockSkinStorage(suite.T())
	suite.mockCache = mocks.NewMockSkinCache(suite.T())
	suite.log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	suite.service = New(suite.mockStorage, suite.mockCache, suite.log)
}

func TestSkinServiceSuite(t *testing.T) {
	suite.Run(t, new(SkinServiceSuite))
}

func (suite *SkinServiceSuite) TestGetSkins_Success_CacheMiss() {
	filter := &models.SkinFilter{
		Weapon:    "AK-47",
		Limit:     10,
		Offset:    0,
		SortBy:    "price",
		SortOrder: "DESC",
	}

	expectedSkins := []models.Skin{
		{
			ID:             uuid.New(),
			Slug:           "ak47-redline",
			MarketHashName: "AK-47 | Redline",
			Name:           "Redline",
			Weapon:         "AK-47",
			Quality:        "Field-Tested",
			Rarity:         "Classified",
			CurrentPrice:   15.50,
			Currency:       "USD",
			ImageURL:       "https://example.com/img.jpg",
			Volume24h:      100,
			PriceChange24h: 0.5,
			PriceChange7d:  1.2,
			LowestPrice:    14.00,
			HighestPrice:   17.00,
			LastUpdated:    time.Now(),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}

	suite.mockCache.On("GetSkinList", suite.ctx, suite.service.generateCacheKey(filter)).
		Return(nil, false)

	suite.mockStorage.On("GetSkins", suite.ctx, filter).
		Return(expectedSkins, 1, nil)

	suite.mockCache.On("SetSkinList", suite.ctx, suite.service.generateCacheKey(filter),
		&models.SkinListResponse{
			Skins:      expectedSkins,
			Total:      1,
			Page:       1,
			PageSize:   10,
			TotalPages: 1,
		}, 2*time.Minute).
		Return(nil)

	result, err := suite.service.GetSkins(suite.ctx, filter)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(1, result.Total)
	suite.Equal(1, len(result.Skins))
	suite.Equal("ak47-redline", result.Skins[0].Slug)
}

func (suite *SkinServiceSuite) TestGetSkins_Success_CacheHit() {
	filter := &models.SkinFilter{
		Weapon: "AWP",
		Limit:  5,
		Offset: 0,
	}

	cachedResponse := &models.SkinListResponse{
		Skins: []models.Skin{
			{
				Slug:   "awp-asiimov",
				Name:   "Asiimov",
				Weapon: "AWP",
			},
		},
		Total:      1,
		Page:       1,
		PageSize:   5,
		TotalPages: 1,
	}

	suite.mockCache.On("GetSkinList", suite.ctx, suite.service.generateCacheKey(filter)).
		Return(cachedResponse, true)

	result, err := suite.service.GetSkins(suite.ctx, filter)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(cachedResponse, result)
	suite.mockStorage.AssertNotCalled(suite.T(), "GetSkins")
}
func (suite *SkinServiceSuite) TestGetSkins_StorageError() {
	filter := &models.SkinFilter{
		Limit:  10,
		Offset: 0,
	}

	suite.mockCache.On("GetSkinList", suite.ctx, suite.service.generateCacheKey(filter)).
		Return(nil, false)

	suite.mockStorage.On("GetSkins", suite.ctx, filter).
		Return(nil, 0, errors.New("database connection failed"))

	result, err := suite.service.GetSkins(suite.ctx, filter)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "get skins from storage")
}

func (suite *SkinServiceSuite) TestGetSkinBySlug_Success() {
	slug := "ak47-redline"
	period := models.Period24h
	skinID := uuid.New()

	expectedSkin := &models.Skin{
		ID:             skinID,
		Slug:           slug,
		MarketHashName: "AK-47 | Redline",
		Name:           "Redline",
		Weapon:         "AK-47",
		CurrentPrice:   15.50,
	}

	expectedHistory := []models.PriceHistory{
		{
			ID:         uuid.New(),
			SkinID:     skinID,
			Price:      15.00,
			Currency:   "USD",
			Source:     "steam",
			Volume:     50,
			RecordedAt: time.Now().Add(-24 * time.Hour),
		},
	}

	expectedStats := &models.SkinStatistics{
		AvgPrice7d:      15.20,
		AvgPrice30d:     14.80,
		TotalVolume7d:   350,
		PriceVolatility: 0.5,
		ViewCount:       0,
	}

	suite.mockCache.On("GetSkinDetail", suite.ctx, slug).
		Return(nil, false)

	suite.mockStorage.On("GetSkinBySlug", suite.ctx, slug).
		Return(expectedSkin, nil)

	suite.mockStorage.On("GetPriceHistory", suite.ctx, skinID, period).
		Return(expectedHistory, nil)

	suite.mockStorage.On("GetSkinStatistics", suite.ctx, skinID).
		Return(expectedStats, nil)

	viewKey := "analytics:views:" + skinID.String()
	suite.mockCache.On("Get", suite.ctx, viewKey).
		Return("", errors.New("not found"))

	suite.mockCache.On("SetSkinDetail", suite.ctx, slug, &models.SkinDetailResponse{
		Skin:         *expectedSkin,
		PriceHistory: expectedHistory,
		Statistics:   *expectedStats,
	}, 5*time.Minute).
		Return(nil)

	suite.mockCache.On("Set", suite.ctx, viewKey, "1", 24*time.Hour).
		Return(nil)

	result, err := suite.service.GetSkinBySlug(suite.ctx, slug, period)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(expectedSkin.Slug, result.Skin.Slug)
	suite.Equal(len(expectedHistory), len(result.PriceHistory))
	suite.Equal(expectedStats.AvgPrice7d, result.Statistics.AvgPrice7d)
}

func (suite *SkinServiceSuite) TestGetSkinBySlug_CacheHit() {
	slug := "awp-dragon-lore"
	period := models.Period24h
	skinID := uuid.New()

	cachedResponse := &models.SkinDetailResponse{
		Skin: models.Skin{
			ID:   skinID,
			Slug: slug,
			Name: "Dragon Lore",
		},
		PriceHistory: []models.PriceHistory{},
		Statistics:   models.SkinStatistics{},
	}

	suite.mockCache.On("GetSkinDetail", suite.ctx, slug).
		Return(cachedResponse, true)

	viewKey := "analytics:views:" + skinID.String()
	suite.mockCache.On("Set", suite.ctx, viewKey, "1", 24*time.Hour).
		Return(nil)

	result, err := suite.service.GetSkinBySlug(suite.ctx, slug, period)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(cachedResponse, result)
	suite.mockStorage.AssertNotCalled(suite.T(), "GetSkinBySlug")
}

func (suite *SkinServiceSuite) TestGetSkinBySlug_NotFound() {
	slug := "non-existent-skin"
	period := models.Period24h

	suite.mockCache.On("GetSkinDetail", suite.ctx, slug).
		Return(nil, false)

	suite.mockStorage.On("GetSkinBySlug", suite.ctx, slug).
		Return(nil, errors.New("skin not found"))

	result, err := suite.service.GetSkinBySlug(suite.ctx, slug, period)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "get skin by slug")
}

func (suite *SkinServiceSuite) TestSearchSkins_Success() {
	query := "asiimov"
	limit := 10

	expectedSkins := []models.Skin{
		{
			Slug: "awp-asiimov",
			Name: "Asiimov",
		},
		{
			Slug: "m4a4-asiimov",
			Name: "Asiimov",
		},
	}

	suite.mockStorage.On("SearchSkins", suite.ctx, query, limit).
		Return(expectedSkins, nil)

	searchKey := "analytics:popular:search:" + query
	suite.mockCache.On("Set", suite.ctx, searchKey, "1", 24*time.Hour).
		Return(nil)

	result, err := suite.service.SearchSkins(suite.ctx, query, limit)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(2, len(result))
	suite.Equal("awp-asiimov", result[0].Slug)
}

func (suite *SkinServiceSuite) TestGetPopularSkins_Success() {
	limit := 5

	expectedSkins := []models.Skin{
		{Slug: "awp-dragon-lore", Volume24h: 1000},
		{Slug: "ak47-fire-serpent", Volume24h: 800},
	}

	cacheKey := "skins:popular:5"

	suite.mockCache.On("Get", suite.ctx, cacheKey).
		Return("", errors.New("not found"))

	suite.mockStorage.On("GetPopularSkins", suite.ctx, limit).
		Return(expectedSkins, nil)

	suite.mockCache.On("Set", suite.ctx, cacheKey, mock.MatchedBy(func(s string) bool {
		return len(s) > 0
	}), 5*time.Minute).
		Return(nil)

	result, err := suite.service.GetPopularSkins(suite.ctx, limit)

	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(2, len(result))
	suite.Equal("awp-dragon-lore", result[0].Slug)
}

func (suite *SkinServiceSuite) TestCreateSkin_Success() {
	skin := &models.Skin{
		ID:             uuid.New(),
		Slug:           "ak47-nightwish-ft",
		MarketHashName: "AK-47 | Nightwish (Field-Tested)",
		Name:           "Nightwish",
		Weapon:         "AK-47",
		Quality:        "Field-Tested",
		Rarity:         "Covert",
		CurrentPrice:   0,
		Currency:       "USD",
		ImageURL:       "https://example.com/img.jpg",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	suite.mockStorage.On("CreateSkin", suite.ctx, skin).
		Return(nil)

	suite.mockCache.On("Delete", suite.ctx, "skins:list:*").
		Return(nil)

	err := suite.service.CreateSkin(suite.ctx, skin)

	suite.NoError(err)
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *SkinServiceSuite) TestCreateSkin_StorageError() {
	skin := &models.Skin{
		Slug:           "ak47-nightwish-ft",
		MarketHashName: "AK-47 | Nightwish (Field-Tested)",
		Name:           "Nightwish",
		Weapon:         "AK-47",
		Quality:        "Field-Tested",
	}

	suite.mockStorage.On("CreateSkin", suite.ctx, skin).
		Return(errors.New("database error"))

	err := suite.service.CreateSkin(suite.ctx, skin)

	suite.Error(err)
	suite.Contains(err.Error(), "create skin")
	suite.mockStorage.AssertExpectations(suite.T())
}

func (suite *SkinServiceSuite) TestCreateSkin_InvalidatesCache() {
	skin := &models.Skin{
		Slug:           "ak47-nightwish-ft",
		MarketHashName: "AK-47 | Nightwish (Field-Tested)",
		Name:           "Nightwish",
		Weapon:         "AK-47",
		Quality:        "Field-Tested",
	}

	suite.mockStorage.On("CreateSkin", suite.ctx, skin).
		Return(nil)

	suite.mockCache.On("Delete", suite.ctx, "skins:list:*").
		Return(nil)

	err := suite.service.CreateSkin(suite.ctx, skin)

	suite.NoError(err)
	suite.mockCache.AssertCalled(suite.T(), "Delete", suite.ctx, "skins:list:*")
}
