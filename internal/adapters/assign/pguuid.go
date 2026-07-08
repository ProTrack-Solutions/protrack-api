package assign

import (
	pgconv "github.com/ProTrack-Solutions/protrack-api/internal/adapters/pgtype"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func SetPgUUIDIfNotNil(dst *pgtype.UUID, src uuid.UUID) {
	if src != (uuid.UUID{}) {
		*dst = pgconv.ParseUUIDToPgType(src)
	}
}
