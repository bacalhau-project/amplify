package util

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

func NullStr(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

func NullBool(b bool) sql.NullBool {
	return sql.NullBool{Bool: b, Valid: true}
}

func NullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: true}
}

func NullUUID(u uuid.UUID) uuid.NullUUID {
	return uuid.NullUUID{UUID: u, Valid: true}
}

func NullInt32(i int32) sql.NullInt32 {
	return sql.NullInt32{Int32: i, Valid: true}
}
