package nfeemission

import (
	"context"

	"github.com/ProTrack-Solutions/protrack-api/internal/nfe_emission/domain"
)

func (c *Client) CadastrarEmpresa(ctx context.Context, payload domain.CadastrarEmpresaPayload) error {
	return c.doJSON(ctx, "POST", "/empresa", payload, nil)
}

func (c *Client) ConfigurarWebhook(ctx context.Context, cnpj, callbackURL, sharedSecret string) error {
	payload := domain.WebhookPayload{
		URL:    callbackURL,
		Method: "POST",
		Headers: map[string]string{
			"Authorization": sharedSecret,
		},
	}
	return c.doJSON(ctx, "POST", "/empresa/"+cnpj+"/webhook", payload, nil)
}
