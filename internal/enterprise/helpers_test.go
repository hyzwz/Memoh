package enterprise

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestUuidToString(t *testing.T) {
	u := pgtype.UUID{
		Bytes: [16]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
		Valid: true,
	}
	got := uuidToString(u)
	want := "12345678-9abc-def0-1234-56789abcdef0"
	if got != want {
		t.Fatalf("uuidToString = %q, want %q", got, want)
	}
}

func TestUuidToStringInvalid(t *testing.T) {
	u := pgtype.UUID{Valid: false}
	if s := uuidToString(u); s != "" {
		t.Fatalf("invalid UUID should return empty string, got %q", s)
	}
}
