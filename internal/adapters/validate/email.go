package validate

import (
	"net/mail"
	"strings"
)

func IsValidEmail(email string) bool {
	email = strings.TrimSpace(email)

	// Verifica comprimento
	if len(email) < 3 || len(email) > 254 {
		return false
	}

	// Usa o parser da stdlib do Go
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}

	// Verifica se o endereço parseado é igual ao original (sem nome de exibição)
	return addr.Address == email
}
