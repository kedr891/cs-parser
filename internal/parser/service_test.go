package parser

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kedr891/cs-parser/internal/domain"
	"github.com/kedr891/cs-parser/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSkinRepository struct {
	mock.Mock
}

func (m *MockSkinRepository) GetAllSkins(ctx context.Context) ([]entity.Skin, error) {
	args := m.Called(ctx)
	return args.Get(0).([]entity.Skin), args.Error(1)
}

func (m *MockSkinRepository) GetSkinByID(ctx context.Context, id uuid.UUID) (*entity.Skin, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Skin), args.Error(1)
}

func (m *MockSkinRepository) GetSkinBySlug(ctx context.Context, slug string) (*entity.Skin, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Skin), args.Error(1)
}

func (m *MockSkinRepository) GetSkinByMarketHashName(ctx context.Context, marketHashName string) (*entity.Skin, error) {
	args := m.Called(ctx, marketHashName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Skin), args.Error(1)
}

func (m *MockSkinRepository) SkinExists(ctx context.Context, marketHashName string) (bool, error) {
	args := m.Called(ctx, marketHashName)
	return args.Bool(0), args.Error(1)
}

func (m *MockSkinRepository) CreateSkin(ctx context.Context, skin *entity.Skin) error {
	args := m.Called(ctx, skin)
	return args.Error(0)
}

func (m *MockSkinRepository) UpdateSkinPrice(ctx context.Context, skinID uuid.UUID, price float64, volume int) error {
	args := m.Called(ctx, skinID, price, volume)
	return args.Error(0)
}

func (m *MockSkinRepository) SavePriceHistory(ctx context.Context, history *entity.PriceHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

func (m *MockSkinRepository) GetSkinsCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

type MockMarketClient struct {
	mock.Mock
}

func (m *MockMarketClient) GetItemPrice(ctx context.Context, marketHashName string) (*domain.PriceData, error) {
	args := m.Called(ctx, marketHashName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PriceData), args.Error(1)
}

func (m *MockMarketClient) SearchItems(ctx context.Context, query string) ([]domain.MarketItem, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]domain.MarketItem), args.Error(1)
}

type MockCacheStorage struct {
	mock.Mock
}

func (m *MockCacheStorage) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCacheStorage) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheStorage) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheStorage) IncrementRateLimit(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	args := m.Called(ctx, key, ttl)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCacheStorage) GetRateLimit(ctx context.Context, key string) (int64, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCacheStorage) ZAdd(ctx context.Context, key string, score float64, member string) error {
	args := m.Called(ctx, key, score, member)
	return args.Error(0)
}

func (m *MockCacheStorage) ZIncrBy(ctx context.Context, key string, increment float64, member string) (float64, error) {
	args := m.Called(ctx, key, increment, member)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockCacheStorage) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	args := m.Called(ctx, key, start, stop)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockCacheStorage) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]domain.ZMember, error) {
	args := m.Called(ctx, key, start, stop)
	return args.Get(0).([]domain.ZMember), args.Error(1)
}

func (m *MockCacheStorage) HSet(ctx context.Context, key, field string, value interface{}) error {
	args := m.Called(ctx, key, field, value)
	return args.Error(0)
}

func (m *MockCacheStorage) HGet(ctx context.Context, key, field string) (string, error) {
	args := m.Called(ctx, key, field)
	return args.String(0), args.Error(1)
}

func (m *MockCacheStorage) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockCacheStorage) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheStorage) GetJSON(ctx context.Context, key string, dest interface{}) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}

func (m *MockCacheStorage) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheStorage) Increment(ctx context.Context, key string) (int64, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(int64), args.Error(1)
}

type MockMessageProducer struct {
	mock.Mock
}

func (m *MockMessageProducer) WriteMessage(ctx context.Context, key string, value interface{}) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockMessageProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, args ...interface{}) {
	m.Called(msg, args)
}

func (m *MockLogger) Info(msg string, args ...interface{}) {
	m.Called(msg, args)
}

func (m *MockLogger) Warn(msg string, args ...interface{}) {
	m.Called(msg, args)
}

func (m *MockLogger) Error(msg string, args ...interface{}) {
	m.Called(msg, args)
}

func (m *MockLogger) Fatal(msg string, args ...interface{}) {
	m.Called(msg, args)
}

