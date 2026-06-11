package validate

import (
	"errors"
	"strings"
	"unicode"
)

var (
	ErrInvalidDocument = errors.New("invalid document")
	ErrInvalidCPF      = errors.New("invalid CPF")
	ErrInvalidCNPJ     = errors.New("invalid CNPJ")
)

// ValidateDocument detecta se é CPF ou CNPJ e valida
func ValidateDocument(value string) (string, error) {
	doc := onlyDigits(value)

	switch len(doc) {
	case 11:
		if !isValidCPF(doc) {
			return "", ErrInvalidCPF
		}
		return "CPF", nil

	case 14:
		if !isValidCNPJ(doc) {
			return "", ErrInvalidCNPJ
		}
		return "CNPJ", nil

	default:
		return "", ErrInvalidDocument
	}
}

// ---------- helpers ----------

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ---------- CPF ----------

func isValidCPF(cpf string) bool {
	// Rejeita sequência repetida
	allEqual := true
	for i := 1; i < 11; i++ {
		if cpf[i] != cpf[0] {
			allEqual = false
			break
		}
	}
	if allEqual {
		return false
	}

	// Primeiro dígito
	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(cpf[i]-'0') * (10 - i)
	}
	d1 := (sum * 10) % 11
	if d1 == 10 {
		d1 = 0
	}
	if d1 != int(cpf[9]-'0') {
		return false
	}

	// Segundo dígito
	sum = 0
	for i := 0; i < 10; i++ {
		sum += int(cpf[i]-'0') * (11 - i)
	}
	d2 := (sum * 10) % 11
	if d2 == 10 {
		d2 = 0
	}
	return d2 == int(cpf[10]-'0')
}

// ---------- CNPJ ----------

func isValidCNPJ(cnpj string) bool {
	// Rejeita sequência repetida
	allEqual := true
	for i := 1; i < 14; i++ {
		if cnpj[i] != cnpj[0] {
			allEqual = false
			break
		}
	}
	if allEqual {
		return false
	}

	weights1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	weights2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

	// Primeiro dígito
	sum := 0
	for i := 0; i < 12; i++ {
		sum += int(cnpj[i]-'0') * weights1[i]
	}
	r := sum % 11
	d1 := 0
	if r >= 2 {
		d1 = 11 - r
	}
	if d1 != int(cnpj[12]-'0') {
		return false
	}

	// Segundo dígito
	sum = 0
	for i := 0; i < 13; i++ {
		sum += int(cnpj[i]-'0') * weights2[i]
	}
	r = sum % 11
	d2 := 0
	if r >= 2 {
		d2 = 11 - r
	}
	return d2 == int(cnpj[13]-'0')
}
