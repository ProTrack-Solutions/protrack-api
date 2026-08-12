package domain

// ---------------------------------------------------------------------------
// Certificado
// ---------------------------------------------------------------------------

type UploadCertificadoResponse struct {
	Message string `json:"message"`
	Data    struct {
		ID string `json:"id"`
	} `json:"data"`
}

// ---------------------------------------------------------------------------
// Empresa
// ---------------------------------------------------------------------------

type CadastrarEmpresaPayload struct {
	CpfCnpj           string          `json:"cpfCnpj"`
	InscricaoEstadual string          `json:"inscricaoEstadual,omitempty"`
	RazaoSocial       string          `json:"razaoSocial"`
	NomeFantasia      string          `json:"nomeFantasia,omitempty"`
	Certificado       string          `json:"certificado"`
	SimplesNacional   bool            `json:"simplesNacional"`
	RegimeTributario  int             `json:"regimeTributario"`
	Endereco          EnderecoEmpresa `json:"endereco"`
	NFe               *ModuloConfig   `json:"nfe,omitempty"`
	NFCe              *ModuloConfig   `json:"nfce,omitempty"`
}

type EnderecoEmpresa struct {
	CodigoPais      string `json:"codigoPais"`
	DescricaoPais   string `json:"descricaoPais"`
	Logradouro      string `json:"logradouro"`
	Numero          string `json:"numero"`
	Complemento     string `json:"complemento,omitempty"`
	Bairro          string `json:"bairro"`
	CodigoCidade    string `json:"codigoCidade"`
	DescricaoCidade string `json:"descricaoCidade"`
	Estado          string `json:"estado"`
	CEP             string `json:"cep"`
}

type ModuloConfig struct {
	Ativo  bool `json:"ativo"`
	Config struct {
		Producao bool `json:"producao"`
	} `json:"config"`
}

type WebhookPayload struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
}

// ---------------------------------------------------------------------------
// NF-e / NFC-e — payload de emissão (compartilhado pelos dois modelos)
// ---------------------------------------------------------------------------

type NFePayload struct {
	IDIntegracao       string             `json:"idIntegracao"`
	Presencial         bool               `json:"presencial"`
	ConsumidorFinal    bool               `json:"consumidorFinal"`
	Natureza           string             `json:"natureza"`
	Emitente           Pessoa             `json:"emitente"`
	Destinatario       *Destinatario      `json:"destinatario,omitempty"`
	Itens              []Item             `json:"itens"`
	Pagamentos         []Pagamento        `json:"pagamentos"`
	ResponsavelTecnico ResponsavelTecnico `json:"responsavelTecnico"`
}

type NFCePayload struct {
	IDIntegracao       string             `json:"idIntegracao"`
	Natureza           string             `json:"natureza"`
	Emitente           Pessoa             `json:"emitente"`
	Itens              []Item             `json:"itens"`
	Pagamentos         []Pagamento        `json:"pagamentos"`
	ResponsavelTecnico ResponsavelTecnico `json:"responsavelTecnico"`
}

type Pessoa struct {
	CpfCnpj string `json:"cpfCnpj"`
}

type Destinatario struct {
	CpfCnpj     string                `json:"cpfCnpj"`
	RazaoSocial string                `json:"razaoSocial"`
	Email       string                `json:"email,omitempty"`
	Endereco    *EnderecoDestinatario `json:"endereco,omitempty"`
}

type EnderecoDestinatario struct {
	Logradouro      string `json:"logradouro"`
	Numero          string `json:"numero"`
	Bairro          string `json:"bairro"`
	CodigoCidade    string `json:"codigoCidade"`
	DescricaoCidade string `json:"descricaoCidade"`
	Estado          string `json:"estado"`
	CEP             string `json:"cep"`
}

