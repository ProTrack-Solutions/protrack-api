package client

import "time"

// Environment escolhe a chave da conta integradora: test = homologação, live = produção.
type Environment string

const (
	EnvTest Environment = "test"
	EnvLive Environment = "live"
)

// Valid diz se o ambiente é um dos dois conhecidos.
func (e Environment) Valid() bool { return e == EnvTest || e == EnvLive }

// SefazEnvironment é o tpAmb da SEFAZ: 1 produção, 2 homologação.
func (e Environment) SefazEnvironment() int {
	if e == EnvLive {
		return 1
	}
	return 2
}

// Situações de uma nota na API fiscal.
const (
	StatusPending     = "PENDING"
	StatusProcessing  = "PROCESSING"
	StatusAuthorized  = "AUTHORIZED"
	StatusRejected    = "REJECTED"
	StatusError       = "ERROR"
	StatusCancelled   = "CANCELLED"
	StatusContingency = "CONTINGENCY"
	StatusSuperseded  = "SUPERSEDED"
	StatusInutilized  = "INUTILIZED"
)

type Issuer struct {
	ID             string     `json:"id"`
	CNPJ           string     `json:"cnpj"`
	LegalName      string     `json:"legalName"`
	TradeName      *string    `json:"tradeName"`
	State          string     `json:"state"`
	Environment    int        `json:"environment"`
	CNPJVerifiedAt *time.Time `json:"cnpjVerifiedAt"`
	DeactivatedAt  *time.Time `json:"deactivatedAt"`
}

type CreateIssuerRequest struct {
	CNPJ      string `json:"cnpj"`
	LegalName string `json:"legalName"`
	TradeName string `json:"tradeName,omitempty"`
	State     string `json:"state,omitempty"`
}

// IssuerPair é o mesmo CNPJ nos dois ambientes (ids diferentes).
type IssuerPair struct {
	Test Issuer `json:"test"`
	Live Issuer `json:"live"`
}

type CertificateUploaded struct {
	SubjectCN    *string   `json:"subjectCN"`
	IssuerCN     *string   `json:"issuerCN"`
	NotBefore    time.Time `json:"notBefore"`
	NotAfter     time.Time `json:"notAfter"`
	SerialNumber string    `json:"serialNumber"`
	CNPJ         *string   `json:"cnpj"`
}

type CertificateStatus struct {
	ExpiresAt    time.Time `json:"expiresAt"`
	UploadedAt   time.Time `json:"uploadedAt"`
	Expired      bool      `json:"expired"`
	DaysToExpire int       `json:"daysToExpire"`
}

type Address struct {
	Street     string  `json:"street"`
	Number     string  `json:"number"`
	Complement *string `json:"complement,omitempty"`
	District   string  `json:"district"`
	CityCode   string  `json:"cityCode"`
	CityName   string  `json:"cityName"`
	ZipCode    string  `json:"zipCode"`
	Phone      *string `json:"phone,omitempty"`
}

type CSC struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

// FiscalProfileRequest: CSC nil mantém o atual.
type FiscalProfileRequest struct {
	StateRegistration string  `json:"stateRegistration"`
	TaxRegime         string  `json:"taxRegime"`
	Address           Address `json:"address"`
	CSC               *CSC    `json:"csc,omitempty"`
}

type FiscalProfile struct {
	IssuerID          string  `json:"issuerId"`
	StateRegistration string  `json:"stateRegistration"`
	TaxRegime         string  `json:"taxRegime"`
	Address           Address `json:"address"`
	QRCodeVersion     int     `json:"qrCodeVersion"`
	CSC               struct {
		Configured bool    `json:"configured"`
		ID         *string `json:"id"`
	} `json:"csc"`
}

type SefazStatus struct {
	State       string    `json:"state"`
	Environment int       `json:"environment"`
	Model       int       `json:"model"`
	Online      bool      `json:"online"`
	StatusCode  string    `json:"statusCode"`
	Reason      string    `json:"reason"`
	LatencyMs   int       `json:"latencyMs"`
	CheckedAt   time.Time `json:"checkedAt"`
}

type CopyFromHomologationResult struct {
	Certificate *struct {
		ExpiresAt time.Time `json:"expiresAt"`
	} `json:"certificate"`
	FiscalProfile *FiscalProfile `json:"fiscalProfile"`
}

// ---- NFC-e ----

type NfceRequest struct {
	Items          []NfceItem     `json:"items"`
	Payments       []NfcePayment  `json:"payments"`
	Customer       *NfceCustomer  `json:"customer,omitempty"`
	AdditionalInfo string         `json:"additionalInfo,omitempty"`
	Contingency    *ContingencyIn `json:"contingency,omitempty"`
}

type ContingencyIn struct {
	Reason string `json:"reason,omitempty"`
}

type NfceCustomer struct {
	CPF  string `json:"cpf"`
	Name string `json:"name,omitempty"`
}

