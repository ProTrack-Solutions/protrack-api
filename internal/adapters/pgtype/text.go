package pgconv

import "github.com/jackc/pgx/v5/pgtype"

// ParseStringToPgType converte string para pgtype.Text
func ParseStringToPgType(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{
		String: value,
		Valid:  true,
	}
}

// ParseStringToPgText é um alias para ParseStringToPgType (mantido para compatibilidade)
func ParseStringToPgText(value string) pgtype.Text {
	return ParseStringToPgType(value)
}

// ParsePgTextToString converte pgtype.Text para string
func ParsePgTextToString(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
