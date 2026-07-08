package assign

import (
	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/jackc/pgx/v5/pgtype"
)

func SetPgTextIfNotEmpty(dst *pgtype.Text, src string) {
	if src != "" {
		*dst = pgconv.ParseStringToPgText(src)
	}
}
