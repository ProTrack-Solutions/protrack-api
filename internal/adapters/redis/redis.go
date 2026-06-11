package redis_connection

import (
	"context"
	"fmt"
	"time"

	"github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedisConnection(cfg *config.Config) (*RedisClient, error) {
	addr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "",
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		log.Error().Err(err).Msg("failed to connect to Redis")
		return nil, err
	}

	log.Info().Msg("Connected to Redis successfully")

	return &RedisClient{
		Client: client,
	}, nil
}

func (r *RedisClient) Close() error {
	return r.Client.Close()
}

func (r *RedisClient) GetClient() *redis.Client {
	return r.Client
}

func (r *RedisClient) Health(ctx context.Context) error {
	_, err := r.Client.Ping(ctx).Result()
	return err
}
