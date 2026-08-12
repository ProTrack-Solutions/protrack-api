package domain

// CadastrarEmpresaRequest carrega os dados mínimos para registrar a empresa
// emitente no provedor fiscal.
type CadastrarEmpresaRequest struct {
	CpfCnpj           string
	RazaoSocial       string
	NomeFantasia      string
	InscricaoEstadual string
	SimplesNacional   bool
	Certificado       string // ID do certificado já cadastrado no provedor
	Endereco          Endereco
	Email             string
}

type Endereco struct {
	Logradouro   string
	Numero       string
	Complemento  string
	Bairro       string
	CodigoCidade string
	Cidade       string
	Estado       string
	CEP          string
}

// EmitirNotaRequest carrega os dados de uma nota a ser emitida, independente
// de ser NF-e ou NFC-e.
type EmitirNotaRequest struct {
	IDIntegracao    string // ID único gerado pelo ProTrack (idempotência)
	CpfCnpjEmitente string
	Itens           []ItemNota
	Pagamentos      []Pagamento
}

type ItemNota struct {
	Codigo        string
	Descricao     string
	NCM           string
	CEST          string
	CFOP          string
	CSOSN         string
	Quantidade    float64
	ValorUnitario float64
	Valor         float64
}

type Pagamento struct {
	Forma string
	Valor float64
}

// NotaStatus representa o status retornado pelo provedor ao consultar uma nota.
type NotaStatus struct {
	Status         string // "processando", "autorizado", "rejeitado", "cancelado"
	ChaveAcesso    string
	Numero         string
	Serie          string
	Protocolo      string
	XMLUrl         string
	DANFEUrl       string
	MotivoRejeicao string
}
