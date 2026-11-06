package domain

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/client"
	"github.com/google/uuid"
)

// Emissão manual: NF-e (55) ou NFC-e (65) avulsa, sem venda e sem estoque. Os itens
// vêm do catálogo (NCM, unidade, origem e perfil tributário do cadastro) e a tela
// pode ajustar descrição, quantidade, preço, desconto, CFOP e perfil.

var ErrInvalidManualDocument = errors.New("nota manual inválida")

const (
	ModelNfe  = 55
	ModelNfce = 65

	DefaultOperationNature = "Venda de mercadoria"
)

var ufs = []string{
	"AC", "AL", "AM", "AP", "BA", "CE", "DF", "ES", "GO", "MA", "MG", "MS", "MT", "PA",
	"PB", "PE", "PI", "PR", "RJ", "RN", "RO", "RR", "RS", "SC", "SE", "SP", "TO",
}

type ManualDocumentInput struct {
	// 55 = NF-e, 65 = NFC-e
	Model int `json:"model"`
	// Gerada pela tela ao abrir o formulário: o mesmo envio (duplo clique) não vira duas notas
	IdempotencyKey string `json:"idempotency_key"`
	// natOp (NF-e). Vazio = "Venda de mercadoria"
	OperationNature string `json:"operation_nature"`
	// indFinal (NF-e). Vazio = consumidor final quando o destinatário não é contribuinte
	FinalConsumer *bool `json:"final_consumer"`
	// CFOP dos itens sem CFOP próprio (ex.: 5910 bonificação). Vazio = CFOP do perfil.
	// Na NF-e para outro estado o 5 vira 6 sozinho.
	DefaultCFOP string `json:"default_cfop"`
	// indPres (NF-e): 0 não se aplica, 1 presencial, 2 internet, 3 teleatendimento,
	// 4 entrega em domicílio, 5 presencial fora do estabelecimento, 9 outros
	Presence       string            `json:"presence"`
	Recipient      *ManualRecipient  `json:"recipient"`
	Items          []ManualItemInput `json:"items"`
	Payments       []ManualPayment   `json:"payments"`
	Transport      *ManualTransport  `json:"transport"`
	AdditionalInfo string            `json:"additional_info"`
}

// ManualRecipient: na NF-e é obrigatório e completo; na NFC-e só o CPF (opcional).
type ManualRecipient struct {
	// CPF ou CNPJ (só dígitos ou com máscara)
	Document string `json:"document"`
	Name     string `json:"name"`
	// indIEDest: 1 contribuinte (IE obrigatória), 2 isento, 9 não contribuinte
	IEIndicator       string        `json:"ie_indicator"`
	StateRegistration string        `json:"state_registration"`
	Email             string        `json:"email"`
	Address           ManualAddress `json:"address"`
}

type ManualAddress struct {
	Street     string `json:"street"`
	Number     string `json:"number"`
	Complement string `json:"complement"`
	District   string `json:"district"`
	// Código IBGE do município (7 dígitos)
	CityCode string `json:"city_code"`
	CityName string `json:"city_name"`
	State    string `json:"state"`
	ZipCode  string `json:"zip_code"`
	Phone    string `json:"phone"`
}

type ManualItemInput struct {
	ProductID uuid.UUID `json:"product_id"`
	// Vazio = nome do produto
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Discount    float64 `json:"discount"`
	// Vazio = CFOP do perfil (5xxx; vira 6xxx na NF-e para outro estado)
	CFOP string `json:"cfop"`
	// Vazio = perfil do produto -> categoria -> padrão da empresa
	TaxProfileID   *uuid.UUID   `json:"tax_profile_id"`
	Freight        *float64     `json:"freight"`
	Insurance      *float64     `json:"insurance"`
	OtherExpenses  *float64     `json:"other_expenses"`
	AdditionalInfo string       `json:"additional_info"`
	Difal          *ManualDifal `json:"difal"`
}

