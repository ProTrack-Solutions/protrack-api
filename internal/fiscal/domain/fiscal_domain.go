package domain

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidTaxProfile    = errors.New("perfil tributário inválido")
	ErrTaxProfileNotFound   = errors.New("perfil tributário não encontrado")
	ErrTaxProfileNameExists = errors.New("já existe um perfil tributário com esse nome")
	ErrInvalidFiscalData    = errors.New("dados fiscais inválidos")
	ErrCategoryNotFound     = errors.New("categoria não encontrada")
	ErrProductNotFound      = errors.New("produto não encontrado")
	ErrCompanyNotFound      = errors.New("empresa não encontrada")
	ErrIssuerNotRegistered  = errors.New("a empresa ainda não foi cadastrada como emitente de NFC-e")
	ErrIssuerAlreadyExists  = errors.New("a empresa já está cadastrada como emitente")
	ErrFiscalNotConfigured  = errors.New("emissão fiscal não configurada no servidor; fale com o suporte")
	ErrAlreadyLive          = errors.New("a empresa já emite em produção")
	ErrNotReadyToEnable     = errors.New("termine a configuração fiscal antes de ligar a emissão")
)

// Environment da empresa: test = homologação (notas sem valor fiscal), live = produção.
const (
	EnvTest = "test"
	EnvLive = "live"
)

// CSOSN e CST aceitos na NFC-e pela API fiscal (venda a consumidor final, sem ST cobrada).
var (
	nfceCSOSN   = []string{"102", "103", "300", "400", "500", "900"}
	nfceCST     = []string{"00", "20", "40", "41", "60", "90"}
	pisNoValue  = []string{"04", "05", "06", "07", "08", "09"}
	pisWithRate = []string{"49", "99"}
	units       = []string{"UN", "KG", "G", "L", "ML"}

	reCFOP        = regexp.MustCompile(`^5\d{3}$`)
	reBenefitCode = regexp.MustCompile(`^([A-Z0-9]{8}|[A-Z0-9]{10})$`)
	reIBSCST      = regexp.MustCompile(`^\d{3}$`)
	reClassCode   = regexp.MustCompile(`^\d{6}$`)
	reNCM         = regexp.MustCompile(`^\d{8}$`)
	reCEST        = regexp.MustCompile(`^\d{7}$`)
	reDigits      = regexp.MustCompile(`\D`)
)

// OnlyDigits tira pontos, traços e barras ("1234.56.78" -> "12345678").
func OnlyDigits(s string) string { return reDigits.ReplaceAllString(s, "") }

// ---------- Perfil tributário ----------

type TaxProfileInput struct {
	Name              string   `json:"name"`
	IcmsCSOSN         string   `json:"icms_csosn"`
	IcmsCST           string   `json:"icms_cst"`
	IcmsRate          *float64 `json:"icms_rate"`
	IcmsBaseReduction *float64 `json:"icms_base_reduction"`
	IcmsFcpRate       *float64 `json:"icms_fcp_rate"`
	CFOP              string   `json:"cfop"`
	BenefitCode       string   `json:"benefit_code"`
	PisCST            string   `json:"pis_cst"`
	PisRate           *float64 `json:"pis_rate"`
	CofinsCST         string   `json:"cofins_cst"`
	CofinsRate        *float64 `json:"cofins_rate"`
	IbsCbsCST         string   `json:"ibs_cbs_cst"`
	IbsCbsClassCode   string   `json:"ibs_cbs_class_code"`
	IbsStateRate      *float64 `json:"ibs_state_rate"`
	IbsCityRate       *float64 `json:"ibs_city_rate"`
	CbsRate           *float64 `json:"cbs_rate"`
}

