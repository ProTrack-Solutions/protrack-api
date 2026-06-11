package pgconv

import "github.com/jackc/pgx/v5/pgtype"

func PgBoolToBool(value pgtype.Bool) bool {
	if !value.Valid {
		return false
	}
	return value.Bool
}

func BoolToPgBool(value bool) pgtype.Bool {
	return pgtype.Bool{
		Bool:  value,
		Valid: true,
	}
}
