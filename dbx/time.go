package dbx

import (
	"database/sql"
	"time"
)

// NullTime converts a time for a nullable timestamp column (Postgres
// TIMESTAMPTZ): a zero time is stored as NULL.
func NullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: !t.IsZero()}
}

// TimeFromNull converts a nullable timestamp column back: NULL becomes the
// zero time, anything else UTC.
func TimeFromNull(t sql.NullTime) time.Time {
	if !t.Valid {
		return time.Time{}
	}

	return t.Time.UTC()
}

// UnixNano converts a time for an INTEGER column (SQLite): Unix nanoseconds
// are compact, sortable and lossless. A zero time is stored as 0.
func UnixNano(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}

	return t.UnixNano()
}

// TimeFromUnixNano converts an INTEGER column back: 0 becomes the zero time,
// anything else UTC.
func TimeFromUnixNano(n int64) time.Time {
	if n == 0 {
		return time.Time{}
	}

	return time.Unix(0, n).UTC()
}
