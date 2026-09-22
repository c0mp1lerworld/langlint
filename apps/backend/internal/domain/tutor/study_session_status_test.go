package tutor_test

import (
	"testing"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain/tutor"
)

func TestStudySessionStatus_IsValid_KnownAndUnknownValues(t *testing.T) {
	cases := []struct {
		status tutor.StudySessionStatus
		want   bool
	}{
		{tutor.StudySessionStatusGenerated, true},
		{tutor.StudySessionStatusActive, true},
		{tutor.StudySessionStatusCompleted, true},
		{tutor.StudySessionStatus(""), false},
		{tutor.StudySessionStatus("archived"), false},
	}
	for _, c := range cases {
		if got := c.status.IsValid(); got != c.want {
			t.Fatalf("%q.IsValid() = %v, want %v", c.status, got, c.want)
		}
	}
}

func TestStudySessionStatus_String_ReturnsWireValue(t *testing.T) {
	if got := tutor.StudySessionStatusActive.String(); got != "active" {
		t.Fatalf("String() = %q, want %q", got, "active")
	}
}
