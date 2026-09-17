package practice

import (
	"strings"

	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
)

// TargetRule is a grammar goal the user wants to practice (PRODUCT_DOMAIN §4.3).
type TargetRule struct {
	Verb  string `json:"verb"`
	Tense string `json:"tense"`
	Note  string `json:"note"`
}

// NewTargetRule validates verb and returns a TargetRule value object.
func NewTargetRule(verb, tense, note string) (TargetRule, error) {
	trimmedVerb := strings.TrimSpace(verb)
	if trimmedVerb == "" {
		return TargetRule{}, &domain.ValidationError{Field: "target_rules.verb", Message: "must not be empty"}
	}
	return TargetRule{
		Verb:  trimmedVerb,
		Tense: strings.TrimSpace(tense),
		Note:  strings.TrimSpace(note),
	}, nil
}
