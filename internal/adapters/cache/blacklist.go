package cache

import (
	"context"
	"fmt"
	"time"

	redis_connection "github.com/ProTrack-Solutions/protrack-api/internal/adapters/redis"
	"github.com/redis/go-redis/v9"
)

type TokenBlacklist struct {
	client *redis.Client
}

func NewTokenBlackList(redisClient *redis_connection.RedisClient) *TokenBlacklist {
	return &TokenBlacklist{
		client: redisClient.GetClient(),
	}
}

func (tb *TokenBlacklist) AddToken(ctx context.Context, tokenID string, expiresIn time.Duration) error {
	key := fmt.Sprintf("blacklist:token:%s", tokenID)
	return tb.client.Set(ctx, key, 1, expiresIn).Err()
}

func (tb TokenBlacklist) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	key := fmt.Sprintf("blacklist:token:%s", tokenID)

	exists, err := tb.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}

	return exists > 0, nil
}

func (tb *TokenBlacklist) RemoveToken(ctx context.Context, tokenID string) error {
	key := fmt.Sprintf("blacklist:token:%s", tokenID)
	return tb.client.Del(ctx, key).Err()
}

func (tb *TokenBlacklist) AddRefreshToken(ctx context.Context, refreshTokenID string, expiresIn time.Duration) error {
	key := fmt.Sprintf("blacklist:refresh:%s", refreshTokenID)
	return tb.client.Set(ctx, key, "1", expiresIn).Err()
}

func (tb *TokenBlacklist) IsRefreshBlacklisted(ctx context.Context, refreshTokenID string) (bool, error) {
	key := fmt.Sprintf("blacklist:refresh:%s", refreshTokenID)

	exists, err := tb.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check refresh token blacklist: %w", err)
	}
	return exists > 0, nil
}

func (tb *TokenBlacklist) ClearUserTokens(ctx context.Context, userID string) error {
	patterns := []string{
		fmt.Sprintf("blacklist:token:%s*", userID),
		fmt.Sprintf("blacklist:refresh:%s*", userID),
	}

	for _, pattern := range patterns {
		keys, err := tb.client.Keys(ctx, pattern).Result()
		if err != nil {
			return fmt.Errorf("failed to get user tokens: %w", err)
		}

		if len(keys) > 0 {
			if err := tb.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
	}

	return nil
}