type Item struct {
	Codigo        string        `json:"codigo"`
	Descricao     string        `json:"descricao"`
	NCM           string        `json:"ncm"`
	CEST          string        `json:"cest,omitempty"`
	CFOP          string        `json:"cfop"`
	ValorUnitario ValorUnitario `json:"valorUnitario"`
	Valor         float64       `json:"valor"`
	Tributos      Tributos      `json:"tributos"`
}

type ValorUnitario struct {
	Comercial  float64 `json:"comercial"`
	Tributavel float64 `json:"tributavel"`
}

type Tributos struct {
	ICMS   ICMS   `json:"icms"`
	PIS    PIS    `json:"pis"`
	Cofins Cofins `json:"cofins"`
}

type ICMS struct {
	Origem      string          `json:"origem"`
	CST         string          `json:"cst,omitempty"`
	CSOSN       string          `json:"csosn,omitempty"`
	BaseCalculo BaseCalculoICMS `json:"baseCalculo"`
	Aliquota    float64         `json:"aliquota"`
	Valor       float64         `json:"valor"`
}

type BaseCalculoICMS struct {
	ModalidadeDeterminacao int     `json:"modalidadeDeterminacao"`
	Valor                  float64 `json:"valor"`
}

type PIS struct {
	CST         string  `json:"cst"`
	BaseCalculo float64 `json:"baseCalculo"`
	Aliquota    float64 `json:"aliquota"`
	Valor       float64 `json:"valor"`
}

type Cofins struct {
	CST         string  `json:"cst"`
	BaseCalculo float64 `json:"baseCalculo"`
	Aliquota    float64 `json:"aliquota"`
	Valor       float64 `json:"valor"`
}

type Pagamento struct {
	AVista bool    `json:"aVista"`
	Meio   string  `json:"meio"`
	Valor  float64 `json:"valor"`
}

type ResponsavelTecnico struct {
	CpfCnpj  string   `json:"cpfCnpj"`
	Nome     string   `json:"nome"`
	Email    string   `json:"email"`
	Telefone Telefone `json:"telefone"`
}

type Telefone struct {
	DDD    string `json:"ddd"`
	Numero string `json:"numero"`
}

// ---------------------------------------------------------------------------
// NF-e / NFC-e — respostas
// ---------------------------------------------------------------------------

type RegistrarNotaResponse []struct {
	ID           string `json:"id"`
	IDIntegracao string `json:"idIntegracao"`
	Status       string `json:"status"`
}

type ConsultaResumoResponse struct {
	Status         string `json:"status"`
	ChaveAcesso    string `json:"chaveAcesso"`
	Numero         string `json:"numero"`
	Serie          string `json:"serie"`
	Protocolo      string `json:"protocolo"`
	MotivoRejeicao string `json:"motivo,omitempty"`
}

type CancelarNotaRequest struct {
	Justificativa string `json:"justificativa"`
}

// WebhookNotification é o corpo recebido no callback configurado via
// POST /empresa/{cnpj}/webhook. Confirmado contra a documentação oficial
// (Central de Atendimento TecnoSpeed, artigo "Explicando e cadastrando
// Webhook/API Callback"). Só dispara em status final: CONCLUIDO, REJEITADO,
// CANCELADO ou DENEGADO — nunca em PROCESSANDO.
type WebhookNotification struct {
	ID              string  `json:"id"`
	IDIntegracao    string  `json:"idIntegracao"`
	Status          string  `json:"status"` // "CONCLUIDO" | "REJEITADO" | "CANCELADO" | "DENEGADO"
	Emitente        string  `json:"emitente"`
	Destinatario    string  `json:"destinatario"`
	Valor           float64 `json:"valor"`
	DataAutorizacao string  `json:"dataAutorizacao"`
	Numero          string  `json:"numero"`
	Serie           string  `json:"serie"`
	Chave           string  `json:"chave"`
	Protocolo       string  `json:"protocolo"`
	Mensagem        string  `json:"mensagem"` // motivo de autorização, rejeição ou cancelamento
	XML             string  `json:"xml"`
	PDF             string  `json:"pdf"`
}
