// Package variation concentra as regras da grade de produtos (cor × tamanho)
// usadas por Produtos e Compras: cada combinação é um produto normal agrupado
// por products.model_name.
package variation

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// Separador do nome gerado: "Camiseta Básica – Preta – M"
const NameSeparator = " – "

// MaxProductName é o tamanho da coluna products.name
const MaxProductName = 100

// Name monta o nome do produto pulando cor/tamanho vazios
func Name(model, color, size string) string {
	parts := []string{strings.TrimSpace(model)}
	for _, p := range []string{color, size} {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, NameSeparator)
}

// Key identifica a combinação sem diferenciar maiúsculas (mesma regra do índice único)
func Key(color, size string) string {
	return strings.ToLower(strings.TrimSpace(color)) + "|" + strings.ToLower(strings.TrimSpace(size))
}

// InternalEAN13 gera um EAN-13 de circulação interna: o prefixo 2 é a faixa
// que o GS1 reserva para uso dentro da loja, então não colide com o código de
// fábrica de nenhum produto.
func InternalEAN13() (string, error) {
	digits := make([]byte, 12)
	digits[0] = '2'
	for i := 1; i < 12; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		digits[i] = byte('0' + n.Int64())
	}

	sum := 0
	for i, d := range digits {
		weight := 1
		if i%2 == 1 {
			weight = 3
		}
		sum += int(d-'0') * weight
	}
	check := (10 - sum%10) % 10
	return string(digits) + string(rune('0'+check)), nil
}
