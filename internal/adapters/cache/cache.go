package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	redis_connection "github.com/ProTrack-Solutions/protrack-api/internal/adapters/redis"
	"github.com/redis/go-redis/v9"
)

type CacheService struct {
	client *redis.Client
}

func NewCacheService(redisClient *redis_connection.RedisClient) *CacheService {
	return &CacheService{
		client: redisClient.GetClient(),
	}
}

func (c *CacheService) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	var data []byte
	var err error

	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		data, err = json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
	}

	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *CacheService) Get(ctx context.Context, key string) (string, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrCacheMiss
	}
	if err != nil {
		return "", fmt.Errorf("failed to get from cache: %w", err)
	}
	return val, nil
}

func (c *CacheService) GetJSON(ctx context.Context, key string, dest any) error {
	val, err := c.Get(ctx, key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

func (c *CacheService) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *CacheService) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	return c.client.Del(ctx, keys...).Err()
}

func (c *CacheService) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (c *CacheService) SetNX(ctx context.Context, key string, value any, ttl time.Duration) (bool, error) {
	var data []byte
	var err error

	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		data, err = json.Marshal(value)
		if err != nil {
			return false, fmt.Errorf("failed to marshal value: %w", err)
		}
	}

	return c.client.SetNX(ctx, key, data, ttl).Result()
}

func (c *CacheService) Increment(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

func (c *CacheService) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

var ErrCacheMiss = fmt.Errorf("cache miss")