func TestService_ParseAllSkins_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSkinRepository)
	mockClient := new(MockMarketClient)
	mockPriceProducer := new(MockMessageProducer)
	mockDiscoveryProducer := new(MockMessageProducer)
	mockCache := new(MockCacheStorage)
	mockLogger := new(MockLogger)

	testSkins := []entity.Skin{
		{
			ID:             uuid.New(),
			Slug:           "awp_acheron_ft",
			MarketHashName: "AWP | Acheron (Field-Tested)",
			Name:           "Acheron",
			Weapon:         "AWP",
			Quality:        "Field-Tested",
			CurrentPrice:   0.50,
			Volume24h:      1000,
		},
	}

	priceData := &domain.PriceData{
		Price:  0.58,
		Volume: 1397,
	}

	mockRepo.On("GetAllSkins", ctx).Return(testSkins, nil)
	mockCache.On("IncrementRateLimit", ctx, mock.Anything, mock.Anything).Return(int64(1), nil)
	mockClient.On("GetItemPrice", ctx, testSkins[0].MarketHashName).Return(priceData, nil)
	mockRepo.On("UpdateSkinPrice", ctx, testSkins[0].ID, priceData.Price, priceData.Volume).Return(nil)
	mockCache.On("Delete", ctx, mock.Anything).Return(nil)
	mockPriceProducer.On("WriteMessage", ctx, mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()

	service := NewService(
		mockRepo,
		mockClient,
		mockPriceProducer,
		mockDiscoveryProducer,
		mockCache,
		mockLogger,
	)

	err := service.ParseAllSkins(ctx)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockClient.AssertExpectations(t)
	mockCache.AssertExpectations(t)
	mockPriceProducer.AssertExpectations(t)
}

func TestService_ParseAllSkins_NoSkins(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSkinRepository)
	mockClient := new(MockMarketClient)
	mockPriceProducer := new(MockMessageProducer)
	mockDiscoveryProducer := new(MockMessageProducer)
	mockCache := new(MockCacheStorage)
	mockLogger := new(MockLogger)

	mockRepo.On("GetAllSkins", ctx).Return([]entity.Skin{}, nil)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Warn", mock.Anything, mock.Anything).Return()

	service := NewService(
		mockRepo,
		mockClient,
		mockPriceProducer,
		mockDiscoveryProducer,
		mockCache,
		mockLogger,
	)

	err := service.ParseAllSkins(ctx)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_ParseAllSkins_RepositoryError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSkinRepository)
	mockClient := new(MockMarketClient)
	mockPriceProducer := new(MockMessageProducer)
	mockDiscoveryProducer := new(MockMessageProducer)
	mockCache := new(MockCacheStorage)
	mockLogger := new(MockLogger)

	expectedError := errors.New("database error")
	mockRepo.On("GetAllSkins", ctx).Return([]entity.Skin{}, expectedError)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()

	service := NewService(
		mockRepo,
		mockClient,
		mockPriceProducer,
		mockDiscoveryProducer,
		mockCache,
		mockLogger,
	)

	err := service.ParseAllSkins(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get all skins")
	mockRepo.AssertExpectations(t)
}

func TestService_DiscoverNewSkins_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSkinRepository)
	mockClient := new(MockMarketClient)
	mockPriceProducer := new(MockMessageProducer)
	mockDiscoveryProducer := new(MockMessageProducer)
	mockCache := new(MockCacheStorage)
	mockLogger := new(MockLogger)

	searchQuery := "AWP"
	marketItems := []domain.MarketItem{
		{
			MarketHashName: "AWP | New Skin (Factory New)",
			Name:           "New Skin",
			Weapon:         "AWP",
			Quality:        "Factory New",
			Rarity:         "Classified",
			Price:          10.50,
			ImageURL:       "https://example.com/image.jpg",
		},
	}

	mockClient.On("SearchItems", ctx, searchQuery).Return(marketItems, nil)
	mockRepo.On("SkinExists", ctx, marketItems[0].MarketHashName).Return(false, nil)
	mockRepo.On("CreateSkin", ctx, mock.AnythingOfType("*entity.Skin")).Return(nil)
	mockDiscoveryProducer.On("WriteMessage", ctx, mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()

	service := NewService(
		mockRepo,
		mockClient,
		mockPriceProducer,
		mockDiscoveryProducer,
		mockCache,
		mockLogger,
	)

	err := service.DiscoverNewSkins(ctx, searchQuery)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockDiscoveryProducer.AssertExpectations(t)
}

