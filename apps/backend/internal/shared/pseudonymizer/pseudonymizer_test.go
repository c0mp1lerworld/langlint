package pseudonymizer

import (
	"strings"
	"testing"
)

func TestNew_EmptySecret_ReturnsError(t *testing.T) {
	if _, err := New(""); err == nil {
		t.Fatal("New() error = nil, want error for empty secret")
	}
}

func TestPseudonymize_SameInputAndSecret_IsDeterministic(t *testing.T) {
	p, err := New("test-secret")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	first := p.Pseudonymize("0192f3a0-0000-7000-8000-000000000000")
	second := p.Pseudonymize("0192f3a0-0000-7000-8000-000000000000")
	if first != second {
		t.Fatalf("Pseudonymize() = %q then %q, want equal", first, second)
	}
}

func TestPseudonymize_DifferentSecret_ChangesOutput(t *testing.T) {
	a, err := New("secret-a")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	b, err := New("secret-b")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	id := "0192f3a0-0000-7000-8000-000000000000"
	if a.Pseudonymize(id) == b.Pseudonymize(id) {
		t.Fatal("pseudonyms under different secrets are equal, want different")
	}
}

func TestPseudonymize_DifferentInput_ChangesOutput(t *testing.T) {
	p, err := New("test-secret")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if p.Pseudonymize("user-a") == p.Pseudonymize("user-b") {
		t.Fatal("pseudonyms of different identifiers are equal, want different")
	}
}

func TestPseudonymize_ReturnsHexHMACSHA256(t *testing.T) {
	p, err := New("test-secret")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got := p.Pseudonymize("user-a")
	if len(got) != 64 {
		t.Fatalf("len(Pseudonymize()) = %d, want 64 (sha256 hex)", len(got))
	}
	if strings.ToLower(got) != got {
		t.Fatalf("Pseudonymize() = %q, want lowercase hex", got)
	}
}
