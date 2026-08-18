package metagraph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(httpClient *http.Client) *Client {
	return &Client{httpClient: httpClient, baseURL: "https://graph.facebook.com/v25.0"}
}

// GraphAPIError representa o formato de erro padrão que a Meta retorna:
// {"error": {"message": "...", "type": "...", "code": ..., ...}}
type GraphAPIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
}

type graphErrorResponse struct {
	Error GraphAPIError `json:"error"`
}

// Do executa uma chamada genérica à Graph API, injeta o Bearer token,
// e retorna o corpo cru da resposta em caso de sucesso.
func (c *Client) Do(ctx context.Context, method, path, accessToken string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("erro ao serializar corpo da requisição: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	url := c.baseURL + "/" + path

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("erro ao montar requisição: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar Graph API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta da Graph API: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp graphErrorResponse
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("erro da Graph API (código %d): %s", errResp.Error.Code, errResp.Error.Message)
		}
		return nil, fmt.Errorf("erro da Graph API, status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
