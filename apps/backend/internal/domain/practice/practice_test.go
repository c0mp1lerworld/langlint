package practice_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
)

var testNow = time.Date(2026, time.September, 17, 12, 0, 0, 0, time.UTC)

func testSourceText(t *testing.T) practice.SourceText {
	t.Helper()
	source, err := practice.NewSourceText("El gato duerme.")
	if err != nil {
		t.Fatalf("NewSourceText() error = %v", err)
	}
	return source
}

func testDraftText(t *testing.T) practice.DraftText {
	t.Helper()
	draft, err := practice.NewDraftText("The cat sleep.")
	if err != nil {
		t.Fatalf("NewDraftText() error = %v", err)
	}
	return draft
}

func testTargetRules(t *testing.T) []practice.TargetRule {
	t.Helper()
	rule, err := practice.NewTargetRule("sleep", "present", "")
	if err != nil {
		t.Fatalf("NewTargetRule() error = %v", err)
	}
	return []practice.TargetRule{rule}
}

func newTestPractice(t *testing.T) *practice.Practice {
	t.Helper()
	p, err := practice.NewPractice(domain.MustNewID(), testSourceText(t), testDraftText(t), testTargetRules(t), testNow)
	if err != nil {
		t.Fatalf("NewPractice() error = %v", err)
	}
	return p
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

func TestNewPractice_Valid_ReturnsDraftPractice(t *testing.T) {
	userID := domain.MustNewID()
	source := testSourceText(t)
	draft := testDraftText(t)
	rules := testTargetRules(t)

	p, err := practice.NewPractice(userID, source, draft, rules, testNow)
	if err != nil {
		t.Fatalf("NewPractice() error = %v", err)
	}
	if !p.ID.IsValid() {
		t.Fatalf("ID = %s, want a valid UUID v7", p.ID)
	}
	if p.UserID != userID {
		t.Fatalf("UserID = %s, want %s", p.UserID, userID)
	}
	if p.SourceText != source {
		t.Fatalf("SourceText = %q, want %q", p.SourceText, source)
	}
	if p.DraftText != draft {
		t.Fatalf("DraftText = %q, want %q", p.DraftText, draft)
	}
	if len(p.TargetRules) != 1 || p.TargetRules[0] != rules[0] {
		t.Fatalf("TargetRules = %+v, want %+v", p.TargetRules, rules)
	}
	if p.Status != practice.PracticeStatusDraft {
		t.Fatalf("Status = %q, want %q", p.Status, practice.PracticeStatusDraft)
	}
	if !p.CreatedAt.Equal(testNow) {
		t.Fatalf("CreatedAt = %v, want %v", p.CreatedAt, testNow)
	}
	if !p.UpdatedAt.Equal(testNow) {
		t.Fatalf("UpdatedAt = %v, want %v", p.UpdatedAt, testNow)
	}
	if p.DeletedAt != nil {
		t.Fatalf("DeletedAt = %v, want nil", p.DeletedAt)
	}
}

func TestNewPractice_GeneratesUniqueIDs(t *testing.T) {
	a := newTestPractice(t)
	b := newTestPractice(t)
	if a.ID == b.ID {
		t.Fatalf("two practices share ID %s", a.ID)
	}
}

func TestNewPractice_EmptyTargetRules_ReturnsValidationError(t *testing.T) {
	_, err := practice.NewPractice(domain.MustNewID(), testSourceText(t), testDraftText(t), nil, testNow)
	if err == nil {
		t.Fatal("NewPractice() with no target rules: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewPractice() error = %T, want *domain.ValidationError", err)
	}
	if target.Field != "target_rules" {
		t.Fatalf("ValidationError.Field = %q, want %q", target.Field, "target_rules")
	}
}

// manyTargetRules builds n distinct valid rules for the limit tests.
func manyTargetRules(t *testing.T, n int) []practice.TargetRule {
	t.Helper()
	rules := make([]practice.TargetRule, 0, n)
	for i := 0; i < n; i++ {
		rule, err := practice.NewTargetRule("verb", "present", "")
		if err != nil {
			t.Fatalf("NewTargetRule() error = %v", err)
		}
		rules = append(rules, rule)
	}
	return rules
}

func TestNewPractice_TooManyTargetRules_ReturnsValidationError(t *testing.T) {
	_, err := practice.NewPractice(domain.MustNewID(), testSourceText(t), testDraftText(t), manyTargetRules(t, 6), testNow)
	if err == nil {
		t.Fatal("NewPractice() with 6 target rules: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("NewPractice() error = %T, want *domain.ValidationError", err)
	}
	if target.Field != "target_rules" {
		t.Fatalf("ValidationError.Field = %q, want target_rules", target.Field)
	}
}

func TestNewPractice_MaxTargetRules_IsAccepted(t *testing.T) {
	if _, err := practice.NewPractice(domain.MustNewID(), testSourceText(t), testDraftText(t), manyTargetRules(t, 5), testNow); err != nil {
		t.Fatalf("NewPractice() with 5 target rules error = %v, want nil", err)
	}
}

func TestNewPractice_MarshalJSON_UsesSnakeCase(t *testing.T) {
	p := newTestPractice(t)

	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"id", "user_id", "source_text", "draft_text", "target_rules", "status", "created_at", "updated_at"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload %s missing snake_case key %q", raw, key)
		}
	}
	if _, ok := got["deleted_at"]; ok {
		t.Fatalf("payload %s must not expose internal deleted_at", raw)
	}
	if len(got) != 8 {
		t.Fatalf("payload %s has %d keys, want exactly 8", raw, len(got))
	}
}

