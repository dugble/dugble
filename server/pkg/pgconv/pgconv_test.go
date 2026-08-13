package pgconv

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestUUIDStringPtr(t *testing.T) {
	t.Parallel()

	if UUIDStringPtr(nil) != nil {
		t.Fatal("UUIDStringPtr(nil) must be nil")
	}
	id := uuid.New()
	value := UUIDStringPtr(&id)
	if value == nil || *value != id.String() {
		t.Fatalf("UUIDStringPtr() = %v, want %s", value, id)
	}
}

func TestUUIDToUUIDPtr(t *testing.T) {
	t.Parallel()

	if UUIDToUUIDPtr(pgtype.UUID{}) != nil {
		t.Fatal("invalid PostgreSQL UUID must convert to nil")
	}
	id := uuid.New()
	value := UUIDToUUIDPtr(pgtype.UUID{Bytes: id, Valid: true})
	if value == nil || *value != id {
		t.Fatalf("UUIDToUUIDPtr() = %v, want %s", value, id)
	}
}

func TestTimestamptzConversionsNormalizeUTCAndNulls(t *testing.T) {
	t.Parallel()

	if TimestamptzToTimePtr(pgtype.Timestamptz{}) != nil {
		t.Fatal("invalid timestamptz must convert to nil")
	}
	input := time.Date(2026, time.August, 9, 12, 30, 0, 0, time.FixedZone("test", 2*60*60))
	encoded := TimestamptzFromTime(input)
	if !encoded.Valid || encoded.Time.Location() != time.UTC || !encoded.Time.Equal(input) {
		t.Fatalf("TimestamptzFromTime() = %+v", encoded)
	}
	decoded := TimestamptzToTimePtr(encoded)
	if decoded == nil || !decoded.Equal(input) {
		t.Fatalf("TimestamptzToTimePtr() = %v, want %v", decoded, input)
	}
}
