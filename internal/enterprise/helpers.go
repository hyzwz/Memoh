package enterprise

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/memohai/memoh/internal/db"
)

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}

func parseUUID(id string) (pgtype.UUID, error) {
	return db.ParseUUID(id)
}
