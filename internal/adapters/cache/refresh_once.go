package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrRefreshReused: o refresh token já foi trocado e passou a janela de tolerância
var ErrRefreshReused = errors.New("refresh token already used")

const (
	// Quanto tempo o mesmo refresh token ainda recebe o par já gerado. O web
	// renova em paralelo (proxy do Next, /api/auth/session e requisições ao
	// mesmo tempo): sem isso, a segunda renovação falhava e derrubava a sessão.
	refreshGrace   = 60 * time.Second
	refreshLockTTL = 10 * time.Second
	refreshWait    = 3 * time.Second
	refreshPoll    = 50 * time.Millisecond
)

func refreshResultKey(tokenID string) string { return fmt.Sprintf("refresh:result:%s", tokenID) }
func refreshLockKey(tokenID string) string   { return fmt.Sprintf("refresh:lock:%s", tokenID) }

// RefreshOnce troca o refresh token uma única vez. Pedidos simultâneos ou
// repetidos dentro da janela de tolerância recebem o mesmo resultado; depois
// dela, o token usado é recusado (ErrRefreshReused). usedTTL é quanto falta
// para o token expirar (tempo em que ele fica marcado como usado).
func (tb *TokenBlacklist) RefreshOnce(ctx context.Context, tokenID string, usedTTL time.Duration, refresh func() ([]byte, error)) ([]byte, error) {
	deadline := time.Now().Add(refreshWait)
	for {
		cached, err := tb.client.Get(ctx, refreshResultKey(tokenID)).Bytes()
		if err == nil {
			return cached, nil
		}
		if !errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("failed to read refresh result: %w", err)
		}

		used, err := tb.IsRefreshBlacklisted(ctx, tokenID)
		if err != nil {
			return nil, err
		}
		if used {
			return nil, ErrRefreshReused
		}

		locked, err := tb.client.SetNX(ctx, refreshLockKey(tokenID), 1, refreshLockTTL).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to lock refresh: %w", err)
		}
		if locked {
			return tb.runRefresh(ctx, tokenID, usedTTL, refresh)
		}

		// Outro pedido está renovando com este token: espera o resultado dele
		if time.Now().After(deadline) {
			return nil, ErrRefreshReused
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(refreshPoll):
		}
	}
}

func (tb *TokenBlacklist) runRefresh(ctx context.Context, tokenID string, usedTTL time.Duration, refresh func() ([]byte, error)) ([]byte, error) {
	defer tb.client.Del(context.WithoutCancel(ctx), refreshLockKey(tokenID))

	result, err := refresh()
	if err != nil {
		return nil, err
	}

	// Primeiro o resultado, depois a marca de usado: quem chegar no meio já acha o par novo
	grace := min(refreshGrace, max(usedTTL, time.Second))
	if err := tb.client.Set(ctx, refreshResultKey(tokenID), result, grace).Err(); err != nil {
		return nil, fmt.Errorf("failed to store refresh result: %w", err)
	}
	if usedTTL > 0 {
		if err := tb.AddRefreshToken(ctx, tokenID, usedTTL); err != nil {
			return nil, err
		}
	}
	return result, nil
}
