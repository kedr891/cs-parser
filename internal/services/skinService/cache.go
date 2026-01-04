package skinservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/kedr891/cs-parser/internal/models"
)

type SkinsCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewSkinsCache(client *redis.Client, ttlSeconds int) *SkinsCache {
	return &SkinsCache{
		client: client,
		ttl:    time.Duration(ttlSeconds) * time.Second,
	}
}

func (c *SkinsCache) key(prefix, id string) string {
	return fmt.Sprintf("skins:%s:%s", prefix, id)
}

func (c *SkinsCache) Get(ctx context.Context, key string) (string, error) {
	if c == nil || c.client == nil {
		return "", fmt.Errorf("cache not initialized")
	}
	return c.client.Get(ctx, key).Result()
}

func (c *SkinsCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("cache not initialized")
	}
	if ttl == 0 {
		ttl = c.ttl
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *SkinsCache) Delete(ctx context.Context, key string) error {
	if c == nil || c.client == nil {
		return fmt.Errorf("cache not initialized")
	}
	return c.client.Del(ctx, key).Err()
}

func (c *SkinsCache) GetSkinList(ctx context.Context, cacheKey string) (*models.SkinListResponse, bool) {
	data, err := c.Get(ctx, cacheKey)
	if err != nil {
		return nil, false
	}

	var response models.SkinListResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		return nil, false
	}
	return &response, true
}

func (c *SkinsCache) SetSkinList(ctx context.Context, cacheKey string, response *models.SkinListResponse, ttl time.Duration) error {
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}
	return c.Set(ctx, cacheKey, string(data), ttl)
}

func (c *SkinsCache) GetSkinDetail(ctx context.Context, slug string) (*models.SkinDetailResponse, bool) {
	cacheKey := c.key("detail", slug)
	data, err := c.Get(ctx, cacheKey)
	if err != nil {
		return nil, false
	}

	var response models.SkinDetailResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		return nil, false
	}
	return &response, true
}

func (c *SkinsCache) SetSkinDetail(ctx context.Context, slug string, response *models.SkinDetailResponse, ttl time.Duration) error {
	cacheKey := c.key("detail", slug)
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}
	return c.Set(ctx, cacheKey, string(data), ttl)
}

func (c *SkinsCache) InvalidateSkin(ctx context.Context, skinID string) error {
	pattern := fmt.Sprintf("skins:*:%s", skinID)
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		_ = c.client.Del(ctx, iter.Val())
	}
	return iter.Err()
}

