package nfeemission

import (
	"context"
	"fmt"

	"github.com/ProTrack-Solutions/protrack-api/internal/nfe_emission/domain"
)

// EmitirNFe envia uma NF-e (modelo 55) via POST /nfe. O payload é um array
// mesmo para uma única nota, conforme exigido pela API.
func (c *Client) EmitirNFe(ctx context.Context, payload domain.NFePayload) (string, error) {
	var out domain.RegistrarNotaResponse
	if err := c.doJSON(ctx, "POST", "/nfe", []domain.NFePayload{payload}, &out); err != nil {
		return "", fmt.Errorf("nfe-emission: falha ao emitir NF-e: %w", err)
	}

	if len(out) == 0 {
		return "", fmt.Errorf("nfe-emission: resposta vazia ao emitir NF-e")
	}
	return out[0].ID, nil
}

// ConsultarNFe consulta o resumo de uma NF-e via GET /nfe/{idNota}/resumo.
func (c *Client) ConsultarNFe(ctx context.Context, idNota string) (domain.ConsultaResumoResponse, error) {
	var out domain.ConsultaResumoResponse
	err := c.doJSON(ctx, "GET", "/nfe/"+idNota+"/resumo", nil, &out)
	return out, err
}

func (c *Client) CancelarNFe(ctx context.Context, idNota, justificativa string) error {
	req := domain.CancelarNotaRequest{Justificativa: justificativa}
	return c.doJSON(ctx, "POST", "/nfe/"+idNota+"/cancelamento", req, nil)
}
