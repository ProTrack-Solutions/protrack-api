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

// ParsePgUUIDToUUID converte pgtype.UUID para uuid.UUID
func PgUUIDToUUID(value pgtype.UUID) uuid.UUID {
	if !value.Valid {
		return uuid.Nil
	}
	return value.Bytes
}