// ManualDifal: venda interestadual a consumidor final não contribuinte (NF-e).
type ManualDifal struct {
	DestinationRate float64  `json:"destination_rate"`
	InterstateRate  int      `json:"interstate_rate"`
	FCPRate         *float64 `json:"fcp_rate"`
	// single (padrão) ou double (base dupla)
	Method string `json:"method"`
}

// ManualPayment: Method é o tPag da SEFAZ (01 dinheiro, 03 crédito, 04 débito,
// 05 crédito loja, 15 boleto, 17 Pix, 18 transferência, 90 sem pagamento, 99 outros).
type ManualPayment struct {
	Method      string  `json:"method"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

type ManualTransport struct {
	// modFrete: 0 emitente, 1 destinatário, 2 terceiros, 3 próprio remetente, 4 próprio destinatário, 9 sem frete
	Mode    string         `json:"mode"`
	Carrier *ManualCarrier `json:"carrier"`
	Volumes []ManualVolume `json:"volumes"`
}

type ManualCarrier struct {
	Document          string `json:"document"`
	Name              string `json:"name"`
	StateRegistration string `json:"state_registration"`
	Address           string `json:"address"`
	CityName          string `json:"city_name"`
	State             string `json:"state"`
}

type ManualVolume struct {
	Quantity    int      `json:"quantity"`
	Species     string   `json:"species"`
	Brand       string   `json:"brand"`
	NetWeight   *float64 `json:"net_weight"`
	GrossWeight *float64 `json:"gross_weight"`
}

// ManualProduct é o produto do catálogo com o perfil já resolvido.
type ManualProduct struct {
	ID      uuid.UUID
	Name    string
	Barcode string
	NCM     string
	CEST    string
	Unit    string
	Origin  int16
	Profile *TaxProfile
}

// ManualBuild é o pedido pronto para a API fiscal e o que a nota guarda no ERP.
type ManualBuild struct {
	Model             int
	Nfce              *client.NfceRequest
	Nfe               *client.NfeRequest
	OperationNature   string
	RecipientName     string
	RecipientDocument string
	// Soma líquida dos itens (+ frete, seguro e outras despesas na NF-e). O vNF da
	// NF-e (com IPI/ST/DIFAL) vem do cálculo da API fiscal.
	ItemsTotal float64
	// Soma dos pagamentos e se a nota é "sem pagamento" (90)
	Paid      float64
	NoPayment bool
}

// Request é o corpo do pedido para a API fiscal.
func (b ManualBuild) Request() any {
	if b.Nfe != nil {
		return *b.Nfe
	}
	return *b.Nfce
}

// ValidCNPJ confere os dígitos verificadores.
func ValidCNPJ(cnpj string) bool {
	if len(cnpj) != 14 || strings.Count(cnpj, cnpj[:1]) == 14 {
		return false
	}
	weights := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	for _, n := range []int{12, 13} {
		sum := 0
		for i := 0; i < n; i++ {
			sum += int(cnpj[i]-'0') * weights[len(weights)-n+i]
		}
		d := sum % 11
		if d < 2 {
			d = 0
		} else {
			d = 11 - d
		}
		if d != int(cnpj[n]-'0') {
			return false
		}
	}
	return true
}

func round(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}

func optMoney(v *float64) *float64 {
	if v == nil || *v <= 0 {
		return nil
	}
	r := round(*v, 2)
	return &r
}

func clean(s string) string { return strings.Join(strings.Fields(s), " ") }

// BuildManual monta o pedido da nota manual. issuerUF é a UF do emitente (decide
// se a NF-e é interestadual). Devolve todos os problemas de uma vez.
func BuildManual(in ManualDocumentInput, products map[uuid.UUID]ManualProduct, issuerUF, taxRegime string) (ManualBuild, []string) {
	var problems []string
	add := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	out := ManualBuild{Model: in.Model}

	if in.Model != ModelNfe && in.Model != ModelNfce {
		return out, []string{"modelo da nota inválido (55 = NF-e, 65 = NFC-e)"}
	}
	nfe := in.Model == ModelNfe
	maxItems := 990
	if len(in.Items) == 0 {
		add("a nota não tem itens")
	} else if len(in.Items) > maxItems {
		add("a nota aceita no máximo %d itens", maxItems)
	}

	// Destinatário
	var recipientUF string
	var nfeCustomer client.NfeCustomer
	var nfceCustomer *client.NfceCustomer
	if r := in.Recipient; r != nil {
		doc := OnlyDigits(r.Document)
		name := clean(r.Name)
		out.RecipientName = truncateRunes(name, 60)
		out.RecipientDocument = doc
		switch {
		case doc == "" && nfe:
			add("informe o CPF ou CNPJ do destinatário")
		case doc == "":
		case len(doc) == 11 && !ValidCPF(doc):
			add("o CPF do destinatário é inválido")
		case len(doc) == 14 && !ValidCNPJ(doc):
			add("o CNPJ do destinatário é inválido")
		case len(doc) != 11 && len(doc) != 14:
			add("o documento do destinatário deve ser um CPF (11 dígitos) ou CNPJ (14 dígitos)")
		case !nfe && len(doc) == 14:
			add("a NFC-e só aceita CPF do consumidor; para CNPJ emita uma NF-e")
		}
		if nfe {
			if len([]rune(name)) < 2 {
				add("informe o nome do destinatário")
			}
			ind := strings.TrimSpace(r.IEIndicator)
			ie := OnlyDigits(r.StateRegistration)
			if strings.EqualFold(strings.TrimSpace(r.StateRegistration), "ISENTO") {
				ie = ""
			}
			switch ind {
			case "1":
				if len(ie) < 2 || len(ie) > 14 {
					add("destinatário contribuinte do ICMS precisa da inscrição estadual")
				}
			case "2":
				ie = ""
			case "9":
			default:
				add("informe se o destinatário é contribuinte do ICMS (1 contribuinte, 2 isento, 9 não contribuinte)")
			}
			a := r.Address
			recipientUF = strings.ToUpper(strings.TrimSpace(a.State))
			addr := client.NfeAddress{
				Street:     truncateRunes(clean(a.Street), 60),
				Number:     truncateRunes(clean(a.Number), 60),
				Complement: truncateRunes(clean(a.Complement), 60),
				District:   truncateRunes(clean(a.District), 60),
				CityCode:   OnlyDigits(a.CityCode),
				CityName:   truncateRunes(clean(a.CityName), 60),
				State:      recipientUF,
				ZipCode:    OnlyDigits(a.ZipCode),
				Phone:      OnlyDigits(a.Phone),
			}
			if len([]rune(addr.Street)) < 2 || addr.Number == "" || len([]rune(addr.District)) < 2 {
				add("o endereço do destinatário precisa de logradouro, número e bairro")
			}
			if len(addr.CityCode) != 7 || len([]rune(addr.CityName)) < 2 {
				add("informe a cidade do destinatário com o código IBGE (7 dígitos); preencha pelo CEP")
			}
			if !slices.Contains(ufs, recipientUF) {
				add("UF do destinatário inválida")
			}
			if addr.ZipCode != "" && len(addr.ZipCode) != 8 {
				add("o CEP do destinatário deve ter 8 dígitos")
			}
			if addr.Phone != "" && (len(addr.Phone) < 6 || len(addr.Phone) > 14) {
				addr.Phone = ""
			}
			nfeCustomer = client.NfeCustomer{
				Name:                       out.RecipientName,
				StateRegistrationIndicator: ind,
				StateRegistration:          ie,
				Email:                      truncateRunes(strings.TrimSpace(r.Email), 60),
				Address:                    addr,
			}
			if len(doc) == 14 {
				nfeCustomer.CNPJ = doc
			} else {
				nfeCustomer.CPF = doc
			}
		} else if len(doc) == 11 {
			nfceCustomer = &client.NfceCustomer{CPF: doc, Name: out.RecipientName}
		}
	} else if nfe {
		add("a NF-e precisa do destinatário")
	}

	issuerUF = strings.ToUpper(strings.TrimSpace(issuerUF))
	interstate := nfe && recipientUF != "" && issuerUF != "" && recipientUF != issuerUF
	finalConsumer := !nfe || nfeCustomer.StateRegistrationIndicator == "9"
	if nfe && in.FinalConsumer != nil {
		finalConsumer = *in.FinalConsumer
	}

	// Itens
	simples := taxRegime == "1" || taxRegime == "4"
	var totalCents int64
	var nfeItems []client.NfeItem
	var nfceItems []client.NfceItem
	for i, it := range in.Items {
		n := i + 1
		p, ok := products[it.ProductID]
		if !ok {
			add("item %d: produto não encontrado", n)
			continue
		}
		desc := truncateRunes(clean(it.Description), 120)
		if desc == "" {
			desc = truncateRunes(p.Name, 120)
		}
		label := fmt.Sprintf("item %d (%s)", n, desc)
		qty := round(it.Quantity, 4)
		price := round(it.UnitPrice, 4)
		if qty <= 0 {
			add("%s: a quantidade deve ser maior que zero", label)
		}
		if price < 0 {
			add("%s: o preço não pode ser negativo", label)
		}
		gross := int64(math.Round(qty * price * 100))
		disc := cents(it.Discount)
		if disc < 0 || disc > gross {
			add("%s: o desconto deve ficar entre zero e o valor do item", label)
			disc = 0
		}
		net := gross - disc
		ncm := OnlyDigits(p.NCM)
		if len(ncm) != 8 {
			add("%s: o produto está sem NCM", label)
		}
		if p.Profile == nil {
			add("%s: sem perfil tributário (escolha um no item ou defina o padrão em Configurações → Fiscal)", label)
			continue
		}
		prof := *p.Profile
		if taxRegime != "" && (prof.IcmsCSOSN != "") != simples {
			add("%s: o perfil \"%s\" não combina com o regime tributário da empresa (Simples usa CSOSN; Normal usa CST)", label, prof.Name)
		}

		cfop := OnlyDigits(it.CFOP)
		if cfop == "" {
			cfop = OnlyDigits(in.DefaultCFOP)
			if cfop == "" {
				cfop = prof.CFOP
			}
			switch {
			case interstate && len(cfop) == 4 && cfop[0] == '5':
				cfop = "6" + cfop[1:]
			case !interstate && len(cfop) == 4 && cfop[0] == '6':
				cfop = "5" + cfop[1:]
			}
		}
		switch {
		case len(cfop) != 4:
			add("%s: CFOP inválido", label)
		case !nfe && cfop[0] != '5':
			add("%s: a NFC-e só aceita CFOP de saída dentro do estado (5xxx)", label)
		case nfe && interstate && cfop[0] != '6':
			add("%s: destinatário de outro estado usa CFOP 6xxx (informado %s)", label, cfop)
		case nfe && !interstate && recipientUF != "" && cfop[0] != '5':
			add("%s: destinatário do mesmo estado usa CFOP 5xxx (informado %s)", label, cfop)
		}

		code := strings.TrimSpace(p.Barcode)
		if code == "" {
			code = strings.ToUpper(strings.ReplaceAll(p.ID.String(), "-", "")[:12])
		}
		gtin := ""
		if ValidGTIN(code) {
			gtin = code
		}
		unit := p.Unit
		if unit == "" {
			unit = "UN"
		}
		var ibs client.IBSCBS
		ibs = client.IBSCBS{CST: prof.IbsCbsCST, ClassCode: prof.IbsCbsClassCode}
		if prof.IbsStateRate != nil && prof.IbsCityRate != nil && prof.CbsRate != nil {
			ibs.Rates = &client.IBSCBSRates{IBSState: *prof.IbsStateRate, IBSCity: *prof.IbsCityRate, CBS: *prof.CbsRate}
		}
		totalCents += net

		if !nfe {
			nfceItems = append(nfceItems, client.NfceItem{
				Code: truncateRunes(code, 60), GTIN: gtin, Description: desc, NCM: ncm,
				CEST: OnlyDigits(p.CEST), BenefitCode: prof.BenefitCode, Unit: unit,
				Quantity: qty, UnitPrice: price, Discount: reais(disc), CFOP: cfop,
				Tax: client.NfceTax{
					ICMS:   icmsFor(prof, p.Origin),
					PIS:    contribution(prof.PisCST, prof.PisRate, net),
					COFINS: contribution(prof.CofinsCST, prof.CofinsRate, net),
					IBSCBS: ibs,
				},
			})
			continue
		}

		item := client.NfeItem{
			Code: truncateRunes(code, 60), GTIN: gtin, Description: desc, NCM: ncm,
			CEST: OnlyDigits(p.CEST), BenefitCode: prof.BenefitCode, Unit: unit,
			Quantity: qty, UnitPrice: price, Discount: reais(disc), CFOP: cfop,
			Freight: optMoney(it.Freight), Insurance: optMoney(it.Insurance), OtherExpenses: optMoney(it.OtherExpenses),
			AdditionalInfo: truncateRunes(clean(it.AdditionalInfo), 500),
			Tax: client.NfeTax{
				ICMS:   icmsFor(prof, p.Origin),
				PIS:    contribution(prof.PisCST, prof.PisRate, net),
				COFINS: contribution(prof.CofinsCST, prof.CofinsRate, net),
				IBSCBS: ibs,
			},
		}
		for _, extra := range []*float64{item.Freight, item.Insurance, item.OtherExpenses} {
			if extra != nil {
				totalCents += cents(*extra)
			}
		}
		if d := it.Difal; d != nil {
			if !interstate || !finalConsumer {
				add("%s: DIFAL só se aplica a venda para consumidor final de outro estado", label)
			}
			if d.InterstateRate != 4 && d.InterstateRate != 7 && d.InterstateRate != 12 {
				add("%s: alíquota interestadual do DIFAL deve ser 4, 7 ou 12", label)
			}
			method := d.Method
			if method != "double" {
				method = "single"
			}
			item.Tax.DIFAL = &client.DIFAL{
				DestinationRate: round(d.DestinationRate, 4), InterstateRate: d.InterstateRate,
				FCPRate: d.FCPRate, Method: method,
			}
		}
		nfeItems = append(nfeItems, item)
	}
	out.ItemsTotal = reais(totalCents)

	// Pagamentos
	var paid int64
	var nfePayments []client.NfePayment
	var nfcePayments []client.NfcePayment
	for i, pay := range in.Payments {
		method := OnlyDigits(pay.Method)
		amount := round(pay.Amount, 2)
		desc := truncateRunes(clean(pay.Description), 60)
		switch {
		case len(method) != 2:
			add("pagamento %d: forma de pagamento inválida", i+1)
			continue
		case amount < 0:
			add("pagamento %d: valor negativo", i+1)
			continue
		case method == "90" && amount != 0:
			add("pagamento %d: \"sem pagamento\" deve ter valor zero", i+1)
		case method == "99" && len([]rune(desc)) < 2:
			add("pagamento %d: descreva a forma de pagamento \"Outros\"", i+1)
		case !nfe && (method == "90" || method == "99"):
			add("a NFC-e não aceita \"sem pagamento\" nem \"outros\"; use dinheiro, cartão, Pix, transferência ou crediário")
		}
		paid += cents(amount)
		if nfe {
			nfePayments = append(nfePayments, client.NfePayment{Method: method, Amount: amount, Description: desc})
		} else {
			nfcePayments = append(nfcePayments, client.NfcePayment{Method: method, Amount: amount})
		}
	}
	if len(in.Payments) > 1 && slices.ContainsFunc(in.Payments, func(p ManualPayment) bool { return OnlyDigits(p.Method) == "90" }) {
		add("\"sem pagamento\" deve ser a única forma de pagamento da nota")
	}
	out.Paid = reais(paid)
	out.NoPayment = len(in.Payments) == 1 && OnlyDigits(in.Payments[0].Method) == "90"
	if len(in.Payments) == 0 {
		add("informe o pagamento (na NF-e sem cobrança, use \"90 - sem pagamento\")")
	} else if !nfe && len(problems) == 0 && paid < totalCents {
		add("os pagamentos (R$ %.2f) não cobrem o total da nota (R$ %.2f)", reais(paid), reais(totalCents))
	}

	if !nfe {
		if in.Transport != nil && in.Transport.Mode != "" && in.Transport.Mode != "9" {
			add("a NFC-e não tem transporte; para entrega com frete emita uma NF-e")
		}
		out.Nfce = &client.NfceRequest{
			Items: nfceItems, Payments: nfcePayments, Customer: nfceCustomer,
			AdditionalInfo: truncateRunes(strings.TrimSpace(in.AdditionalInfo), 2000),
		}
		return out, problems
	}

	nature := truncateRunes(clean(in.OperationNature), 60)
	if nature == "" {
		nature = DefaultOperationNature
	}
	out.OperationNature = nature
	presence := strings.TrimSpace(in.Presence)
	if presence == "" {
		presence = "1"
	}
	if !slices.Contains([]string{"0", "1", "2", "3", "4", "5", "9"}, presence) {
		add("indicador de presença inválido")
	}
	if nfeCustomer.StateRegistrationIndicator == "9" && !finalConsumer {
		add("destinatário não contribuinte do ICMS é sempre consumidor final")
	}

	req := client.NfeRequest{
		OperationNature: nature,
		FinalConsumer:   finalConsumer,
		Presence:        presence,
		Customer:        nfeCustomer,
		Items:           nfeItems,
		Payments:        nfePayments,
		AdditionalInfo:  truncateRunes(strings.TrimSpace(in.AdditionalInfo), 4000),
	}
	if t := in.Transport; t != nil {
		mode := strings.TrimSpace(t.Mode)
		if mode == "" {
			mode = "9"
		}
		if !slices.Contains([]string{"0", "1", "2", "3", "4", "9"}, mode) {
			add("modalidade do frete inválida")
		}
		tr := client.NfeTransport{Mode: mode}
		if c := t.Carrier; c != nil && clean(c.Name) != "" {
			doc := OnlyDigits(c.Document)
			carrier := client.NfeCarrier{
				Name:              truncateRunes(clean(c.Name), 60),
				StateRegistration: OnlyDigits(c.StateRegistration),
				Address:           truncateRunes(clean(c.Address), 60),
				CityName:          truncateRunes(clean(c.CityName), 60),
				State:             strings.ToUpper(strings.TrimSpace(c.State)),
			}
			switch len(doc) {
			case 0:
			case 11:
				carrier.CPF = doc
			case 14:
				carrier.CNPJ = doc
			default:
				add("o documento da transportadora deve ser CPF ou CNPJ")
			}
			if carrier.State != "" && !slices.Contains(ufs, carrier.State) {
				add("UF da transportadora inválida")
			}
			tr.Carrier = &carrier
		}
		for i, v := range t.Volumes {
			if v.Quantity < 1 {
				add("volume %d: a quantidade deve ser pelo menos 1", i+1)
				continue
			}
			tr.Volumes = append(tr.Volumes, client.NfeVolume{
				Quantity: v.Quantity, Species: truncateRunes(clean(v.Species), 60), Brand: truncateRunes(clean(v.Brand), 60),
				NetWeight: roundPtr(v.NetWeight, 3), GrossWeight: roundPtr(v.GrossWeight, 3),
			})
		}
		req.Transport = &tr
	}
	out.Nfe = &req
	return out, problems
}

func roundPtr(v *float64, decimals int) *float64 {
	if v == nil || *v <= 0 {
		return nil
	}
	r := round(*v, decimals)
	return &r
}