type NfceItem struct {
	Code        string  `json:"code"`
	GTIN        string  `json:"gtin,omitempty"`
	Description string  `json:"description"`
	NCM         string  `json:"ncm"`
	CEST        string  `json:"cest,omitempty"`
	BenefitCode string  `json:"benefitCode,omitempty"`
	Unit        string  `json:"unit"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	Discount    float64 `json:"discount,omitempty"`
	CFOP        string  `json:"cfop"`
	Tax         NfceTax `json:"tax"`
}

type NfceTax struct {
	ICMS   ICMS         `json:"icms"`
	PIS    Contribution `json:"pis"`
	COFINS Contribution `json:"cofins"`
	IBSCBS IBSCBS       `json:"ibsCbs"`
}

// ICMS da NFC-e: CSOSN (CRT 1/4) ou CST (CRT 2/3), nunca os dois. CST 90 e
// CSOSN 900 mandam o ICMS próprio em Own; os demais CSTs tributados, nos campos soltos.
type ICMS struct {
	CSOSN         string   `json:"csosn,omitempty"`
	CST           string   `json:"cst,omitempty"`
	Origin        string   `json:"origin"`
	Rate          *float64 `json:"rate,omitempty"`
	BaseReduction *float64 `json:"baseReduction,omitempty"`
	FCPRate       *float64 `json:"fcpRate,omitempty"`
	Own           *OwnICMS `json:"own,omitempty"`
}

type OwnICMS struct {
	Rate          float64  `json:"rate"`
	BaseReduction *float64 `json:"baseReduction,omitempty"`
	FCPRate       *float64 `json:"fcpRate,omitempty"`
}

// Contribution é PIS ou COFINS. CSTs 49/99 levam base, alíquota e valor.
type Contribution struct {
	CST       string   `json:"cst"`
	BaseValue *float64 `json:"baseValue,omitempty"`
	Rate      *float64 `json:"rate,omitempty"`
	Value     *float64 `json:"value,omitempty"`
}

type IBSCBS struct {
	CST       string       `json:"cst"`
	ClassCode string       `json:"classCode"`
	Rates     *IBSCBSRates `json:"rates,omitempty"`
}

type IBSCBSRates struct {
	IBSState float64 `json:"ibsState"`
	IBSCity  float64 `json:"ibsCity"`
	CBS      float64 `json:"cbs"`
}

type NfcePayment struct {
	Method          string  `json:"method"`
	Amount          float64 `json:"amount"`
	IntegrationType string  `json:"integrationType,omitempty"`
}

// ---- NF-e (modelo 55) ----

type NfeRequest struct {
	OperationNature string        `json:"operationNature"`
	FinalConsumer   bool          `json:"finalConsumer"`
	Presence        string        `json:"presence"`
	Customer        NfeCustomer   `json:"customer"`
	Items           []NfeItem     `json:"items"`
	Transport       *NfeTransport `json:"transport,omitempty"`
	Payments        []NfePayment  `json:"payments"`
	AdditionalInfo  string        `json:"additionalInfo,omitempty"`
}

// NfeCustomer: CNPJ ou CPF (um dos dois). StateRegistrationIndicator é o indIEDest:
// 1 contribuinte (IE obrigatória), 2 isento, 9 não contribuinte.
type NfeCustomer struct {
	CNPJ                       string     `json:"cnpj,omitempty"`
	CPF                        string     `json:"cpf,omitempty"`
	Name                       string     `json:"name"`
	StateRegistrationIndicator string     `json:"stateRegistrationIndicator"`
	StateRegistration          string     `json:"stateRegistration,omitempty"`
	Email                      string     `json:"email,omitempty"`
	Address                    NfeAddress `json:"address"`
}

type NfeAddress struct {
	Street     string `json:"street"`
	Number     string `json:"number"`
	Complement string `json:"complement,omitempty"`
	District   string `json:"district"`
	CityCode   string `json:"cityCode"`
	CityName   string `json:"cityName"`
	State      string `json:"state"`
	ZipCode    string `json:"zipCode,omitempty"`
	Phone      string `json:"phone,omitempty"`
}

type NfeItem struct {
	Code           string   `json:"code"`
	GTIN           string   `json:"gtin,omitempty"`
	Description    string   `json:"description"`
	NCM            string   `json:"ncm"`
	CEST           string   `json:"cest,omitempty"`
	BenefitCode    string   `json:"benefitCode,omitempty"`
	Unit           string   `json:"unit"`
	Quantity       float64  `json:"quantity"`
	UnitPrice      float64  `json:"unitPrice"`
	Discount       float64  `json:"discount,omitempty"`
	CFOP           string   `json:"cfop"`
	Freight        *float64 `json:"freight,omitempty"`
	Insurance      *float64 `json:"insurance,omitempty"`
	OtherExpenses  *float64 `json:"otherExpenses,omitempty"`
	AdditionalInfo string   `json:"additionalInfo,omitempty"`
	Tax            NfeTax   `json:"tax"`
}

type NfeTax struct {
	ICMS   ICMS         `json:"icms"`
	PIS    Contribution `json:"pis"`
	COFINS Contribution `json:"cofins"`
	DIFAL  *DIFAL       `json:"difal,omitempty"`
	IBSCBS IBSCBS       `json:"ibsCbs"`
}

// DIFAL: venda interestadual a consumidor final não contribuinte (a API calcula os valores).
type DIFAL struct {
	DestinationRate float64  `json:"destinationRate"`
	InterstateRate  int      `json:"interstateRate"`
	FCPRate         *float64 `json:"fcpRate,omitempty"`
	Method          string   `json:"method,omitempty"`
}

// NfeTransport: Mode é o modFrete (0 emitente, 1 destinatário, 2 terceiros,
// 3 próprio remetente, 4 próprio destinatário, 9 sem frete).
type NfeTransport struct {
	Mode    string      `json:"mode"`
	Carrier *NfeCarrier `json:"carrier,omitempty"`
	Volumes []NfeVolume `json:"volumes,omitempty"`
}

type NfeCarrier struct {
	CNPJ              string `json:"cnpj,omitempty"`
	CPF               string `json:"cpf,omitempty"`
	Name              string `json:"name"`
	StateRegistration string `json:"stateRegistration,omitempty"`
	Address           string `json:"address,omitempty"`
	CityName          string `json:"cityName,omitempty"`
	State             string `json:"state,omitempty"`
}

type NfeVolume struct {
	Quantity    int      `json:"quantity"`
	Species     string   `json:"species,omitempty"`
	Brand       string   `json:"brand,omitempty"`
	NetWeight   *float64 `json:"netWeight,omitempty"`
	GrossWeight *float64 `json:"grossWeight,omitempty"`
}

// NfePayment: 90 = sem pagamento (valor 0); 99 = outros (descrição obrigatória).
type NfePayment struct {
	Method      string  `json:"method"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description,omitempty"`
}

