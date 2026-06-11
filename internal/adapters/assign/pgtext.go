package assign

import (
	pgconv "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/pgtype"
	"github.com/jackc/pgx/v5/pgtype"
)

func SetPgTextIfNotEmpty(dst *pgtype.Text, src string) {
	if src != "" {
		*dst = pgconv.ParseStringToPgText(src)
	}
}