func TestPractice_StartAnalysis_Draft_TransitionsToAnalyzing(t *testing.T) {
	p := newTestPractice(t)
	later := testNow.Add(time.Hour)

	if err := p.StartAnalysis(later); err != nil {
		t.Fatalf("StartAnalysis() error = %v", err)
	}
	if p.Status != practice.PracticeStatusAnalyzing {
		t.Fatalf("Status = %q, want %q", p.Status, practice.PracticeStatusAnalyzing)
	}
	if !p.UpdatedAt.Equal(later) {
		t.Fatalf("UpdatedAt = %v, want %v", p.UpdatedAt, later)
	}
}

func TestPractice_StartAnalysis_NotDraft_ReturnsInvalidStateError(t *testing.T) {
	p := newTestPractice(t)
	_ = p.StartAnalysis(testNow)
	_ = p.MarkCompleted(testNow)

	assertInvalidState(t, p.StartAnalysis(testNow))
}

func TestPractice_MarkCompleted_Analyzing_TransitionsToCompleted(t *testing.T) {
	p := newTestPractice(t)
	_ = p.StartAnalysis(testNow)
	later := testNow.Add(time.Hour)

	if err := p.MarkCompleted(later); err != nil {
		t.Fatalf("MarkCompleted() error = %v", err)
	}
	if p.Status != practice.PracticeStatusCompleted {
		t.Fatalf("Status = %q, want %q", p.Status, practice.PracticeStatusCompleted)
	}
	if !p.UpdatedAt.Equal(later) {
		t.Fatalf("UpdatedAt = %v, want %v", p.UpdatedAt, later)
	}
}

func TestPractice_MarkCompleted_NotAnalyzing_ReturnsInvalidStateError(t *testing.T) {
	p := newTestPractice(t)

	assertInvalidState(t, p.MarkCompleted(testNow))
}

func TestPractice_MarkFailed_Analyzing_TransitionsToFailed(t *testing.T) {
	p := newTestPractice(t)
	_ = p.StartAnalysis(testNow)
	later := testNow.Add(time.Hour)

	if err := p.MarkFailed(later); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
	if p.Status != practice.PracticeStatusFailed {
		t.Fatalf("Status = %q, want %q", p.Status, practice.PracticeStatusFailed)
	}
	if !p.UpdatedAt.Equal(later) {
		t.Fatalf("UpdatedAt = %v, want %v", p.UpdatedAt, later)
	}
}

func TestPractice_MarkFailed_NotAnalyzing_ReturnsInvalidStateError(t *testing.T) {
	p := newTestPractice(t)

	assertInvalidState(t, p.MarkFailed(testNow))
}

func TestPractice_Edit_Draft_UpdatesProvidedFields(t *testing.T) {
	p := newTestPractice(t)
	newSource, err := practice.NewSourceText("La casa es grande.")
	if err != nil {
		t.Fatalf("NewSourceText() error = %v", err)
	}
	newDraft, err := practice.NewDraftText("The house is big.")
	if err != nil {
		t.Fatalf("NewDraftText() error = %v", err)
	}
	newRule, err := practice.NewTargetRule("be", "present", "irregular")
	if err != nil {
		t.Fatalf("NewTargetRule() error = %v", err)
	}
	later := testNow.Add(time.Hour)

	if err := p.Edit(&newSource, &newDraft, []practice.TargetRule{newRule}, later); err != nil {
		t.Fatalf("Edit() error = %v", err)
	}
	if p.SourceText != newSource {
		t.Fatalf("SourceText = %q, want %q", p.SourceText, newSource)
	}
	if p.DraftText != newDraft {
		t.Fatalf("DraftText = %q, want %q", p.DraftText, newDraft)
	}
	if len(p.TargetRules) != 1 || p.TargetRules[0] != newRule {
		t.Fatalf("TargetRules = %+v, want %+v", p.TargetRules, newRule)
	}
	if !p.UpdatedAt.Equal(later) {
		t.Fatalf("UpdatedAt = %v, want %v", p.UpdatedAt, later)
	}
}

