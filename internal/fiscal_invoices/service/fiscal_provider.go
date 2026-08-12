package service

import (
	"context"

	nfeEmissionDomain "github.com/ProTrack-Solutions/protrack-api/internal/nfe_emission/domain"
)

// FiscalProvider abstrai o provedor de emissão fiscal (hoje PlugNotas).
// Qualquer implementação futura (outro provedor, ou uma API fiscal própria)
// só precisa satisfazer esta interface para ser usada por FiscalInvoicesService
// sem alterar domain, handler ou route deste módulo.
//
// *plugnotas.Client já satisfaz esta interface hoje, pois seus métodos têm
// exatamente estas assinaturas.
type FiscalProvider interface {
	// CadastrarEmpresa registra ou atualiza a empresa emitente no provedor.
	CadastrarEmpresa(ctx context.Context, payload nfeEmissionDomain.CadastrarEmpresaPayload) error

	// ConfigurarWebhook registra a URL de callback e o segredo compartilhado
	// para a empresa identificada por cnpj.
	ConfigurarWebhook(ctx context.Context, cnpj, callbackURL, sharedSecret string) error

	// UploadCertificado envia o certificado A1 (bytes brutos do .pfx/.p12) ao
	// provedor. Retorna o ID do certificado no provedor.
	UploadCertificado(ctx context.Context, certBytes []byte, senha string) (providerCertID string, err error)

	// EmitirNFe envia uma NF-e (modelo 55) para emissão. Retorna o ID de
	// rastreio no provedor (usado depois em ConsultarNFe/CancelarNFe).
	EmitirNFe(ctx context.Context, payload nfeEmissionDomain.NFePayload) (providerInvoiceID string, err error)

	// ConsultarNFe consulta o status atual de uma NF-e já enviada.
	ConsultarNFe(ctx context.Context, providerInvoiceID string) (nfeEmissionDomain.ConsultaResumoResponse, error)

	// CancelarNFe solicita o cancelamento de uma NF-e autorizada.
	CancelarNFe(ctx context.Context, providerInvoiceID, justificativa string) error

	// EmitirNFCe envia uma NFC-e (modelo 65) para emissão.
	EmitirNFCe(ctx context.Context, payload nfeEmissionDomain.NFCePayload) (providerInvoiceID string, err error)

	// ConsultarNFCe consulta o status atual de uma NFC-e já enviada.
	ConsultarNFCe(ctx context.Context, providerInvoiceID string) (nfeEmissionDomain.ConsultaResumoResponse, error)

	// CancelarNFCe solicita o cancelamento de uma NFC-e autorizada.
	CancelarNFCe(ctx context.Context, providerInvoiceID, justificativa string) error
}
