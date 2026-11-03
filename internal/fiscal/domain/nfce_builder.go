package domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ProTrack-Solutions/protrack-api/internal/fiscal/client"
	"github.com/google/uuid"
)

// NfceSale é a venda como o montador precisa (já lida do banco).
type NfceSale struct {
	Subtotal      float64
	Discount      float64
	Total         float64
	DownPayment   float64 // entrada da venda a prazo (paga em dinheiro no PDV)
	Credit        float64 // parte paga com vale-troca
	PaymentMethod string
	BuyerDocument string
	Items         []NfceSaleItem
}

type NfceSaleItem struct {
	ProductID uuid.UUID
	Name      string
	Barcode   string
	NCM       string
	CEST      string
	Unit      string
	Origin    int16
	Quantity  int32
	UnitPrice float64
	Discount  float64
	Profile   *TaxProfile // já resolvido: produto -> categoria -> padrão da empresa
}

// Códigos de pagamento (tPag) da NFC-e por forma de pagamento do ERP.
// 99 (outros) e 90 (sem pagamento) não são aceitos na NFC-e.
const (
	tPagCash         = "01"
	tPagCreditCard   = "03"
	tPagDebitCard    = "04"
	tPagStoreCredit  = "05" // crédito loja: crediário e vale-troca
	tPagPix          = "17"
	tPagBankTransfer = "18"
)

var paymentCodes = map[string]string{
	"cash":          tPagCash,
	"credit_card":   tPagCreditCard,
	"debit_card":    tPagDebitCard,
	"pix":           tPagPix,
	"bank_transfer": tPagBankTransfer,
	"store_credit":  tPagStoreCredit,
	"installments":  tPagStoreCredit,
}

func cents(v float64) int64 { return int64(math.Round(v * 100)) }

func reais(c int64) float64 { return float64(c) / 100 }

// ValidCPF confere os dígitos verificadores.
func ValidCPF(cpf string) bool {
	if len(cpf) != 11 || strings.Count(cpf, cpf[:1]) == 11 {
		return false
	}
	for _, n := range []int{9, 10} {
		sum := 0
		for i := 0; i < n; i++ {
			sum += int(cpf[i]-'0') * (n + 1 - i)
		}
		d := (sum * 10) % 11
		if d == 10 {
			d = 0
		}
		if d != int(cpf[n]-'0') {
			return false
		}
	}
	return true
}

// ValidGTIN diz se o código de barras pode ir como GTIN: 8, 12, 13 ou 14 dígitos
// com dígito verificador certo e fora da faixa de uso interno (EAN-13 com prefixo 2,
// como os códigos gerados pela grade). O resto sai como "SEM GTIN".
func ValidGTIN(code string) bool {
	switch len(code) {
	case 8, 12, 13, 14:
	default:
		return false
	}
	if OnlyDigits(code) != code {
		return false
	}
	if len(code) == 13 && code[0] == '2' {
		return false
	}
	sum := 0
	for i := len(code) - 2; i >= 0; i-- {
		d := int(code[i] - '0')
		if (len(code)-2-i)%2 == 0 {
			d *= 3
		}
		sum += d
	}
	return (10-sum%10)%10 == int(code[len(code)-1]-'0')
}

func truncateRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return strings.TrimSpace(string([]rune(s)[:max]))
}