type NfeTotalsRequest struct {
	Items         []NfeItem `json:"items"`
	FinalConsumer bool      `json:"finalConsumer"`
}

// NfeTotals: os totais como a emissão calcula. Total é o vNF (o que os pagamentos cobrem).
type NfeTotals struct {
	Products      float64 `json:"products"`
	Discount      float64 `json:"discount"`
	Freight       float64 `json:"freight"`
	Insurance     float64 `json:"insurance"`
	OtherExpenses float64 `json:"otherExpenses"`
	ICMSBase      float64 `json:"icmsBase"`
	ICMS          float64 `json:"icms"`
	FCP           float64 `json:"fcp"`
	STBase        float64 `json:"stBase"`
	ST            float64 `json:"st"`
	FCPST         float64 `json:"fcpSt"`
	IPI           float64 `json:"ipi"`
	PIS           float64 `json:"pis"`
	COFINS        float64 `json:"cofins"`
	DIFAL         *struct {
		ICMS float64 `json:"icms"`
		FCP  float64 `json:"fcp"`
	} `json:"difal"`
	IBSCBS struct {
		Base     float64 `json:"base"`
		IBSState float64 `json:"ibsState"`
		IBSCity  float64 `json:"ibsCity"`
		CBS      float64 `json:"cbs"`
	} `json:"ibsCbs"`
	Total           float64 `json:"total"`
	TotalWithIBSCBS float64 `json:"totalWithIbsCbs"`
}

// Document é a nota como a API fiscal devolve (emissão, consulta e webhooks).
type Document struct {
	ID           string  `json:"id"`
	Model        int     `json:"model"`
	Number       int     `json:"number"`
	AccessKey    *string `json:"accessKey"`
	Status       string  `json:"status"`
	Protocol     *string `json:"protocol"`
	StatusCode   *string `json:"statusCode"`
	RejectReason *string `json:"rejectReason"`
	Environment  int     `json:"environment"`
	EmissionType int     `json:"emissionType"`
	Contingency  *struct {
		EnteredAt time.Time `json:"enteredAt"`
		Reason    string    `json:"reason"`
	} `json:"contingency"`
	ReplacedByID *string    `json:"replacedById"`
	AuthorizedAt *time.Time `json:"authorizedAt"`
	Replayed     bool       `json:"replayed,omitempty"`
}

type FiscalEvent struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Sequence     int     `json:"sequence"`
	Status       string  `json:"status"`
	Protocol     *string `json:"protocol"`
	StatusCode   *string `json:"statusCode"`
	RejectReason *string `json:"rejectReason"`
	Reason       *string `json:"reason"`
}

type CancellationResult struct {
	Event          FiscalEvent `json:"event"`
	DocumentStatus string      `json:"documentStatus"`
	Replayed       bool        `json:"replayed"`
}

// WebhookPayload é o corpo dos webhooks da API fiscal.
type WebhookPayload struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	Data      struct {
		Document     *Document    `json:"document,omitempty"`
		Cancellation *FiscalEvent `json:"cancellation,omitempty"`
	} `json:"data"`
}