func TestService_DiscoverNewSkins_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSkinRepository)
	mockClient := new(MockMarketClient)
	mockPriceProducer := new(MockMessageProducer)
	mockDiscoveryProducer := new(MockMessageProducer)
	mockCache := new(MockCacheStorage)
	mockLogger := new(MockLogger)

	searchQuery := "AWP"
	marketItems := []domain.MarketItem{
		{
			MarketHashName: "AWP | Existing Skin (Field-Tested)",
			Name:           "Existing Skin",
			Weapon:         "AWP",
			Quality:        "Field-Tested",
		},
	}

	mockClient.On("SearchItems", ctx, searchQuery).Return(marketItems, nil)
	mockRepo.On("SkinExists", ctx, marketItems[0].MarketHashName).Return(true, nil)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()

	service := NewService(
		mockRepo,
		mockClient,
		mockPriceProducer,
		mockDiscoveryProducer,
		mockCache,
		mockLogger,
	)

	err := service.DiscoverNewSkins(ctx, searchQuery)

	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "CreateSkin", mock.Anything, mock.Anything)
	mockDiscoveryProducer.AssertNotCalled(t, "WriteMessage", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_GetStats_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSkinRepository)
	mockClient := new(MockMarketClient)
	mockPriceProducer := new(MockMessageProducer)
	mockDiscoveryProducer := new(MockMessageProducer)
	mockCache := new(MockCacheStorage)
	mockLogger := new(MockLogger)

	expectedCount := 1500
	expectedRateLimit := int64(45)

	mockRepo.On("GetSkinsCount", ctx).Return(expectedCount, nil)
	mockCache.On("GetRateLimit", ctx, _rateLimitKey).Return(expectedRateLimit, nil)

	service := NewService(
		mockRepo,
		mockClient,
		mockPriceProducer,
		mockDiscoveryProducer,
		mockCache,
		mockLogger,
	)

	stats, err := service.GetStats(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, expectedCount, stats.TotalSkins)
	assert.Equal(t, int(expectedRateLimit), stats.RequestsLastMin)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestService_RateLimit(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockSkinRepository)
	mockClient := new(MockMarketClient)
	mockPriceProducer := new(MockMessageProducer)
	mockDiscoveryProducer := new(MockMessageProducer)
	mockCache := new(MockCacheStorage)
	mockLogger := new(MockLogger)

	testSkin := entity.Skin{
		ID:             uuid.New(),
		Slug:           "awp_test_ft",
		MarketHashName: "AWP | Test (Field-Tested)",
		CurrentPrice:   1.00,
	}

	mockRepo.On("GetAllSkins", ctx).Return([]entity.Skin{testSkin}, nil)
	mockCache.On("IncrementRateLimit", ctx, mock.Anything, mock.Anything).Return(int64(61), nil)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	service := NewService(
		mockRepo,
		mockClient,
		mockPriceProducer,
		mockDiscoveryProducer,
		mockCache,
		mockLogger,
	)

	err := service.ParseAllSkins(ctx)

	assert.NoError(t, err)
	mockCache.AssertExpectations(t)
}

func BenchmarkService_ParseAllSkins(b *testing.B) {
	ctx := context.Background()
	mockRepo := new(MockSkinRepository)
	mockClient := new(MockMarketClient)
	mockPriceProducer := new(MockMessageProducer)
	mockDiscoveryProducer := new(MockMessageProducer)
	mockCache := new(MockCacheStorage)
	mockLogger := new(MockLogger)

	skins := make([]entity.Skin, 100)
	for i := 0; i < 100; i++ {
		skins[i] = entity.Skin{
			ID:             uuid.New(),
			Slug:           "test_skin_ft",
			MarketHashName: "Test Skin (Field-Tested)",
			CurrentPrice:   1.00,
		}
	}

	priceData := &domain.PriceData{Price: 1.05, Volume: 100}

	mockRepo.On("GetAllSkins", ctx).Return(skins, nil)
	mockCache.On("IncrementRateLimit", ctx, mock.Anything, mock.Anything).Return(int64(1), nil)
	mockClient.On("GetItemPrice", ctx, mock.Anything).Return(priceData, nil)
	mockRepo.On("UpdateSkinPrice", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockCache.On("Delete", ctx, mock.Anything).Return(nil)
	mockPriceProducer.On("WriteMessage", ctx, mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()

	service := NewService(
		mockRepo,
		mockClient,
		mockPriceProducer,
		mockDiscoveryProducer,
		mockCache,
		mockLogger,
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.ParseAllSkins(ctx)
	}
}
