package pgconv

import (
	"math"

	"github.com/jackc/pgx/v5/pgtype"
)

// IntToPgInt4 converte int para pgtype.Int4
func IntToPgInt4(value int) pgtype.Int4 {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return pgtype.Int4{Valid: false}
	}

	return pgtype.Int4{
		Int32: int32(value),
		Valid: true,
	}
}

// OptionalIntToPgInt4 retorna NULL (Valid: false) quando value <= 0; caso contrário converte para pgtype.Int4
func OptionalIntToPgInt4(value int) pgtype.Int4 {
	if value <= 0 {
		return pgtype.Int4{Valid: false}
	}
	return IntToPgInt4(value)
}

// PgInt4ToInt converte pgtype.Int4 para int
func PgInt4ToInt(value pgtype.Int4) int {
	if !value.Valid {
		return 0
	}

	return int(value.Int32)
}
