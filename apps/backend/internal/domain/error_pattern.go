package domain

import "strings"

// ErrorPatternCode is the error taxonomy the LLM assigns to a fragment
// (PRODUCT_DOMAIN §4.6). It lives in the root domain package because it is
// shared by the analysis and analytics bounded contexts (A3).
type ErrorPatternCode string

// Known error pattern codes.
const (
	ErrorPatternCodeInfinitiveConjugation   ErrorPatternCode = "infinitive_conjugation"
	ErrorPatternCodePassiveVoiceMisuse      ErrorPatternCode = "passive_voice_misuse"
	ErrorPatternCodeIdiomLiteralTranslation ErrorPatternCode = "idiom_literal_translation"
	ErrorPatternCodePrepositionInfinitive   ErrorPatternCode = "preposition_infinitive"
	ErrorPatternCodePronounPossession       ErrorPatternCode = "pronoun_possession"
	ErrorPatternCodeFalseFriend             ErrorPatternCode = "false_friend"
	ErrorPatternCodeWordOrder               ErrorPatternCode = "word_order"
	ErrorPatternCodeTenseAgreement          ErrorPatternCode = "tense_agreement"
)

// IsValid reports whether the code belongs to the known taxonomy.
func (c ErrorPatternCode) IsValid() bool {
	switch c {
	case ErrorPatternCodeInfinitiveConjugation,
		ErrorPatternCodePassiveVoiceMisuse,
		ErrorPatternCodeIdiomLiteralTranslation,
		ErrorPatternCodePrepositionInfinitive,
		ErrorPatternCodePronounPossession,
		ErrorPatternCodeFalseFriend,
		ErrorPatternCodeWordOrder,
		ErrorPatternCodeTenseAgreement:
		return true
	default:
		return false
	}
}

// ErrorPatternSeverity grades the impact of an error (PRODUCT_DOMAIN §4.6).
type ErrorPatternSeverity string

// Known severities.
const (
	ErrorPatternSeverityMinor    ErrorPatternSeverity = "minor"
	ErrorPatternSeverityModerate ErrorPatternSeverity = "moderate"
	ErrorPatternSeverityCritical ErrorPatternSeverity = "critical"
)

// IsValid reports whether the severity belongs to the known set.
func (s ErrorPatternSeverity) IsValid() bool {
	switch s {
	case ErrorPatternSeverityMinor, ErrorPatternSeverityModerate, ErrorPatternSeverityCritical:
		return true
	default:
		return false
	}
}

// ErrorPattern classifies a single error detected in a fragment (PRODUCT_DOMAIN §4.6).
type ErrorPattern struct {
	Code     ErrorPatternCode     `json:"code"`
	Severity ErrorPatternSeverity `json:"severity"`
	Note     string               `json:"note"`
}

// NewErrorPattern validates the code and severity and returns an ErrorPattern.
func NewErrorPattern(code ErrorPatternCode, severity ErrorPatternSeverity, note string) (ErrorPattern, error) {
	if !code.IsValid() {
		return ErrorPattern{}, &ValidationError{Field: "error_pattern.code", Message: "must be a known error pattern code"}
	}
	if !severity.IsValid() {
		return ErrorPattern{}, &ValidationError{Field: "error_pattern.severity", Message: "must be a known error pattern severity"}
	}
	return ErrorPattern{Code: code, Severity: severity, Note: strings.TrimSpace(note)}, nil
}
