package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func manualProfile() *TaxProfile {
	return &TaxProfile{TaxProfileInput: TaxProfileInput{
		Name: "Padrão", IcmsCSOSN: "102", CFOP: "5102", PisCST: "49", CofinsCST: "49",
		IbsCbsCST: "000", IbsCbsClassCode: "000001",
	}}
}

func manualCatalog(id uuid.UUID) map[uuid.UUID]ManualProduct {
	return map[uuid.UUID]ManualProduct{id: {
		ID: id, Name: "Camiseta polo", Barcode: "7891234567895", NCM: "61051000", Unit: "UN", Profile: manualProfile(),
	}}
}

func nfeRecipient(uf string) *ManualRecipient {
	return &ManualRecipient{
		Document: "11.222.333/0001-81", Name: "Loja Cliente LTDA", IEIndicator: "1", StateRegistration: "123456789",
		Address: ManualAddress{
			Street: "Rua A", Number: "10", District: "Centro", CityCode: "3550308", CityName: "São Paulo",
			State: uf, ZipCode: "01001-000",
		},
	}
}

func hasProblem(problems []string, part string) bool {
	for _, p := range problems {
		if strings.Contains(p, part) {
			return true
		}
	}
	return false
}

func TestValidCNPJ(t *testing.T) {
	if !ValidCNPJ("11222333000181") {
		t.Fatal("CNPJ válido recusado")
	}
	if ValidCNPJ("11222333000182") || ValidCNPJ("11111111111111") {
		t.Fatal("CNPJ inválido aceito")
	}
}

func TestBuildManualNfeInterstateSwapsCFOP(t *testing.T) {
	id := uuid.New()
	in := ManualDocumentInput{
		Model:     ModelNfe,
		Recipient: nfeRecipient("SP"),
		Items:     []ManualItemInput{{ProductID: id, Quantity: 2, UnitPrice: 50, Discount: 10}},
		Payments:  []ManualPayment{{Method: "15", Amount: 90}},
		Transport: &ManualTransport{Mode: "0", Volumes: []ManualVolume{{Quantity: 1, Species: "Caixa"}}},
	}
	out, problems := BuildManual(in, manualCatalog(id), "MG", "1")
	if len(problems) > 0 {
		t.Fatalf("problemas inesperados: %v", problems)
	}
	req := out.Nfe
	if req == nil || out.Nfce != nil {
		t.Fatal("esperava pedido de NF-e")
	}
	if got := req.Items[0].CFOP; got != "6102" {
		t.Fatalf("CFOP interestadual = %s, quero 6102", got)
	}
	if req.Customer.CNPJ != "11222333000181" || req.Customer.CPF != "" {
		t.Fatalf("destinatário: %+v", req.Customer)
	}
	if req.FinalConsumer {
		t.Fatal("contribuinte (indicador 1) não é consumidor final por padrão")
	}
	if req.OperationNature != DefaultOperationNature || req.Presence != "1" {
		t.Fatalf("padrões: natureza %q presença %q", req.OperationNature, req.Presence)
	}
	if out.ItemsTotal != 90 || req.Items[0].Discount != 10 {
		t.Fatalf("total %v desconto %v", out.ItemsTotal, req.Items[0].Discount)
	}
	if req.Items[0].GTIN != "7891234567895" {
		t.Fatalf("GTIN = %q", req.Items[0].GTIN)
	}
}

func TestBuildManualNfeSameStateRejectsInterstateCFOP(t *testing.T) {
	id := uuid.New()
	in := ManualDocumentInput{
		Model:     ModelNfe,
		Recipient: nfeRecipient("MG"),
		Items:     []ManualItemInput{{ProductID: id, Quantity: 1, UnitPrice: 10, CFOP: "6.102"}},
		Payments:  []ManualPayment{{Method: "90", Amount: 0}},
	}
	_, problems := BuildManual(in, manualCatalog(id), "MG", "1")
	if !hasProblem(problems, "mesmo estado usa CFOP 5xxx") {
		t.Fatalf("esperava recusa do CFOP 6xxx: %v", problems)
	}
}

func TestBuildManualNfeRecipientChecks(t *testing.T) {
	id := uuid.New()
	r := nfeRecipient("SP")
	r.StateRegistration = ""
	r.Address.CityCode = ""
	in := ManualDocumentInput{
		Model: ModelNfe, Recipient: r,
		Items:    []ManualItemInput{{ProductID: id, Quantity: 1, UnitPrice: 10}},
		Payments: []ManualPayment{{Method: "99", Amount: 10}},
	}
	_, problems := BuildManual(in, manualCatalog(id), "MG", "1")
	for _, want := range []string{"inscrição estadual", "código IBGE", "Outros"} {
		if !hasProblem(problems, want) {
			t.Errorf("faltou o problema %q: %v", want, problems)
		}
	}
	if _, problems := BuildManual(ManualDocumentInput{Model: ModelNfe, Items: in.Items, Payments: in.Payments}, manualCatalog(id), "MG", "1"); !hasProblem(problems, "precisa do destinatário") {
		t.Errorf("NF-e sem destinatário passou: %v", problems)
	}
}

