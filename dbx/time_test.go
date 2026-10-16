package dbx

import (
	"database/sql"
	"testing"
	"time"
)

func TestNullTime(t *testing.T) {
	if NullTime(time.Time{}).Valid {
		t.Error("zero time must be NULL")
	}

	ts := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	if got := NullTime(ts); !got.Valid || !got.Time.Equal(ts) {
		t.Errorf("NullTime(%s) = %+v", ts, got)
	}
}

func TestTimeFromNull(t *testing.T) {
	if !TimeFromNull(sql.NullTime{}).IsZero() {
		t.Error("NULL must be the zero time")
	}

	local := time.Date(2026, 10, 6, 14, 0, 0, 0, time.FixedZone("CEST", 2*60*60))

	got := TimeFromNull(sql.NullTime{Time: local, Valid: true})
	if !got.Equal(local) || got.Location() != time.UTC {
		t.Errorf("TimeFromNull(%s) = %s, want the same instant in UTC", local, got)
	}
}

func TestUnixNano(t *testing.T) {
	if UnixNano(time.Time{}) != 0 {
		t.Error("zero time must be 0")
	}

	ts := time.Date(2026, 10, 6, 12, 30, 15, 123456789, time.UTC)
	if got := UnixNano(ts); got != ts.UnixNano() {
		t.Errorf("UnixNano(%s) = %d, want %d", ts, got, ts.UnixNano())
	}
}

func TestTimeFromUnixNano(t *testing.T) {
	if !TimeFromUnixNano(0).IsZero() {
		t.Error("0 must be the zero time")
	}

	local := time.Date(2026, 10, 6, 14, 30, 15, 123456789, time.FixedZone("CEST", 2*60*60))

	got := TimeFromUnixNano(UnixNano(local))
	if !got.Equal(local) || got.Location() != time.UTC {
		t.Errorf("round trip of %s = %s, want the same instant in UTC, to the nanosecond", local, got)
	}
}
