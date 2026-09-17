package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

func TestID_MustNewID_ReturnsValidVersion7(t *testing.T) {
	id := domain.MustNewID()

	if !id.IsValid() {
		t.Fatalf("MustNewID() returned an invalid ID: %s", id)
	}
	if got := id.Version(); got != 7 {
		t.Fatalf("Version() = %d, want 7", got)
	}
}

func TestID_NewID_GeneratesUniqueIDs(t *testing.T) {
	const n = 1000
	seen := make(map[domain.ID]struct{}, n)
	for i := 0; i < n; i++ {
		id, err := domain.NewID()
		if err != nil {
			t.Fatalf("NewID() error = %v", err)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("NewID() produced duplicate ID: %s", id)
		}
		seen[id] = struct{}{}
	}
}

func TestID_IsValid_ZeroValue_ReturnsFalse(t *testing.T) {
	var zero domain.ID
	if zero.IsValid() {
		t.Fatal("zero ID must be invalid")
	}
	if !zero.IsZero() {
		t.Fatal("zero ID must report IsZero() == true")
	}
}

func TestID_String_ReturnsCanonicalUUID(t *testing.T) {
	id := domain.MustNewID()
	got := id.String()

	if len(got) != 36 {
		t.Fatalf("String() length = %d, want 36 (%q)", len(got), got)
	}
	for _, i := range []int{8, 13, 18, 23} {
		if got[i] != '-' {
			t.Fatalf("String() = %q, want '-' at index %d", got, i)
		}
	}
	if got[14] != '7' {
		t.Fatalf("String() = %q, want version nibble '7' at index 14", got)
	}
}

func TestID_MarshalJSON_ReturnsQuotedUUID(t *testing.T) {
	id := domain.MustNewID()

	raw, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if want := `"` + id.String() + `"`; string(raw) != want {
		t.Fatalf("Marshal() = %s, want %s", raw, want)
	}
}

func TestID_UnmarshalJSON_RoundTrips(t *testing.T) {
	want := domain.MustNewID()
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got domain.ID
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got != want {
		t.Fatalf("round-trip = %s, want %s", got, want)
	}
}

func TestID_UnmarshalJSON_InvalidInput_ReturnsError(t *testing.T) {
	var id domain.ID
	if err := json.Unmarshal([]byte(`"zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz"`), &id); err == nil {
		t.Fatal("Unmarshal() of invalid UUID: want error, got nil")
	}
}

func TestID_UnmarshalJSON_WrongLength_ReturnsError(t *testing.T) {
	var id domain.ID
	if err := json.Unmarshal([]byte(`123`), &id); err == nil {
		t.Fatal("Unmarshal() of non-quoted input: want error, got nil")
	}
}

func TestParseID_Valid_ReturnsSameID(t *testing.T) {
	want := domain.MustNewID()

	got, err := domain.ParseID(want.String())
	if err != nil {
		t.Fatalf("ParseID() error = %v", err)
	}
	if got != want {
		t.Fatalf("ParseID() = %s, want %s", got, want)
	}
}

func TestParseID_Malformed_ReturnsError(t *testing.T) {
	cases := map[string]string{
		"wrong length":       "too-short",
		"wrong separators":   "123456780123-1234-1234-1234-123456789012",
		"non-hex characters": "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz",
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := domain.ParseID(input); err == nil {
				t.Fatalf("ParseID(%q): want error, got nil", input)
			}
		})
	}
}
