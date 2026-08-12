package nfeemission

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ProTrack-Solutions/protrack-api/internal/config"
)

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		apiKey:  cfg.FiscalAPIKey,
		baseURL: cfg.FiscalBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("plugnotas: falha ao serializar corpo da requisição: %w", err)
		}

		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("nfe-emission: falha ao montar requisição: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("nfe-emission: falha de rede: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("nfe-emission: falha ao ler resposta: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("nfe-emission: erro %d: %s", resp.StatusCode, string(respBody))
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("nfe-emission: falha ao decodificar resposta: %w", err)
		}
	}

	return nil
}
