package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cs-parser/pkg/logger"
	"github.com/cs-parser/pkg/redis"
)

const (
	_steamMarketBaseURL   = "https://steamcommunity.com/market"
	_steamMarketPriceURL  = _steamMarketBaseURL + "/priceoverview"
	_steamMarketSearchURL = _steamMarketBaseURL + "/search/render"
	_appID                = "730" // CS2 App ID
	_defaultTimeout       = 10 * time.Second
	_cacheKeyPrefix       = "steam:cache:"
	_cacheTTL             = 2 * time.Minute
)

// SteamClient - клиент для Steam Market API
type SteamClient struct {
	httpClient *http.Client
	redis      *redis.Redis
	log        *logger.Logger
}

// NewSteamClient - создать Steam клиент
func NewSteamClient(redis *redis.Redis, log *logger.Logger) *SteamClient {
	return &SteamClient{
		httpClient: &http.Client{
			Timeout: _defaultTimeout,
		},
		redis: redis,
		log:   log,
	}
}

// ItemPrice - цена предмета
type ItemPrice struct {
	Price       float64 `json:"price"`
	Currency    string  `json:"currency"`
	Volume      int     `json:"volume"`
	LowestPrice float64 `json:"lowest_price"`
}

// GetItemPrice - получить цену предмета
func (c *SteamClient) GetItemPrice(ctx context.Context, marketHashName string) (*ItemPrice, error) {
	// Проверить кэш
	cacheKey := _cacheKeyPrefix + "price:" + marketHashName
	if cached, err := c.redis.GetCache(ctx, cacheKey); err == nil {
		var price ItemPrice
		if err := json.Unmarshal([]byte(cached), &price); err == nil {
			c.log.Debug("Price loaded from cache", "item", marketHashName)
			return &price, nil
		}
	}

	// Подготовить запрос
	params := url.Values{}
	params.Set("appid", _appID)
	params.Set("market_hash_name", marketHashName)
	params.Set("currency", "1") // USD

	reqURL := fmt.Sprintf("%s?%s", _steamMarketPriceURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Заголовки для обхода блокировок
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	// Выполнить запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bad status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Прочитать ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Парсинг Steam ответа
	var steamResp struct {
		Success     bool   `json:"success"`
		LowestPrice string `json:"lowest_price"`
		Volume      string `json:"volume"`
		MedianPrice string `json:"median_price"`
	}

	if err := json.Unmarshal(body, &steamResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !steamResp.Success {
		return nil, fmt.Errorf("steam api returned success=false")
	}

	// Парсинг цены (формат: "$15.50")
	price, err := parseSteamPrice(steamResp.LowestPrice)
	if err != nil {
		return nil, fmt.Errorf("parse price: %w", err)
	}

	medianPrice, _ := parseSteamPrice(steamResp.MedianPrice)

	// Парсинг объёма
	volume := 0
	if steamResp.Volume != "" {
		volume, _ = strconv.Atoi(strings.ReplaceAll(steamResp.Volume, ",", ""))
	}

	result := &ItemPrice{
		Price:       price,
		Currency:    "USD",
		Volume:      volume,
		LowestPrice: price,
	}

	if medianPrice > 0 {
		result.Price = medianPrice // используем median как основную цену
	}

	// Сохранить в кэш
	if data, err := json.Marshal(result); err == nil {
		if err := c.redis.SetCache(ctx, cacheKey, string(data), _cacheTTL); err != nil {
			c.log.Warn("Failed to cache price", "error", err)
		}
	}

	return result, nil
}

// SearchItem - элемент поиска
type SearchItem struct {
	MarketHashName string  `json:"market_hash_name"`
	Name           string  `json:"name"`
	Weapon         string  `json:"weapon"`
	Quality        string  `json:"quality"`
	Rarity         string  `json:"rarity"`
	Price          float64 `json:"price"`
	ImageURL       string  `json:"image_url"`
}

// SearchItems - поиск предметов
func (c *SteamClient) SearchItems(ctx context.Context, query string) ([]SearchItem, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("appid", _appID)
	params.Set("norender", "1")
	params.Set("count", "100")

	reqURL := fmt.Sprintf("%s?%s", _steamMarketSearchURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Парсинг Steam search response
	var steamResp struct {
		Success bool `json:"success"`
		Results []struct {
			HashName         string `json:"hash_name"`
			Name             string `json:"name"`
			SellPrice        int    `json:"sell_price"` // в центах
			AssetDescription struct {
				IconURL string `json:"icon_url"`
			} `json:"asset_description"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &steamResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if !steamResp.Success {
		return nil, fmt.Errorf("steam api returned success=false")
	}

	// Конвертация в наш формат
	items := make([]SearchItem, 0, len(steamResp.Results))
	for _, result := range steamResp.Results {
		weapon, quality := parseMarketHashName(result.HashName)

		item := SearchItem{
			MarketHashName: result.HashName,
			Name:           result.Name,
			Weapon:         weapon,
			Quality:        quality,
			Price:          float64(result.SellPrice) / 100.0, // центы -> доллары
			ImageURL:       "https://community.cloudflare.steamstatic.com/economy/image/" + result.AssetDescription.IconURL,
		}

		items = append(items, item)
	}

	return items, nil
}

// parseSteamPrice - парсинг цены из Steam формата ("$15.50")
func parseSteamPrice(priceStr string) (float64, error) {
	if priceStr == "" {
		return 0, nil
	}

	// Убрать валютный символ и пробелы
	priceStr = strings.TrimSpace(priceStr)
	priceStr = strings.TrimPrefix(priceStr, "$")
	priceStr = strings.TrimPrefix(priceStr, "€")
	priceStr = strings.TrimPrefix(priceStr, "£")
	priceStr = strings.ReplaceAll(priceStr, ",", "")

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return 0, fmt.Errorf("parse float: %w", err)
	}

	return price, nil
}

// parseMarketHashName - парсинг market_hash_name
// Пример: "AK-47 | Redline (Field-Tested)" -> weapon="AK-47", quality="Field-Tested"
func parseMarketHashName(marketHashName string) (weapon, quality string) {
	// Ищем качество в скобках
	if idx := strings.LastIndex(marketHashName, "("); idx != -1 {
		quality = strings.TrimSuffix(strings.TrimSpace(marketHashName[idx+1:]), ")")
		marketHashName = strings.TrimSpace(marketHashName[:idx])
	}

	// Ищем оружие (до первого "|")
	if idx := strings.Index(marketHashName, "|"); idx != -1 {
		weapon = strings.TrimSpace(marketHashName[:idx])
	} else {
		weapon = marketHashName
	}

	return weapon, quality
}

// GetMarketHistory - получить историю продаж (расширенная функция для будущего)
func (c *SteamClient) GetMarketHistory(ctx context.Context, marketHashName string, days int) ([]ItemPrice, error) {
	// TODO: Реализовать парсинг графика цен Steam
	// Steam возвращает JSON с историей цен за период
	// Формат: [{timestamp, price, volume}, ...]
	return nil, fmt.Errorf("not implemented")
}

// HealthCheck - проверка доступности Steam Market
func (c *SteamClient) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, _steamMarketBaseURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	return nil
}