func pct(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

// icmsFor monta o grupo de ICMS do perfil.
func icmsFor(p TaxProfile, origin int16) client.ICMS {
	icms := client.ICMS{Origin: strconv.Itoa(int(origin))}
	if p.IcmsCSOSN != "" {
		icms.CSOSN = p.IcmsCSOSN
		if p.IcmsCSOSN == "900" && p.IcmsRate != nil {
			icms.Own = &client.OwnICMS{Rate: *p.IcmsRate, BaseReduction: p.IcmsBaseReduction}
		}
		return icms
	}
	icms.CST = p.IcmsCST
	switch p.IcmsCST {
	case "00":
		icms.Rate, icms.FCPRate = p.IcmsRate, p.IcmsFcpRate
	case "20":
		icms.Rate, icms.BaseReduction, icms.FCPRate = p.IcmsRate, p.IcmsBaseReduction, p.IcmsFcpRate
	case "90":
		if p.IcmsRate != nil {
			icms.Own = &client.OwnICMS{Rate: *p.IcmsRate, BaseReduction: p.IcmsBaseReduction, FCPRate: p.IcmsFcpRate}
		}
	}
	return icms
}

// contribution monta PIS/COFINS: 49/99 com base = valor líquido do item.
func contribution(cst string, rate *float64, netCents int64) client.Contribution {
	out := client.Contribution{CST: cst}
	if cst == "49" || cst == "99" {
		base := reais(netCents)
		r := pct(rate)
		value := reais(int64(math.Round(float64(netCents) * r / 100)))
		out.BaseValue, out.Rate, out.Value = &base, &r, &value
	}
	return out
}

// BuildNfce transforma a venda no pedido de NFC-e da API fiscal. Devolve a lista
// do que impede a emissão (produto sem NCM, sem perfil...) em vez de um erro só,
// para o lojista corrigir tudo de uma vez.
func BuildNfce(sale NfceSale, taxRegime string) (client.NfceRequest, []string) {
	var problems []string
	var req client.NfceRequest
	if len(sale.Items) == 0 {
		return req, []string{"a venda não tem itens"}
	}

	// Desconto: usa o rateio gravado nos itens e acerta os centavos no maior item
	gross := make([]int64, len(sale.Items))
	disc := make([]int64, len(sale.Items))
	var sumDisc int64
	largest := 0
	for i, it := range sale.Items {
		gross[i] = int64(it.Quantity) * cents(it.UnitPrice)
		disc[i] = cents(it.Discount)
		if disc[i] > gross[i] {
			disc[i] = gross[i]
		}
		sumDisc += disc[i]
		if gross[i]-disc[i] > gross[largest]-disc[largest] {
			largest = i
		}
	}
	if diff := cents(sale.Discount) - sumDisc; diff != 0 {
		disc[largest] += diff
		if disc[largest] < 0 || disc[largest] > gross[largest] {
			problems = append(problems, "o desconto da venda não fecha com os itens")
		}
	}

	simples := taxRegime == "1" || taxRegime == "4"
	var totalNet int64
	for i, it := range sale.Items {
		name := truncateRunes(it.Name, 120)
		ncm := OnlyDigits(it.NCM)
		if len(ncm) != 8 {
			problems = append(problems, fmt.Sprintf("o produto \"%s\" está sem NCM", name))
		}
		if it.Profile == nil {
			problems = append(problems, fmt.Sprintf("o produto \"%s\" está sem perfil tributário (defina o perfil padrão em Configurações → Fiscal)", name))
			continue
		}
		p := *it.Profile
		if taxRegime != "" && (p.IcmsCSOSN != "") != simples {
			problems = append(problems, fmt.Sprintf("o perfil \"%s\" não combina com o regime tributário da empresa (Simples usa CSOSN; Normal usa CST)", p.Name))
		}
		net := gross[i] - disc[i]
		totalNet += net
		code := strings.TrimSpace(it.Barcode)
		if code == "" {
			code = strings.ToUpper(strings.ReplaceAll(it.ProductID.String(), "-", "")[:12])
		}
		item := client.NfceItem{
			Code:        truncateRunes(code, 60),
			Description: name,
			NCM:         ncm,
			CEST:        OnlyDigits(it.CEST),
			BenefitCode: p.BenefitCode,
			Unit:        it.Unit,
			Quantity:    float64(it.Quantity),
			UnitPrice:   it.UnitPrice,
			Discount:    reais(disc[i]),
			CFOP:        p.CFOP,
			Tax: client.NfceTax{
				ICMS:   icmsFor(p, it.Origin),
				PIS:    contribution(p.PisCST, p.PisRate, net),
				COFINS: contribution(p.CofinsCST, p.CofinsRate, net),
				IBSCBS: client.IBSCBS{CST: p.IbsCbsCST, ClassCode: p.IbsCbsClassCode},
			},
		}
		if item.Unit == "" {
			item.Unit = "UN"
		}
		if ValidGTIN(code) {
			item.GTIN = code
		}
		if p.IbsStateRate != nil && p.IbsCityRate != nil && p.CbsRate != nil {
			item.Tax.IBSCBS.Rates = &client.IBSCBSRates{IBSState: *p.IbsStateRate, IBSCity: *p.IbsCityRate, CBS: *p.CbsRate}
		}
		req.Items = append(req.Items, item)
	}

	// Pagamentos: precisam somar o total da nota
	total := cents(sale.Total)
	if len(problems) == 0 && totalNet != total {
		problems = append(problems, fmt.Sprintf("o total dos itens (R$ %.2f) não bate com o total da venda (R$ %.2f)", reais(totalNet), reais(total)))
	}
	add := func(code string, amount int64) {
		if amount > 0 {
			req.Payments = append(req.Payments, client.NfcePayment{Method: code, Amount: reais(amount)})
		}
	}
	credit := cents(sale.Credit)
	rest := total - credit
	add(tPagStoreCredit, credit)
	switch code, ok := paymentCodes[sale.PaymentMethod]; {
	case sale.PaymentMethod == "installments":
		down := cents(sale.DownPayment)
		add(tPagCash, down)
		add(tPagStoreCredit, rest-down)
	case ok:
		add(code, rest)
	case rest > 0:
		problems = append(problems, "a forma de pagamento \"Outros\" não é aceita na NFC-e; use dinheiro, cartão, Pix, transferência ou crediário")
	}
	if len(req.Payments) == 0 && len(problems) == 0 {
		problems = append(problems, "a venda não tem pagamento para informar na nota")
	}

	if doc := OnlyDigits(sale.BuyerDocument); doc != "" {
		switch {
		case len(doc) == 11 && ValidCPF(doc):
			req.Customer = &client.NfceCustomer{CPF: doc}
		case len(doc) == 11:
			problems = append(problems, "o CPF informado na venda é inválido")
		}
		// CNPJ na NFC-e não é suportado pela API fiscal: a nota sai sem destinatário
	}
	return req, problems
}
