package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord"
	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord/domain"
	"github.com/gin-gonic/gin"
)

const (
	// O Discord limita ~30 mensagens/min por webhook, então as requisições são
	// agrupadas e enviadas em lote a cada requestLogFlushInterval.
	requestLogFlushInterval = 5 * time.Second
	// Máximo de mensagens por lote (2 a cada 5s = 24/min, abaixo do limite).
	requestLogMaxMessagesPerFlush = 2
	// Limite de caracteres por mensagem (descrição do embed aceita até 4096).
	requestLogMaxMessageLen = 3800
	requestLogBufferSize    = 2000
)

type requestLogLine struct {
	text        string
	serverError bool
}

// GinDiscordRequestLogMiddleware registra todas as chamadas da API (método,
// rota, status, tempo de resposta e IP) no canal de log da API no Discord.
// Não registra query string, headers nem corpo para não vazar dados sensíveis.
func GinDiscordRequestLogMiddleware(discordLog *discord.DiscordLogger) gin.HandlerFunc {
	lines := make(chan requestLogLine, requestLogBufferSize)
	go flushRequestLogs(discordLog, lines)

	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		// Health check é chamado o tempo todo pela infraestrutura.
		if c.Request.URL.Path == "/health" {
			return
		}

		status := c.Writer.Status()
		line := requestLogLine{
			text: fmt.Sprintf("%s %d %-6s %s %s %s",
				start.UTC().Format("15:04:05"), status, c.Request.Method, c.Request.URL.Path,
				time.Since(start).Round(time.Millisecond), c.ClientIP()),
			serverError: status >= 500,
		}

		// Nunca bloqueia a requisição: se o buffer estiver cheio, descarta.
		select {
		case lines <- line:
		default:
		}
	}
}

func flushRequestLogs(discordLog *discord.DiscordLogger, lines <-chan requestLogLine) {
	ticker := time.NewTicker(requestLogFlushInterval)
	defer ticker.Stop()

	var pending []requestLogLine

	for {
		select {
		case line := <-lines:
			pending = append(pending, line)
		case <-ticker.C:
			pending = sendRequestLogBatch(discordLog, pending)
		}
	}
}

// sendRequestLogBatch envia até requestLogMaxMessagesPerFlush mensagens e
// devolve as linhas que não couberam para o próximo lote.
func sendRequestLogBatch(discordLog *discord.DiscordLogger, pending []requestLogLine) []requestLogLine {
	for sent := 0; sent < requestLogMaxMessagesPerFlush && len(pending) > 0; sent++ {
		var sb strings.Builder
		level := domain.LevelInfo
		count := 0

		for _, line := range pending {
			if sb.Len()+len(line.text)+1 > requestLogMaxMessageLen {
				break
			}
			sb.WriteString(line.text)
			sb.WriteByte('\n')
			if line.serverError {
				level = domain.LevelWarning
			}
			count++
		}

		pending = pending[count:]
		discordLog.Send(level, fmt.Sprintf("Requisições da API (%d) — horários em UTC", count), "```\n"+sb.String()+"```")
	}

	// Evita que o acúmulo cresça sem limite em picos de tráfego.
	if len(pending) > requestLogBufferSize {
		dropped := len(pending) - requestLogBufferSize
		pending = pending[dropped:]
		discordLog.Send(domain.LevelWarning, "Requisições da API", fmt.Sprintf("%d registros descartados por excesso de volume.", dropped))
	}

	return pending
}
