package domain

import (
	"errors"
	"strings"
	"testing"
)

func f(v float64) *float64 { return &v }

func baseProfile() TaxProfileInput {
	return TaxProfileInput{
		Name:            "Revenda Simples",
		IcmsCSOSN:       "102",
		IbsCbsCST:       "000",
		IbsCbsClassCode: "000001",
	}
}

func TestTaxProfileDefaults(t *testing.T) {
	out, err := baseProfile().Normalize("1")
	if err != nil {
		t.Fatal(err)
	}
	if out.CFOP != "5102" || out.PisCST != "49" || out.CofinsCST != "49" {
		t.Fatalf("padrões errados: %+v", out)
	}
}

func TestTaxProfileCleansFields(t *testing.T) {
	in := baseProfile()
	in.IcmsRate = f(18) // CSOSN 102 não tem ICMS próprio
	in.IcmsFcpRate = f(2)
	in.PisCST = "07"
	in.PisRate = f(1.65) // CST 07 não leva valor
	in.BenefitCode = " mg800001 "
	in.CFOP = "5.405"
	out, err := in.Normalize("1")
	if err != nil {
		t.Fatal(err)
	}
	if out.IcmsRate != nil || out.IcmsFcpRate != nil || out.PisRate != nil {
		t.Fatalf("campos sem uso não foram limpos: %+v", out)
	}
	if out.BenefitCode != "MG800001" || out.CFOP != "5405" {
		t.Fatalf("normalização errada: %+v", out)
	}
}

func TestTaxProfileValidation(t *testing.T) {
	cases := []struct {
		name   string
		regime string
		edit   func(*TaxProfileInput)
		want   string
	}{
		{"nome curto", "1", func(p *TaxProfileInput) { p.Name = "a" }, "nome"},
		{"csosn e cst", "1", func(p *TaxProfileInput) { p.IcmsCST = "00" }, "não os dois"},
		{"sem icms", "1", func(p *TaxProfileInput) { p.IcmsCSOSN = "" }, "CSOSN"},
		{"csosn com st", "1", func(p *TaxProfileInput) { p.IcmsCSOSN = "201" }, "não é aceito"},
		{"csosn no regime normal", "3", func(p *TaxProfileInput) {}, "use CST"},
		{"cst no simples", "1", func(p *TaxProfileInput) { p.IcmsCSOSN = ""; p.IcmsCST = "00"; p.IcmsRate = f(18) }, "usa CSOSN"},
		{"cst 00 sem alíquota", "3", func(p *TaxProfileInput) { p.IcmsCSOSN = ""; p.IcmsCST = "00" }, "alíquota"},
		{"cst 20 sem redução", "3", func(p *TaxProfileInput) { p.IcmsCSOSN = ""; p.IcmsCST = "20"; p.IcmsRate = f(18) }, "redução"},
		{"cst 10 (ST)", "3", func(p *TaxProfileInput) { p.IcmsCSOSN = ""; p.IcmsCST = "10" }, "não é aceito"},
		{"alíquota acima de 100", "3", func(p *TaxProfileInput) { p.IcmsCSOSN = ""; p.IcmsCST = "00"; p.IcmsRate = f(120) }, "entre 0 e 100"},
		{"cfop interestadual", "1", func(p *TaxProfileInput) { p.CFOP = "6102" }, "5xxx"},
		{"cbenef inválido", "1", func(p *TaxProfileInput) { p.BenefitCode = "MG80" }, "cBenef"},
		{"pis cst inválido", "1", func(p *TaxProfileInput) { p.PisCST = "01" }, "PIS"},
		{"ibs cst", "1", func(p *TaxProfileInput) { p.IbsCbsCST = "1" }, "3 dígitos"},
		{"class code", "1", func(p *TaxProfileInput) { p.IbsCbsClassCode = "12" }, "6 dígitos"},
		{"alíquotas ibs incompletas", "1", func(p *TaxProfileInput) { p.CbsRate = f(0.9) }, "três alíquotas"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := baseProfile()
			tc.edit(&in)
			_, err := in.Normalize(tc.regime)
			if !errors.Is(err, ErrInvalidTaxProfile) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("esperava erro com %q, veio %v", tc.want, err)
			}
		})
	}
}

func TestTaxProfileNormalRegime(t *testing.T) {
	in := baseProfile()
	in.IcmsCSOSN = ""
	in.IcmsCST = "20"
	in.IcmsRate = f(18)
	in.IcmsBaseReduction = f(33.33)
	in.IbsStateRate, in.IbsCityRate, in.CbsRate = f(0.1), f(0), f(0.9)
	out, err := in.Normalize("3")
	if err != nil {
		t.Fatal(err)
	}
	if *out.IcmsBaseReduction != 33.33 || *out.CbsRate != 0.9 {
		t.Fatalf("valores perdidos: %+v", out)
	}
	// Regime ainda não informado aceita os dois
	if _, err := baseProfile().Normalize(""); err != nil {
		t.Fatal(err)
	}
}

func TestProductFiscalNormalize(t *testing.T) {
	out, err := ProductFiscalInput{NCM: "6109.10.00", CEST: "28.038.00", Unit: "un"}.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if out.NCM != "61091000" || out.CEST != "2803800" || out.Unit != "UN" {
		t.Fatalf("normalização errada: %+v", out)
	}
	for name, in := range map[string]ProductFiscalInput{
		"ncm curto":  {NCM: "6109"},
		"cest curto": {CEST: "123"},
		"unidade":    {Unit: "CX"},
		"origem":     {Origin: 9},
	} {
		if _, err := in.Normalize(); !errors.Is(err, ErrInvalidFiscalData) {
			t.Errorf("%s: esperava erro, veio %v", name, err)
		}
	}
}

func TestFiscalProfileNormalize(t *testing.T) {
	valid := FiscalProfileInput{
		StateRegistration: "001.234.567/0089", TaxRegime: "1", Street: "Rua A", Number: "10",
		District: "Centro", CityCode: "3106200", CityName: "Belo Horizonte", ZipCode: "30.110-000",
		Phone: "(31) 3333-4444",
	}
	out, err := valid.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if out.StateRegistration != "0012345670089" || out.ZipCode != "30110000" || out.Phone != "3133334444" {
		t.Fatalf("normalização errada: %+v", out)
	}
	bad := map[string]func(*FiscalProfileInput){
		"ie":         func(p *FiscalProfileInput) { p.StateRegistration = "" },
		"regime":     func(p *FiscalProfileInput) { p.TaxRegime = "5" },
		"ibge":       func(p *FiscalProfileInput) { p.CityCode = "310620" },
		"cep":        func(p *FiscalProfileInput) { p.ZipCode = "301" },
		"csc só id":  func(p *FiscalProfileInput) { p.CSCID = "1" },
		"rua longa":  func(p *FiscalProfileInput) { p.Street = strings.Repeat("a", 61) },
		"sem bairro": func(p *FiscalProfileInput) { p.District = " " },
	}
	for name, edit := range bad {
		in := valid
		edit(&in)
		if _, err := in.Normalize(); !errors.Is(err, ErrInvalidFiscalData) {
			t.Errorf("%s: esperava erro, veio %v", name, err)
		}
	}
}
