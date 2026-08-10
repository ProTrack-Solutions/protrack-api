package pgconv

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// ParseUUIDToPgType converte uuid.UUID para pgtype.UUID
func ParseUUIDToPgType(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: value,
		Valid: true,
	}
}

// OptionalUUIDToPgType converte uuid.UUID para pgtype.UUID, tratando uuid.Nil
// como ausência de valor (NULL), diferente de ParseUUIDToPgType que sempre
// marca Valid: true. Uso: campos opcionais como sales.customer_id (venda avulsa).
func OptionalUUIDToPgType(value uuid.UUID) pgtype.UUID {
	if value == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{
		Bytes: value,
		Valid: true,
	}
}

// ParsePgUUIDToUUID converte pgtype.UUID para uuid.UUID
func PgUUIDToUUID(value pgtype.UUID) uuid.UUID {
	if !value.Valid {
		return uuid.Nil
	}
	return value.Bytes
}
