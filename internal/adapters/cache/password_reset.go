package cache

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	redis_connection "github.com/ProTrack-Solutions/protrack-api/internal/adapters/redis"
	"github.com/redis/go-redis/v9"
)

const (
	passwordResetKeyPrefix     = "password_reset:token:"
	passwordResetUserKeyPrefix = "password_reset:user:"
	passwordResetTTL           = 30 * time.Minute
)

// ErrResetTokenNotFound é retornado quando o token não existe ou expirou.
var ErrResetTokenNotFound = errors.New("reset token not found or expired")

// PasswordResetStore guarda, no Redis, o hash de tokens de recuperação de
// senha vinculados a um userID, com expiração automática (TTL) e uso único.
type PasswordResetStore struct {
	client *redis.Client
}

func NewPasswordResetStore(redisClient *redis_connection.RedisClient) *PasswordResetStore {
	return &PasswordResetStore{
		client: redisClient.GetClient(),
	}
}

func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// GenerateToken cria um token aleatório criptograficamente seguro (32 bytes).
// O valor em texto puro é o que vai no e-mail; apenas o hash é persistido.
func (s *PasswordResetStore) GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("gerando token aleatório: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Store salva o hash do token vinculado ao userID e invalida qualquer
// token anterior daquele usuário (garante no máximo 1 token válido por vez).
func (s *PasswordResetStore) Store(ctx context.Context, userID string, token string) error {
	if err := s.InvalidateByUserID(ctx, userID); err != nil {
		return fmt.Errorf("invalidando token anterior: %w", err)
	}

	tokenHash := hashResetToken(token)

	pipe := s.client.Pipeline()
	pipe.Set(ctx, passwordResetKeyPrefix+tokenHash, userID, passwordResetTTL)
	pipe.Set(ctx, passwordResetUserKeyPrefix+userID, tokenHash, passwordResetTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("salvando token de reset: %w", err)
	}

	return nil
}

// Validate verifica se o token existe e retorna o userID associado.
func (s *PasswordResetStore) Validate(ctx context.Context, token string) (string, error) {
	userID, err := s.client.Get(ctx, passwordResetKeyPrefix+hashResetToken(token)).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrResetTokenNotFound
	}
	if err != nil {
		return "", fmt.Errorf("validando token de reset: %w", err)
	}
	return userID, nil
}

// Invalidate remove o token após o uso, garantindo uso único.
func (s *PasswordResetStore) Invalidate(ctx context.Context, token string) error {
	tokenHash := hashResetToken(token)
	pipe := s.client.Pipeline()
	pipe.Del(ctx, passwordResetKeyPrefix+tokenHash)
	_, err := pipe.Exec(ctx)
	return err
}

// InvalidateByUserID remove qualquer token de reset pendente de um usuário.
func (s *PasswordResetStore) InvalidateByUserID(ctx context.Context, userID string) error {
	tokenHash, err := s.client.Get(ctx, passwordResetUserKeyPrefix+userID).Result()
	if errors.Is(err, redis.Nil) {
		return nil // não há token pendente
	}
	if err != nil {
		return fmt.Errorf("buscando token anterior do usuário: %w", err)
	}

	pipe := s.client.Pipeline()
	pipe.Del(ctx, passwordResetKeyPrefix+tokenHash)
	pipe.Del(ctx, passwordResetUserKeyPrefix+userID)
	_, err = pipe.Exec(ctx)
	return err
}
