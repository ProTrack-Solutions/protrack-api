package nfeemission

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"

	"github.com/ProTrack-Solutions/protrack-api/internal/nfe_emission/domain"
)

// UploadCertificado envia o certificado A1 (.pfx/.p12) via multipart/form-data
// conforme POST /certificado. Retorna o ID do certificado no Emissor.
func (c *Client) UploadCertificado(ctx context.Context, certBytes []byte, senha string) (string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("arquivo", "certificado.pfx")
	if err != nil {
		return "", fmt.Errorf("nfe-emission: falha ao montar multipart: %w", err)
	}
	if _, err := part.Write(certBytes); err != nil {
		return "", fmt.Errorf("nfe-emission: falha ao escrever certificado: %w", err)
	}
	if err := writer.WriteField("senha", senha); err != nil {
		return "", fmt.Errorf("nfe-emission: falha ao escrever senha: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("nfe-emission: falha ao fechar multipart: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/certificado", &buf)
	if err != nil {
		return "", fmt.Errorf("nfe-emission: falha ao montar requisição: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("nfe-emission: falha de rede: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("nfe-emission: erro %d ao enviar certificado", resp.StatusCode)
	}

	var out domain.UploadCertificadoResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("nfe-emission: falha ao decodificar resposta: %w", err)
	}

	return out.Data.ID, nil
}
