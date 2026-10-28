package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Sessão única por usuário e aplicação: session:{user}:{aud} guarda o id da
// sessão ativa. Um login novo grava outro id e os tokens da sessão anterior
// deixam de valer. Fica junto da blacklist por usar o mesmo Redis e já chegar
// em todos os middlewares de autenticação.

func sessionKey(userID, aud string) string {
	return fmt.Sprintf("session:%s:%s", userID, aud)
}

// SetSession grava (ou substitui) a sessão ativa do usuário na aplicação
func (tb *TokenBlacklist) SetSession(ctx context.Context, userID, aud, sessionID string, ttl time.Duration) error {
	return tb.client.Set(ctx, sessionKey(userID, aud), sessionID, ttl).Err()
}

// GetSession devolve o id da sessão ativa; vazio se não houver
func (tb *TokenBlacklist) GetSession(ctx context.Context, userID, aud string) (string, error) {
	sid, err := tb.client.Get(ctx, sessionKey(userID, aud)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to read session: %w", err)
	}
	return sid, nil
}

// TouchSession estende a validade da sessão na renovação do token
func (tb *TokenBlacklist) TouchSession(ctx context.Context, userID, aud string, ttl time.Duration) error {
	return tb.client.Expire(ctx, sessionKey(userID, aud), ttl).Err()
}

// EndSession apaga a sessão só se ela ainda for a informada: o logout de uma
// sessão já substituída não derruba a nova.
func (tb *TokenBlacklist) EndSession(ctx context.Context, userID, aud, sessionID string) error {
	const script = `if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) end return 0`
	return tb.client.Eval(ctx, script, []string{sessionKey(userID, aud)}, sessionID).Err()
}

// RevokeUserSessions derruba as sessões do usuário em todas as aplicações
// (troca de senha, usuário desativado ou removido)
func (tb *TokenBlacklist) RevokeUserSessions(ctx context.Context, userID string, auds ...string) error {
	keys := make([]string, 0, len(auds))
	for _, aud := range auds {
		keys = append(keys, sessionKey(userID, aud))
	}
	if len(keys) == 0 {
		return nil
	}
	return tb.client.Del(ctx, keys...).Err()
}
