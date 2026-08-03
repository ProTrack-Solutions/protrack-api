package discord

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/ProTrack-Solutions/protrack-api/internal/logger/discord/domain"
	"github.com/rs/zerolog/log"
)

// Cores no formato decimal aceito pelo Discord
var ColorMap = map[domain.LogLevel]int{
	domain.LevelInfo:    3447003,  // Azul
	domain.LevelWarning: 16776960, // Amarelo
	domain.LevelError:   15158332, // Vermelho
}

type DiscordLogger struct {
	webhookURL string
	client     *http.Client
}

func NewDiscordLogger(cfg *config.Config) *DiscordLogger {
	return &DiscordLogger{
		webhookURL: cfg.WebhookURLDiscord,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (d *DiscordLogger) Send(level domain.LogLevel, title, message string) {
	if d.webhookURL == "" {
		log.Error().Msg("Webhook URL para o Discord não está configurado")
		return
	}

	go func() {
		payload := domain.DiscordPayload{
			Embeds: []domain.DiscordEmbed{
				{
					Title:       title,
					Description: message,
					Color:       ColorMap[level],
					Timestamp:   time.Now().UTC().Format(time.RFC3339),
					Footer:      domain.DiscordEmbedFooter{Text: "ProTrack Go API Logger"},
				},
			},
		}

		if level == domain.LevelError {
			payload.Content = "@here - Alerta de Erro Crítico"
		}

		body, err := json.Marshal(payload)
		if err != nil {
			log.Error().Err(err).Msg("Falha ao serializar JSON para envio ao Discord")
			return
		}

		resp, err := d.client.Post(d.webhookURL, "application/json", bytes.NewBuffer(body))
		if err != nil {
			log.Error().Err(err).Msg("Falha ao enviar log para o Discord")
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			log.Error().Int("status", resp.StatusCode).Msg("Falha ao enviar log para o Discord")
		}
	}()
}
