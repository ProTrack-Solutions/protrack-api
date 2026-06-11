package pgconv

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

// StringBRToPgNumeric converte uma string decimal (formato brasileiro ou americano) para pgtype.Numeric
// Aceita formatos: "25,00" (BR) ou "25.00" (US) ou "1.250,50" (BR com milhares) ou "1,250.50" (US com milhares)
func StringBRToPgNumeric(value string) (pgtype.Numeric, error) {
	value = strings.TrimSpace(value)

	// Verifica se tem vírgula (formato brasileiro)
	if strings.Contains(value, ",") {
		// Formato brasileiro: remove pontos (milhares) e substitui vírgula por ponto
		value = strings.ReplaceAll(value, ".", "")
		value = strings.ReplaceAll(value, ",", ".")
	} else if strings.Contains(value, ".") {
		// Formato americano: verifica se tem múltiplos pontos (milhares)
		dotCount := strings.Count(value, ".")
		if dotCount > 1 {
			// Tem pontos de milhares, remove todos exceto o último (que é decimal)
			lastDotIndex := strings.LastIndex(value, ".")
			// Remove todos os pontos antes do último
			beforeLastDot := strings.ReplaceAll(value[:lastDotIndex], ".", "")
			afterLastDot := value[lastDotIndex+1:]
			value = beforeLastDot + "." + afterLastDot
		}
		// Se tem apenas um ponto, assume que é decimal e usa como está
	}

	var n pgtype.Numeric
	if err := n.Scan(value); err != nil {
		return n, err
	}

	return n, nil
}

// PgNumericToString converte pgtype.Numeric para string
func PgNumericToString(n pgtype.Numeric) string {
	if !n.Valid {
		return ""
	}

	if n.Int == nil {
		return ""
	}

	// Constrói big.Rat a partir de Int e Exp
	rat := new(big.Rat).SetInt(n.Int)

	// Aplica o expoente (10^Exp)
	// O valor armazenado é Int * 10^Exp, então precisamos multiplicar por 10^Exp
	// Para expoentes negativos, dividimos por 10^(-Exp)
	if n.Exp != 0 {
		if n.Exp > 0 {
			// Expoente positivo: multiplica por 10^Exp
			exp := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n.Exp)), nil))
			rat.Mul(rat, exp)
		} else {
			// Expoente negativo: divide por 10^(-Exp)
			exp := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-n.Exp)), nil))
			rat.Quo(rat, exp)
		}
	}

	scale := int(-n.Exp)
	if scale < 0 {
		scale = 0
	}
	return rat.FloatString(scale)
}

func Float64ToPgNumeric(v float64) pgtype.Numeric {
	var n pgtype.Numeric

	// converte para string (respeita casas decimais)
	err := n.Scan(fmt.Sprintf("%.2f", v))
	if err != nil {
		return pgtype.Numeric{}
	}

	return n
}

func PgNumericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}

	f, err := n.Float64Value()
	if err != nil {
		return 0
	}

	return f.Float64
}
