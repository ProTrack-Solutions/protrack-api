package assign

import (
	pgconv "github.com/GabrielFerrarez19/ProTrack-2.0/protrack-server/internal/adapters/pgtype"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func SetPgUUIDIfNotNil(dst *pgtype.UUID, src uuid.UUID) {
	if src != (uuid.UUID{}) {
		*dst = pgconv.ParseUUIDToPgType(src)
	}
}
