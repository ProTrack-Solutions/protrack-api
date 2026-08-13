package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
	"github.com/rs/zerolog/log"
)

type Whatsapp struct {
	cfg  *config.Config
	http *http.Client
	
}

type MessagePayload struct {
	Number string `json:"number"`
	Text   string `json:"text"`
}

func NewWhatsapp(cfg *config.Config) *Whatsapp {
	return &Whatsapp{
		cfg: cfg,
		http: &http.Client{
			Timeout: 1 * time.Minute,
		},
	}
}

func (w *Whatsapp) SendWhatsAppMessage(targetNumber string, messageContent string, instanceName string) error {
	apiURL := fmt.Sprintf("%s/message/sendText/%s", w.cfg.EvolutionApiUrl, instanceName)
	apiKey := w.cfg.EvolutionKey

	client := &http.Client{}
	var resp *http.Response
	var err error

	if apiURL == "" {
		log.Warn().Msg("Evolution API não configurada - envio ignorado")
		return nil
	}

	payload := MessagePayload{
		Number: targetNumber,
		Text:   messageContent,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", apiKey)

	for i := 0; i < 3; i++ {
		resp, err = client.Do(req)
		if err == nil {
			break
		}
		log.Warn().Msgf("Tentativa %d falhou, aguardando a Evolution API iniciar...", i+1)
		time.Sleep(20 * time.Second)
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("url", apiURL).
			Str("body", string(body)).
			Msg("A Evolution API recusou a requisição")
		return fmt.Errorf("erro na api: status %d corpo: %s", resp.StatusCode, string(body))
	}

	log.Info().Msg("WhatsApp enviado com sucesso!")
	return nil
}

func (w *Whatsapp) WhatsAppWebhookHandler(rw http.ResponseWriter, r *http.Request) {
	var data map[string]any

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	event := data["event"].(string)

	if event == "messages.upsert" {
		log.Info().Msg("Nova mensagem recebida!")
	}

	rw.WriteHeader(http.StatusOK)
}

// evolution_client.go

func (w *Whatsapp) SetWebhook(ctx context.Context, instance, webhookURL string) error {
	payload, _ := json.Marshal(map[string]any{
		"webhook": map[string]any{
			"enabled": true,
			"url":     webhookURL,
			"events":  []string{"MESSAGES_UPSERT"},
		},
	})

	url := fmt.Sprintf("%s/webhook/set/%s", w.cfg.EvolutionApiUrl, instance)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", w.cfg.EvolutionKey)

	resp, err := w.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call evolution api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("evolution api returned status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
