package practice

import (
	"time"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// Practice is the aggregate root of the writing exercise (PRODUCT_DOMAIN §4.2.2).
type Practice struct {
	ID          domain.ID      `json:"id"`
	UserID      domain.ID      `json:"user_id"`
	SourceText  SourceText     `json:"source_text"`
	DraftText   DraftText      `json:"draft_text"`
	TargetRules []TargetRule   `json:"target_rules"`
	Status      PracticeStatus `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`

	// DeletedAt marks a soft-delete (A8). It is internal state consumed by the
	// purge-raw-data job and is never exposed on the wire.
	DeletedAt *time.Time `json:"-"`
}

// generateID is a seam for deterministic tests; it defaults to domain.NewID.
var generateID = domain.NewID

// NewPractice creates a Practice in draft status, generating its UUID v7.
// TargetRules must contain at least one rule.
func NewPractice(userID domain.ID, source SourceText, draft DraftText, rules []TargetRule, createdAt time.Time) (*Practice, error) {
	if len(rules) == 0 {
		return nil, &domain.ValidationError{Field: "target_rules", Message: "must contain at least one rule"}
	}
	if len(rules) > maxTargetRules {
		return nil, &domain.ValidationError{Field: "target_rules", Message: "must contain at most 5 rules"}
	}

	id, err := generateID()
	if err != nil {
		return nil, err
	}

	return &Practice{
		ID:          id,
		UserID:      userID,
		SourceText:  source,
		DraftText:   draft,
		TargetRules: rules,
		Status:      PracticeStatusDraft,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}, nil
}

// StartAnalysis transitions draft -> analyzing. Any other state is invalid.
func (p *Practice) StartAnalysis(now time.Time) error {
	if p.Status != PracticeStatusDraft {
		return &domain.InvalidStateError{Field: "status", Message: "must be draft to start analysis"}
	}
	p.Status = PracticeStatusAnalyzing
	p.UpdatedAt = now
	return nil
}

// MarkCompleted transitions analyzing -> completed. Any other state is invalid.
func (p *Practice) MarkCompleted(now time.Time) error {
	if p.Status != PracticeStatusAnalyzing {
		return &domain.InvalidStateError{Field: "status", Message: "must be analyzing to be completed"}
	}
	p.Status = PracticeStatusCompleted
	p.UpdatedAt = now
	return nil
}

// MarkFailed transitions analyzing -> failed. Any other state is invalid.
func (p *Practice) MarkFailed(now time.Time) error {
	if p.Status != PracticeStatusAnalyzing {
		return &domain.InvalidStateError{Field: "status", Message: "must be analyzing to fail"}
	}
	p.Status = PracticeStatusFailed
	p.UpdatedAt = now
	return nil
}

// Edit applies a partial update (PATCH semantics) to a draft practice. A nil
// field is left unchanged; a non-nil but empty rule set is invalid.
func (p *Practice) Edit(source *SourceText, draft *DraftText, rules []TargetRule, now time.Time) error {
	if p.Status != PracticeStatusDraft {
		return &domain.InvalidStateError{Field: "status", Message: "must be draft to be edited"}
	}
	if rules != nil && len(rules) == 0 {
		return &domain.ValidationError{Field: "target_rules", Message: "must contain at least one rule"}
	}
	if len(rules) > maxTargetRules {
		return &domain.ValidationError{Field: "target_rules", Message: "must contain at most 5 rules"}
	}

	changed := false
	if source != nil {
		p.SourceText = *source
		changed = true
	}
	if draft != nil {
		p.DraftText = *draft
		changed = true
	}
	if rules != nil {
		p.TargetRules = rules
		changed = true
	}
	if changed {
		p.UpdatedAt = now
	}
	return nil
}

// Delete soft-deletes the practice (A8), recording when it happened so the
// purge-raw-data job can enforce retention. Deletion is blocked while analyzing.
func (p *Practice) Delete(now time.Time) error {
	if p.Status == PracticeStatusAnalyzing {
		return &domain.InvalidStateError{Field: "status", Message: "must not be analyzing to be deleted"}
	}
	if p.DeletedAt != nil {
		return &domain.InvalidStateError{Field: "deleted_at", Message: "practice is already deleted"}
	}
	p.DeletedAt = &now
	p.UpdatedAt = now
	return nil
}
