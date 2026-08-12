package nfeemission

import (
	"context"
	"fmt"

	"github.com/ProTrack-Solutions/protrack-api/internal/nfe_emission/domain"
)

// EmitirNFCe envia uma NFC-e (modelo 65) via POST /nfce.
func (c *Client) EmitirNFCe(ctx context.Context, payload domain.NFCePayload) (string, error) {
	var out domain.RegistrarNotaResponse
	if err := c.doJSON(ctx, "POST", "/nfce", []domain.NFCePayload{payload}, &out); err != nil {
		return "", fmt.Errorf("nfe-emission: falha ao emitir NFC-e: %w", err)
	}
	if len(out) == 0 {
		return "", fmt.Errorf("nfe-emission: resposta vazia ao emitir NFC-e")
	}
	return out[0].ID, nil
}

// ConsultarNFCe consulta o resumo via GET /nfce/{idNota}/resumo.
func (c *Client) ConsultarNFCe(ctx context.Context, idNota string) (domain.ConsultaResumoResponse, error) {
	var out domain.ConsultaResumoResponse
	err := c.doJSON(ctx, "GET", "/nfce/"+idNota+"/resumo", nil, &out)
	return out, err
}

// CancelarNFCe solicita o cancelamento via POST /nfce/{idNota}/cancelamento.
func (c *Client) CancelarNFCe(ctx context.Context, idNota, justificativa string) error {
	req := domain.CancelarNotaRequest{Justificativa: justificativa}
	return c.doJSON(ctx, "POST", "/nfce/"+idNota+"/cancelamento", req, nil)
}
