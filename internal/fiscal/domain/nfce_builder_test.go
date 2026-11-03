package domain

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func simplesProfile() *TaxProfile {
	return &TaxProfile{TaxProfileInput: TaxProfileInput{
		Name: "Revenda", IcmsCSOSN: "102", CFOP: "5102", PisCST: "49", PisRate: f(0.65),
		CofinsCST: "49", CofinsRate: f(3), IbsCbsCST: "000", IbsCbsClassCode: "000001",
		IbsStateRate: f(0.1), IbsCityRate: f(0), CbsRate: f(0.9),
	}}
}

func item(name string, qty int32, price, disc float64) NfceSaleItem {
	return NfceSaleItem{
		ProductID: uuid.New(), Name: name, NCM: "61091000", Unit: "UN",
		Quantity: qty, UnitPrice: price, Discount: disc, Profile: simplesProfile(),
	}
}

func TestBuildNfceBasic(t *testing.T) {
	sale := NfceSale{
		Subtotal: 59.80, Discount: 0, Total: 59.80, PaymentMethod: "pix",
		Items: []NfceSaleItem{item("Camiseta – Preta – M", 2, 29.90, 0)},
	}
	sale.Items[0].Barcode = "7891000315507" // GTIN válido
	req, problems := BuildNfce(sale, "1")
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	it := req.Items[0]
	if it.GTIN != "7891000315507" || it.Code != "7891000315507" || it.CFOP != "5102" || it.Quantity != 2 {
		t.Fatalf("item errado: %+v", it)
	}
	if it.Tax.ICMS.CSOSN != "102" || it.Tax.ICMS.CST != "" || it.Tax.ICMS.Origin != "0" {
		t.Fatalf("icms errado: %+v", it.Tax.ICMS)
	}
	if *it.Tax.PIS.BaseValue != 59.80 || *it.Tax.PIS.Value != 0.39 || *it.Tax.COFINS.Value != 1.79 {
		t.Fatalf("pis/cofins errado: %+v %+v", it.Tax.PIS, it.Tax.COFINS)
	}
	if it.Tax.IBSCBS.Rates == nil || it.Tax.IBSCBS.Rates.CBS != 0.9 {
		t.Fatalf("ibs/cbs errado: %+v", it.Tax.IBSCBS)
	}
	if len(req.Payments) != 1 || req.Payments[0].Method != "17" || req.Payments[0].Amount != 59.80 {
		t.Fatalf("pagamento errado: %+v", req.Payments)
	}
	if req.Customer != nil {
		t.Fatal("sem CPF não manda destinatário")
	}
}

func TestBuildNfceDiscountCents(t *testing.T) {
	// 10,00 de desconto rateado em 3 itens iguais: 3,33 + 3,33 + 3,33 = 9,99 -> acerta 1 centavo
	sale := NfceSale{
		Subtotal: 30, Discount: 10, Total: 20, PaymentMethod: "cash",
		Items: []NfceSaleItem{item("A", 1, 10, 3.33), item("B", 1, 10, 3.33), item("C", 1, 10, 3.33)},
	}
	req, problems := BuildNfce(sale, "1")
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	var sum float64
	for _, it := range req.Items {
		sum += it.Discount
	}
	if cents(sum) != 1000 {
		t.Fatalf("desconto total %.2f, esperava 10,00", sum)
	}
	if req.Payments[0].Method != "01" || req.Payments[0].Amount != 20 {
		t.Fatalf("pagamento errado: %+v", req.Payments)
	}
}