func TestPractice_Edit_Draft_PartialUpdate_LeavesOtherFields(t *testing.T) {
	p := newTestPractice(t)
	originalDraft := p.DraftText
	originalRules := p.TargetRules
	newSource, err := practice.NewSourceText("Otro texto.")
	if err != nil {
		t.Fatalf("NewSourceText() error = %v", err)
	}
	later := testNow.Add(time.Hour)

	if err := p.Edit(&newSource, nil, nil, later); err != nil {
		t.Fatalf("Edit() error = %v", err)
	}
	if p.SourceText != newSource {
		t.Fatalf("SourceText = %q, want %q", p.SourceText, newSource)
	}
	if p.DraftText != originalDraft {
		t.Fatalf("DraftText = %q, want it unchanged %q", p.DraftText, originalDraft)
	}
	if len(p.TargetRules) != len(originalRules) || p.TargetRules[0] != originalRules[0] {
		t.Fatalf("TargetRules = %+v, want it unchanged %+v", p.TargetRules, originalRules)
	}
	if !p.UpdatedAt.Equal(later) {
		t.Fatalf("UpdatedAt = %v, want %v", p.UpdatedAt, later)
	}
}

func TestPractice_Edit_Draft_NoFields_IsNoOp(t *testing.T) {
	p := newTestPractice(t)

	if err := p.Edit(nil, nil, nil, testNow.Add(time.Hour)); err != nil {
		t.Fatalf("Edit() error = %v", err)
	}
	if !p.UpdatedAt.Equal(testNow) {
		t.Fatalf("UpdatedAt = %v, want it unchanged %v", p.UpdatedAt, testNow)
	}
}

func TestPractice_Edit_NotDraft_ReturnsInvalidStateError(t *testing.T) {
	p := newTestPractice(t)
	_ = p.StartAnalysis(testNow)
	_ = p.MarkCompleted(testNow)

	assertInvalidState(t, p.Edit(nil, nil, nil, testNow))
}

func TestPractice_Edit_EmptyRules_ReturnsValidationError(t *testing.T) {
	p := newTestPractice(t)

	err := p.Edit(nil, nil, []practice.TargetRule{}, testNow)
	if err == nil {
		t.Fatal("Edit() with empty target rules: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("Edit() error = %T, want *domain.ValidationError", err)
	}
}

func TestPractice_Edit_TooManyRules_ReturnsValidationError(t *testing.T) {
	p := newTestPractice(t)

	err := p.Edit(nil, nil, manyTargetRules(t, 6), testNow)
	if err == nil {
		t.Fatal("Edit() with 6 target rules: want error, got nil")
	}
	var target *domain.ValidationError
	if !errors.As(err, &target) {
		t.Fatalf("Edit() error = %T, want *domain.ValidationError", err)
	}
}

func TestPractice_Delete_Draft_SetsDeletedAt(t *testing.T) {
	p := newTestPractice(t)
	later := testNow.Add(time.Hour)

	if err := p.Delete(later); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if p.DeletedAt == nil {
		t.Fatal("DeletedAt = nil, want a soft-delete timestamp")
	}
	if !p.DeletedAt.Equal(later) {
		t.Fatalf("DeletedAt = %v, want %v", p.DeletedAt, later)
	}
	if !p.UpdatedAt.Equal(later) {
		t.Fatalf("UpdatedAt = %v, want %v", p.UpdatedAt, later)
	}
}

func TestPractice_Delete_Analyzing_ReturnsInvalidStateError(t *testing.T) {
	p := newTestPractice(t)
	_ = p.StartAnalysis(testNow)

	assertInvalidState(t, p.Delete(testNow))
}

func TestPractice_Delete_AlreadyDeleted_ReturnsInvalidStateError(t *testing.T) {
	p := newTestPractice(t)
	_ = p.Delete(testNow)

	assertInvalidState(t, p.Delete(testNow.Add(time.Hour)))
}