func TestBuildManualDifalOnlyForInterstateFinalConsumer(t *testing.T) {
	id := uuid.New()
	in := ManualDocumentInput{
		Model:     ModelNfe,
		Recipient: nfeRecipient("SP"), // contribuinte: não é consumidor final
		Items: []ManualItemInput{{ProductID: id, Quantity: 1, UnitPrice: 100,
			Difal: &ManualDifal{DestinationRate: 18, InterstateRate: 12}}},
		Payments: []ManualPayment{{Method: "17", Amount: 100}},
	}
	if _, problems := BuildManual(in, manualCatalog(id), "MG", "1"); !hasProblem(problems, "DIFAL") {
		t.Fatalf("DIFAL para contribuinte passou: %v", problems)
	}
	in.Recipient.Document = "529.982.247-25"
	in.Recipient.IEIndicator = "9"
	in.Recipient.StateRegistration = ""
	out, problems := BuildManual(in, manualCatalog(id), "MG", "1")
	if len(problems) > 0 {
		t.Fatalf("problemas: %v", problems)
	}
	if !out.Nfe.FinalConsumer || out.Nfe.Items[0].Tax.DIFAL == nil || out.Nfe.Items[0].Tax.DIFAL.Method != "single" {
		t.Fatalf("DIFAL/consumidor final: %+v", out.Nfe)
	}
	if out.Nfe.Customer.CPF != "52998224725" {
		t.Fatalf("CPF = %q", out.Nfe.Customer.CPF)
	}
}

func TestBuildManualNfce(t *testing.T) {
	id := uuid.New()
	in := ManualDocumentInput{
		Model:     ModelNfce,
		Recipient: &ManualRecipient{Document: "529.982.247-25"},
		Items:     []ManualItemInput{{ProductID: id, Quantity: 3, UnitPrice: 10, Description: "Polo azul"}},
		Payments:  []ManualPayment{{Method: "01", Amount: 20}, {Method: "17", Amount: 10}},
	}
	out, problems := BuildManual(in, manualCatalog(id), "MG", "1")
	if len(problems) > 0 {
		t.Fatalf("problemas: %v", problems)
	}
	if out.Nfce == nil || out.Nfe != nil {
		t.Fatal("esperava pedido de NFC-e")
	}
	if out.Nfce.Customer == nil || out.Nfce.Customer.CPF != "52998224725" {
		t.Fatalf("CPF: %+v", out.Nfce.Customer)
	}
	if out.Nfce.Items[0].Description != "Polo azul" || out.Nfce.Items[0].CFOP != "5102" {
		t.Fatalf("item: %+v", out.Nfce.Items[0])
	}

	in.Payments = []ManualPayment{{Method: "01", Amount: 20}}
	if _, problems := BuildManual(in, manualCatalog(id), "MG", "1"); !hasProblem(problems, "não cobrem o total") {
		t.Fatalf("pagamento menor que o total passou: %v", problems)
	}
	in.Payments = []ManualPayment{{Method: "01", Amount: 30}}
	in.Recipient = &ManualRecipient{Document: "11222333000181"}
	if _, problems := BuildManual(in, manualCatalog(id), "MG", "1"); !hasProblem(problems, "só aceita CPF") {
		t.Fatalf("CNPJ na NFC-e passou: %v", problems)
	}
	in.Recipient = nil
	in.Items[0].CFOP = "6102"
	if _, problems := BuildManual(in, manualCatalog(id), "MG", "1"); !hasProblem(problems, "5xxx") {
		t.Fatalf("CFOP 6xxx na NFC-e passou: %v", problems)
	}
}

func TestBuildManualProductProblems(t *testing.T) {
	id := uuid.New()
	catalog := manualCatalog(id)
	p := catalog[id]
	p.NCM = ""
	p.Profile = nil
	catalog[id] = p
	in := ManualDocumentInput{
		Model:    ModelNfce,
		Items:    []ManualItemInput{{ProductID: id, Quantity: 1, UnitPrice: 10}, {ProductID: uuid.New(), Quantity: 1}},
		Payments: []ManualPayment{{Method: "01", Amount: 10}},
	}
	_, problems := BuildManual(in, catalog, "MG", "1")
	for _, want := range []string{"sem NCM", "sem perfil tributário", "item 2: produto não encontrado"} {
		if !hasProblem(problems, want) {
			t.Errorf("faltou %q: %v", want, problems)
		}
	}
}

func TestBuildManualDefaultCFOPFollowsRecipientState(t *testing.T) {
	id := uuid.New()
	in := ManualDocumentInput{
		Model:       ModelNfe,
		DefaultCFOP: "5.910",
		Recipient:   nfeRecipient("SP"),
		Items:       []ManualItemInput{{ProductID: id, Quantity: 1, UnitPrice: 10}, {ProductID: id, Quantity: 1, UnitPrice: 5, CFOP: "6102"}},
		Payments:    []ManualPayment{{Method: "90", Amount: 0}},
	}
	out, problems := BuildManual(in, manualCatalog(id), "MG", "1")
	if len(problems) > 0 {
		t.Fatalf("problemas: %v", problems)
	}
	if out.Nfe.Items[0].CFOP != "6910" || out.Nfe.Items[1].CFOP != "6102" {
		t.Fatalf("CFOPs = %s, %s", out.Nfe.Items[0].CFOP, out.Nfe.Items[1].CFOP)
	}
	in.Recipient = nfeRecipient("MG")
	in.Items = in.Items[:1]
	in.DefaultCFOP = "6910"
	out, problems = BuildManual(in, manualCatalog(id), "MG", "1")
	if len(problems) > 0 || out.Nfe.Items[0].CFOP != "5910" {
		t.Fatalf("mesmo estado: %v %v", problems, out.Nfe.Items)
	}
}
