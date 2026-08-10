package cache

import (
	"context"
	"fmt"
	"time"

	redis_connection "github.com/ProTrack-Solutions/protrack-api/internal/adapters/redis"
	"github.com/redis/go-redis/v9"
)

// RateLimiter implementa um contador simples de janela fixa usando Redis
// (INCR + EXPIRE), suficiente para proteger endpoints públicos sensíveis
// como forgot-password contra abuso/spam.

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(redisClient *redis_connection.RedisClient) *RateLimiter {
	return &RateLimiter{
		client: redisClient.GetClient(),
	}
}

// Allow retorna true se a ação identificada por "key" ainda está dentro do
// limite "max" na janela de tempo "window". Cada chamada conta como 1 hit.
func (r *RateLimiter) Allow(ctx context.Context, key string, max int64, window time.Duration) (bool, error) {
	fullKey := "rate_limit:" + key

	count, err := r.client.Incr(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("incrementando contador de rate limit: %w", err)
	}

	if count == 1 {
		if err := r.client.Expire(ctx, fullKey, window).Err(); err != nil {
			return false, fmt.Errorf("configurando TTL do rate limit: %w", err)
		}
	}

	return count <= max, nil
}
