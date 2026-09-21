package analysis_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
)

var testNow = time.Date(2026, time.September, 17, 12, 0, 0, 0, time.UTC)

func testFragments(t *testing.T) []analysis.Fragment {
	t.Helper()
	pattern, err := domain.NewErrorPattern(domain.ErrorPatternCodeTenseAgreement, domain.ErrorPatternSeverityModerate, "agreement")
	if err != nil {
		t.Fatalf("NewErrorPattern() error = %v", err)
	}
	return []analysis.Fragment{{
		SourceES:   "El gato duerme.",
		UserDraft:  "The cat sleep.",
		Correction: "The cat sleeps.",
		TargetVerbReview: analysis.TargetVerbReview{
			Verb:         "sleep",
			CorrectForm:  "sleeps",
			Rule:         "tercera persona singular",
			Why:          "el sujeto es singular",
			ESContrast:   "en español no cambia",
			Alternatives: []string{"sleeps"},
		},
		LexicalClarification: analysis.LexicalClarification{
			Term:    "cat",
			Meaning: "gato",
		},
		GrammarExplanation: analysis.GrammarExplanation{
			RuleName:       "tercera persona singular",
			Explanation:    "el verbo añade -s",
			Construction:   "verbo + -s",
			Counterexample: "sleep -> sleeps",
			Exception:      "irregulares",
			ESContrast:     "no aplica en español",
		},
		ErrorPatterns: []domain.ErrorPattern{pattern},
	}}
}

func newTestAnalysis(t *testing.T) *analysis.Analysis {
	t.Helper()
	a, err := analysis.NewAnalysis(domain.MustNewID(), "gpt-4o", "2024-05-13", testNow)
	if err != nil {
		t.Fatalf("NewAnalysis() error = %v", err)
	}
	return a
}

func assertInvalidState(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("want *domain.InvalidStateError, got nil")
	}
	var target *domain.InvalidStateError
	if !errors.As(err, &target) {
		t.Fatalf("error = %T, want *domain.InvalidStateError", err)
	}
}

func TestNewAnalysis_Valid_ReturnsPendingAnalysis(t *testing.T) {
	practiceID := domain.MustNewID()

	a, err := analysis.NewAnalysis(practiceID, "gpt-4o", "2024-05-13", testNow)
	if err != nil {
		t.Fatalf("NewAnalysis() error = %v", err)
	}
	if !a.ID.IsValid() {
		t.Fatalf("ID = %s, want a valid UUID v7", a.ID)
	}
	if a.PracticeID != practiceID {
		t.Fatalf("PracticeID = %s, want %s", a.PracticeID, practiceID)
	}
	if a.Model != "gpt-4o" {
		t.Fatalf("Model = %q, want %q", a.Model, "gpt-4o")
	}
	if a.ModelVersion != "2024-05-13" {
		t.Fatalf("ModelVersion = %q, want %q", a.ModelVersion, "2024-05-13")
	}
	if a.Status != analysis.AnalysisStatusPending {
		t.Fatalf("Status = %q, want %q", a.Status, analysis.AnalysisStatusPending)
	}
	if len(a.Fragments) != 0 {
		t.Fatalf("Fragments = %+v, want empty", a.Fragments)
	}
	if !a.CreatedAt.Equal(testNow) {
		t.Fatalf("CreatedAt = %v, want %v", a.CreatedAt, testNow)
	}
}

func TestNewAnalysis_GeneratesUniqueIDs(t *testing.T) {
	a := newTestAnalysis(t)
	b := newTestAnalysis(t)
	if a.ID == b.ID {
		t.Fatalf("two analyses share ID %s", a.ID)
	}
}

func TestNewAnalysis_MarshalJSON_UsesSnakeCase(t *testing.T) {
	a := newTestAnalysis(t)

	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"id", "practice_id", "fragments", "model", "model_version", "status", "created_at"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
	if len(got) != 7 {
		t.Fatalf("payload %s has %d keys, want exactly 7", raw, len(got))
	}
}

func TestAnalysis_Complete_PendingWithFragments_TransitionsToCompleted(t *testing.T) {
	a := newTestAnalysis(t)
	fragments := testFragments(t)

	if err := a.Complete(fragments); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if a.Status != analysis.AnalysisStatusCompleted {
		t.Fatalf("Status = %q, want %q", a.Status, analysis.AnalysisStatusCompleted)
	}
	if len(a.Fragments) != len(fragments) {
		t.Fatalf("Fragments = %+v, want %+v", a.Fragments, fragments)
	}
}

func TestAnalysis_Complete_PendingWithEmptyFragments_ReturnsValidationError(t *testing.T) {
	a := newTestAnalysis(t)

	err := a.Complete(nil)
	if err == nil {
		t.Fatal("Complete() with empty fragments: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("Complete() error = %T, want *domain.ValidationError", err)
	}
	if target.Field != "fragments" {
		t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "fragments")
	}
}

func TestAnalysis_Complete_NotPending_ReturnsInvalidStateError(t *testing.T) {
	a := newTestAnalysis(t)
	_ = a.Complete(testFragments(t))

	assertInvalidState(t, a.Complete(testFragments(t)))
}

func TestAnalysis_Fail_Pending_TransitionsToFailed(t *testing.T) {
	a := newTestAnalysis(t)

	if err := a.Fail(); err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	if a.Status != analysis.AnalysisStatusFailed {
		t.Fatalf("Status = %q, want %q", a.Status, analysis.AnalysisStatusFailed)
	}
}

func TestAnalysis_Fail_NotPending_ReturnsInvalidStateError(t *testing.T) {
	a := newTestAnalysis(t)
	_ = a.Fail()

	assertInvalidState(t, a.Fail())
}