type TaxProfile struct {
	ID uuid.UUID `json:"id"`
	TaxProfileInput
	ProductsCount   int64     `json:"products_count"`
	CategoriesCount int64     `json:"categories_count"`
	IsDefault       bool      `json:"is_default"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidTaxProfile, fmt.Sprintf(format, args...))
}

func checkPercent(name string, v *float64) error {
	if v != nil && (*v < 0 || *v > 100) {
		return invalid("%s deve ficar entre 0 e 100", name)
	}
	return nil
}

// Normalize limpa e confere o perfil. taxRegime ("1".."4" ou vazio) decide se o
// ICMS vai por CSOSN (Simples/MEI) ou CST (Normal/excesso de sublimite).
func (in TaxProfileInput) Normalize(taxRegime string) (TaxProfileInput, error) {
	out := in
	out.Name = strings.TrimSpace(in.Name)
	out.IcmsCSOSN = strings.TrimSpace(in.IcmsCSOSN)
	out.IcmsCST = strings.TrimSpace(in.IcmsCST)
	out.CFOP = OnlyDigits(in.CFOP)
	out.BenefitCode = strings.ToUpper(strings.TrimSpace(in.BenefitCode))
	out.PisCST = strings.TrimSpace(in.PisCST)
	out.CofinsCST = strings.TrimSpace(in.CofinsCST)
	out.IbsCbsCST = strings.TrimSpace(in.IbsCbsCST)
	out.IbsCbsClassCode = OnlyDigits(in.IbsCbsClassCode)

	if l := len([]rune(out.Name)); l < 2 || l > 80 {
		return out, invalid("o nome deve ter entre 2 e 80 caracteres")
	}

	// ICMS
	switch {
	case out.IcmsCSOSN != "" && out.IcmsCST != "":
		return out, invalid("informe CSOSN ou CST, não os dois")
	case out.IcmsCSOSN == "" && out.IcmsCST == "":
		return out, invalid("informe o CSOSN (Simples Nacional/MEI) ou o CST do ICMS")
	}
	simples := taxRegime == "1" || taxRegime == "4"
	normal := taxRegime == "2" || taxRegime == "3"
	if out.IcmsCSOSN != "" {
		if normal {
			return out, invalid("a empresa não é do Simples Nacional: use CST, não CSOSN")
		}
		if !slices.Contains(nfceCSOSN, out.IcmsCSOSN) {
			return out, invalid("CSOSN %s não é aceito na NFC-e (use %s)", out.IcmsCSOSN, strings.Join(nfceCSOSN, ", "))
		}
		// Só o 900 tem ICMS próprio; FCP não existe no CSOSN
		if out.IcmsCSOSN != "900" {
			out.IcmsRate, out.IcmsBaseReduction = nil, nil
		}
		out.IcmsFcpRate = nil
	} else {
		if simples {
			return out, invalid("empresa do Simples Nacional/MEI usa CSOSN, não CST")
		}
		if !slices.Contains(nfceCST, out.IcmsCST) {
			return out, invalid("CST %s não é aceito na NFC-e (use %s)", out.IcmsCST, strings.Join(nfceCST, ", "))
		}
		switch out.IcmsCST {
		case "00":
			if out.IcmsRate == nil {
				return out, invalid("CST 00 exige a alíquota do ICMS")
			}
			out.IcmsBaseReduction = nil
		case "20":
			if out.IcmsRate == nil || out.IcmsBaseReduction == nil {
				return out, invalid("CST 20 exige a alíquota e o percentual de redução da base")
			}
		case "40", "41", "60":
			out.IcmsRate, out.IcmsBaseReduction, out.IcmsFcpRate = nil, nil, nil
		case "90":
			if out.IcmsRate == nil {
				out.IcmsBaseReduction, out.IcmsFcpRate = nil, nil
			}
		}
	}
	for name, v := range map[string]*float64{
		"a alíquota do ICMS": out.IcmsRate, "a redução da base": out.IcmsBaseReduction, "o FCP": out.IcmsFcpRate,
	} {
		if err := checkPercent(name, v); err != nil {
			return out, err
		}
	}

	if out.CFOP == "" {
		out.CFOP = "5102"
	}
	if !reCFOP.MatchString(out.CFOP) {
		return out, invalid("a NFC-e só aceita CFOP de venda dentro do estado (5xxx)")
	}
	if out.BenefitCode != "" && !reBenefitCode.MatchString(out.BenefitCode) {
		return out, invalid("código de benefício fiscal (cBenef) deve ter 8 ou 10 letras/números")
	}

	// PIS e COFINS
	for _, c := range []struct {
		name string
		cst  *string
		rate **float64
	}{{"PIS", &out.PisCST, &out.PisRate}, {"COFINS", &out.CofinsCST, &out.CofinsRate}} {
		if *c.cst == "" {
			*c.cst = "49"
		}
		switch {
		case slices.Contains(pisNoValue, *c.cst):
			*c.rate = nil
		case slices.Contains(pisWithRate, *c.cst):
			if err := checkPercent("a alíquota do "+c.name, *c.rate); err != nil {
				return out, err
			}
		default:
			return out, invalid("CST do %s %s não é aceito (use 04 a 09, 49 ou 99)", c.name, *c.cst)
		}
	}

	// IBS/CBS
	if !reIBSCST.MatchString(out.IbsCbsCST) {
		return out, invalid("CST do IBS/CBS deve ter 3 dígitos")
	}
	if !reClassCode.MatchString(out.IbsCbsClassCode) {
		return out, invalid("cClassTrib do IBS/CBS deve ter 6 dígitos")
	}
	rates := []*float64{out.IbsStateRate, out.IbsCityRate, out.CbsRate}
	filled := 0
	for _, r := range rates {
		if r != nil {
			filled++
		}
	}
	if filled != 0 && filled != 3 {
		return out, invalid("informe as três alíquotas do IBS/CBS (estadual, municipal e CBS) ou nenhuma")
	}
	for name, v := range map[string]*float64{
		"o IBS estadual": out.IbsStateRate, "o IBS municipal": out.IbsCityRate, "a CBS": out.CbsRate,
	} {
		if err := checkPercent(name, v); err != nil {
			return out, err
		}
	}
	return out, nil
}

// ---------- Tributação de produto ----------

type ProductFiscalInput struct {
	NCM          string     `json:"ncm"`
	CEST         string     `json:"cest"`
	Unit         string     `json:"unit"`
	Origin       int16      `json:"origin"`
	TaxProfileID *uuid.UUID `json:"tax_profile_id"`
}

type ProductFiscal struct {
	ProductID            uuid.UUID  `json:"product_id"`
	Name                 string     `json:"name"`
	ModelName            *string    `json:"model_name"`
	NCM                  *string    `json:"ncm"`
	CEST                 *string    `json:"cest"`
	Unit                 string     `json:"unit"`
	Origin               int16      `json:"origin"`
	TaxProfileID         *uuid.UUID `json:"tax_profile_id"`
	CategoryTaxProfileID *uuid.UUID `json:"category_tax_profile_id"`
}

// Normalize confere NCM (8 dígitos), CEST (7), unidade e origem (0 a 8).
func (in ProductFiscalInput) Normalize() (ProductFiscalInput, error) {
	out := in
	out.NCM = OnlyDigits(in.NCM)
	out.CEST = OnlyDigits(in.CEST)
	out.Unit = strings.ToUpper(strings.TrimSpace(in.Unit))
	if out.NCM != "" && !reNCM.MatchString(out.NCM) {
		return out, fmt.Errorf("%w: o NCM deve ter 8 dígitos", ErrInvalidFiscalData)
	}
	if out.CEST != "" && !reCEST.MatchString(out.CEST) {
		return out, fmt.Errorf("%w: o CEST deve ter 7 dígitos", ErrInvalidFiscalData)
	}
	if out.Unit != "" && !slices.Contains(units, out.Unit) {
		return out, fmt.Errorf("%w: unidade deve ser %s", ErrInvalidFiscalData, strings.Join(units, ", "))
	}
	if out.Origin < 0 || out.Origin > 8 {
		return out, fmt.Errorf("%w: origem da mercadoria vai de 0 a 8", ErrInvalidFiscalData)
	}
	return out, nil
}

type CategoryFiscal struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Color        *string    `json:"color"`
	TaxProfileID *uuid.UUID `json:"tax_profile_id"`
}

type SetCategoryTaxProfileInput struct {
	TaxProfileID *uuid.UUID `json:"tax_profile_id"`
}

// ---------- Configuração da empresa ----------

type FiscalSettings struct {
	Enabled             bool       `json:"enabled"`
	Environment         string     `json:"environment"`
	TaxRegime           *string    `json:"tax_regime"`
	DefaultTaxProfileID *uuid.UUID `json:"default_tax_profile_id"`
	IssuerRegistered    bool       `json:"issuer_registered"`
	CertificateExpires  *time.Time `json:"certificate_expires_at"`
	// O servidor tem chave da API fiscal para o ambiente atual
	ServerConfigured bool `json:"server_configured"`
	// Produtos ativos sem NCM válido (não saem na NFC-e)
	ProductsMissingNCM int64             `json:"products_missing_ncm"`
	Company            CompanyFiscalData `json:"company"`
}

// CompanyFiscalData pré-preenche o cadastro (vem do cadastro da empresa).
type CompanyFiscalData struct {
	Name              string `json:"name"`
	TradeName         string `json:"trade_name"`
	CNPJ              string `json:"cnpj"`
	StateRegistration string `json:"state_registration"`
	Phone             string `json:"phone"`
	Street            string `json:"street"`
	Number            string `json:"number"`
	Complement        string `json:"complement"`
	District          string `json:"district"`
	City              string `json:"city"`
	State             string `json:"state"`
	ZipCode           string `json:"zip_code"`
}

type UpdateSettingsInput struct {
	Enabled             bool       `json:"enabled"`
	DefaultTaxProfileID *uuid.UUID `json:"default_tax_profile_id"`
}

type FiscalProfileInput struct {
	StateRegistration string `json:"state_registration"`
	TaxRegime         string `json:"tax_regime"`
	Street            string `json:"street"`
	Number            string `json:"number"`
	Complement        string `json:"complement"`
	District          string `json:"district"`
	CityCode          string `json:"city_code"`
	CityName          string `json:"city_name"`
	ZipCode           string `json:"zip_code"`
	Phone             string `json:"phone"`
	CSCID             string `json:"csc_id"`
	CSCSecret         string `json:"csc_secret"`
}

var (
	reIE       = regexp.MustCompile(`^\d{2,14}$`)
	reCityCode = regexp.MustCompile(`^\d{7}$`)
	reZip      = regexp.MustCompile(`^\d{8}$`)
	rePhone    = regexp.MustCompile(`^\d{6,14}$`)
	reCSCID    = regexp.MustCompile(`^\d{1,6}$`)
)

func badProfile(msg string) error { return fmt.Errorf("%w: %s", ErrInvalidFiscalData, msg) }

// Normalize confere o perfil fiscal antes de mandar para a API fiscal.
func (in FiscalProfileInput) Normalize() (FiscalProfileInput, error) {
	out := in
	out.StateRegistration = OnlyDigits(in.StateRegistration)
	out.CityCode = OnlyDigits(in.CityCode)
	out.ZipCode = OnlyDigits(in.ZipCode)
	out.Phone = OnlyDigits(in.Phone)
	out.CSCID = OnlyDigits(in.CSCID)
	out.CSCSecret = strings.TrimSpace(in.CSCSecret)
	for _, f := range []*string{&out.Street, &out.Number, &out.Complement, &out.District, &out.CityName} {
		*f = strings.TrimSpace(*f)
	}
	switch {
	case !reIE.MatchString(out.StateRegistration):
		return out, badProfile("inscrição estadual deve ter de 2 a 14 dígitos")
	case !slices.Contains([]string{"1", "2", "3", "4"}, out.TaxRegime):
		return out, badProfile("escolha o regime tributário")
	case out.Street == "" || out.Number == "" || out.District == "" || out.CityName == "":
		return out, badProfile("preencha rua, número, bairro e cidade")
	case !reCityCode.MatchString(out.CityCode):
		return out, badProfile("código IBGE da cidade deve ter 7 dígitos")
	case !reZip.MatchString(out.ZipCode):
		return out, badProfile("CEP deve ter 8 dígitos")
	case out.Phone != "" && !rePhone.MatchString(out.Phone):
		return out, badProfile("telefone deve ter de 6 a 14 dígitos")
	case (out.CSCID == "") != (out.CSCSecret == ""):
		return out, badProfile("informe o ID e o código do CSC juntos")
	case out.CSCID != "" && !reCSCID.MatchString(out.CSCID):
		return out, badProfile("ID do CSC deve ter até 6 dígitos")
	case len(out.CSCSecret) > 64:
		return out, badProfile("código do CSC muito longo")
	}
	for _, f := range []string{out.Street, out.Number, out.Complement, out.District, out.CityName} {
		if len([]rune(f)) > 60 {
			return out, badProfile("endereço: cada campo vai até 60 caracteres")
		}
	}
	return out, nil
}

type FiscalProfile struct {
	FiscalProfileInput
	CSCConfigured bool `json:"csc_configured"`
}

type CertificateInfo struct {
	ExpiresAt    time.Time `json:"expires_at"`
	Expired      bool      `json:"expired"`
	DaysToExpire int       `json:"days_to_expire"`
}

type SefazStatus struct {
	Online     bool      `json:"online"`
	StatusCode string    `json:"status_code"`
	Reason     string    `json:"reason"`
	LatencyMs  int       `json:"latency_ms"`
	CheckedAt  time.Time `json:"checked_at"`
}

type GoLiveInput struct {
	CSCID     string `json:"csc_id"`
	CSCSecret string `json:"csc_secret"`
}