func TestBuildNfcePayments(t *testing.T) {
	base := func() NfceSale {
		return NfceSale{Subtotal: 100, Total: 100, Items: []NfceSaleItem{item("A", 1, 100, 0)}}
	}
	cases := []struct {
		name string
		edit func(*NfceSale)
		want []string // "código:valor"
	}{
		{"crédito", func(s *NfceSale) { s.PaymentMethod = "credit_card" }, []string{"03:100"}},
		{"débito", func(s *NfceSale) { s.PaymentMethod = "debit_card" }, []string{"04:100"}},
		{"transferência", func(s *NfceSale) { s.PaymentMethod = "bank_transfer" }, []string{"18:100"}},
		{"crediário com entrada", func(s *NfceSale) { s.PaymentMethod = "installments"; s.DownPayment = 30 }, []string{"01:30", "05:70"}},
		{"crediário sem entrada", func(s *NfceSale) { s.PaymentMethod = "installments" }, []string{"05:100"}},
		{"vale-troca parcial + pix", func(s *NfceSale) { s.PaymentMethod = "pix"; s.Credit = 40 }, []string{"05:40", "17:60"}},
		{"vale-troca total", func(s *NfceSale) { s.PaymentMethod = "store_credit"; s.Credit = 100 }, []string{"05:100"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sale := base()
			tc.edit(&sale)
			req, problems := BuildNfce(sale, "1")
			if len(problems) > 0 {
				t.Fatal(problems)
			}
			var got []string
			for _, p := range req.Payments {
				got = append(got, fmt.Sprintf("%s:%g", p.Method, p.Amount))
			}
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("pagamentos %v, esperava %v", got, tc.want)
			}
		})
	}
}

func TestBuildNfceProblems(t *testing.T) {
	noNcm := item("Sem NCM", 1, 10, 0)
	noNcm.NCM = ""
	noProfile := item("Sem perfil", 1, 10, 0)
	noProfile.Profile = nil
	sale := NfceSale{Subtotal: 20, Total: 20, PaymentMethod: "other", BuyerDocument: "111.111.111-12",
		Items: []NfceSaleItem{noNcm, noProfile}}
	_, problems := BuildNfce(sale, "1")
	joined := strings.Join(problems, " | ")
	for _, want := range []string{"\"Sem NCM\" está sem NCM", "\"Sem perfil\" está sem perfil", "Outros", "CPF"} {
		if !strings.Contains(joined, want) {
			t.Errorf("faltou %q em %q", want, joined)
		}
	}
}

func TestBuildNfceRegimeMismatch(t *testing.T) {
	sale := NfceSale{Subtotal: 10, Total: 10, PaymentMethod: "cash", Items: []NfceSaleItem{item("A", 1, 10, 0)}}
	_, problems := BuildNfce(sale, "3") // regime normal com perfil de CSOSN
	if len(problems) != 1 || !strings.Contains(problems[0], "regime") {
		t.Fatalf("esperava erro de regime: %v", problems)
	}
}

func TestBuildNfceNormalRegimeIcms(t *testing.T) {
	it := item("A", 1, 100, 0)
	it.Origin = 2
	it.Profile.IcmsCSOSN = ""
	it.Profile.IcmsCST = "90"
	it.Profile.IcmsRate = f(18)
	sale := NfceSale{Subtotal: 100, Total: 100, PaymentMethod: "cash", BuyerDocument: "529.982.247-25",
		Items: []NfceSaleItem{it}}
	req, problems := BuildNfce(sale, "3")
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	icms := req.Items[0].Tax.ICMS
	if icms.CST != "90" || icms.Origin != "2" || icms.Own == nil || icms.Own.Rate != 18 || icms.Rate != nil {
		t.Fatalf("icms 90 errado: %+v", icms)
	}
	if req.Customer == nil || req.Customer.CPF != "52998224725" {
		t.Fatalf("CPF não foi: %+v", req.Customer)
	}
}

func TestValidGTIN(t *testing.T) {
	for code, want := range map[string]bool{
		"7891000315507":  true,
		"7891000315508":  false, // dígito errado
		"2000000000015":  false, // uso interno (grade)
		"96385074":       true,  // EAN-8
		"036000291452":   true,  // UPC-A
		"17891000315504": true,  // GTIN-14
		"123":            false,
		"ABC1000315507":  false,
	} {
		if got := ValidGTIN(code); got != want {
			t.Errorf("%s: %v, esperava %v", code, got, want)
		}
	}
}

func TestValidCPF(t *testing.T) {
	if !ValidCPF("52998224725") || ValidCPF("52998224726") || ValidCPF("11111111111") || ValidCPF("123") {
		t.Fatal("validação de CPF errada")
	}
}
