package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/services"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analytics"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/identity"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/httpx"
)

// retryAfterSeconds is the polling hint sent with the 202 of analyze.
const retryAfterSeconds = "2"

// Server implements ServerInterface over the application services. It is the
// only place where wire and domain types are converted (AP-MR2).
type Server struct {
	practices *services.PracticeService
	analytics *services.AnalyticsService
	identity  *services.IdentityService
}

var _ ServerInterface = (*Server)(nil)

// NewServer wires the HTTP handlers through the application services.
func NewServer(practices *services.PracticeService, analytics *services.AnalyticsService, identity *services.IdentityService) *Server {
	return &Server{practices: practices, analytics: analytics, identity: identity}
}

// CreatePractice handles POST /practices.
func (s *Server) CreatePractice(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.UserFrom(r.Context())
	if !ok {
		writeDomainError(w, &domain.InternalError{Field: "user", Message: "missing user context"})
		return
	}

	var req CreatePracticeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeDomainError(w, &domain.ValidationError{Field: "body", Message: "malformed JSON"})
		return
	}

	p, err := s.practices.Create(r.Context(), userID, services.PracticeInput{
		SourceText:  req.SourceText,
		DraftText:   req.DraftText,
		TargetRules: targetRuleInputs(req.TargetRules),
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, practiceToWire(p))
}

// ListPractices handles GET /practices.
func (s *Server) ListPractices(w http.ResponseWriter, r *http.Request, params ListPracticesParams) {
	userID, ok := httpx.UserFrom(r.Context())
	if !ok {
		writeDomainError(w, &domain.InternalError{Field: "user", Message: "missing user context"})
		return
	}

	limit, offset := page(params.Limit, params.Offset)
	items, total, err := s.practices.List(r.Context(), userID, limit, offset)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	wire := make([]Practice, 0, len(items))
	for i := range items {
		wire = append(wire, practiceToWire(&items[i]))
	}
	writeJSON(w, http.StatusOK, PracticeList{Items: wire, Total: total})
}

// GetPractice handles GET /practices/{practiceId}.
func (s *Server) GetPractice(w http.ResponseWriter, r *http.Request, practiceID PracticeId) {
	userID, ok := httpx.UserFrom(r.Context())
	if !ok {
		writeDomainError(w, &domain.InternalError{Field: "user", Message: "missing user context"})
		return
	}

	id, err := toDomainID(practiceID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	p, a, err := s.practices.Get(r.Context(), userID, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	detail := practiceDetailWire(p)
	if a != nil {
		detail.Analysis = analysisToWire(a)
	}
	writeJSON(w, http.StatusOK, detail)
}

// UpdatePractice handles PATCH /practices/{practiceId}.
func (s *Server) UpdatePractice(w http.ResponseWriter, r *http.Request, practiceID PracticeId) {
	userID, ok := httpx.UserFrom(r.Context())
	if !ok {
		writeDomainError(w, &domain.InternalError{Field: "user", Message: "missing user context"})
		return
	}

	id, err := toDomainID(practiceID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	var req UpdatePracticeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeDomainError(w, &domain.ValidationError{Field: "body", Message: "malformed JSON"})
		return
	}

	in := services.PracticeUpdateInput{SourceText: req.SourceText, DraftText: req.DraftText}
	if req.TargetRules != nil {
		rules := targetRuleInputs(*req.TargetRules)
		in.TargetRules = &rules
	}

	p, err := s.practices.Update(r.Context(), userID, id, in)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, practiceToWire(p))
}

// DeletePractice handles DELETE /practices/{practiceId}.
func (s *Server) DeletePractice(w http.ResponseWriter, r *http.Request, practiceID PracticeId) {
	userID, ok := httpx.UserFrom(r.Context())
	if !ok {
		writeDomainError(w, &domain.InternalError{Field: "user", Message: "missing user context"})
		return
	}

	id, err := toDomainID(practiceID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	if err := s.practices.Delete(r.Context(), userID, id); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AnalyzePractice handles POST /practices/{practiceId}/analyze.
func (s *Server) AnalyzePractice(w http.ResponseWriter, r *http.Request, practiceID PracticeId) {
	userID, ok := httpx.UserFrom(r.Context())
	if !ok {
		writeDomainError(w, &domain.InternalError{Field: "user", Message: "missing user context"})
		return
	}

	id, err := toDomainID(practiceID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	if err := s.practices.Analyze(r.Context(), userID, id); err != nil {
		writeDomainError(w, err)
		return
	}
	w.Header().Set("Retry-After", retryAfterSeconds)
	w.WriteHeader(http.StatusAccepted)
}

// GetErrorPatternStats handles GET /analytics/error-patterns.
func (s *Server) GetErrorPatternStats(w http.ResponseWriter, r *http.Request, params GetErrorPatternStatsParams) {
	userID, ok := httpx.UserFrom(r.Context())
	if !ok {
		writeDomainError(w, &domain.InternalError{Field: "user", Message: "missing user context"})
		return
	}

	window := analytics.WindowWeek
	if params.Window != nil {
		window = analytics.Window(*params.Window)
		if !window.IsValid() {
			writeDomainError(w, &domain.ValidationError{Field: "window", Message: "must be day, week or month"})
			return
		}
	}

	metrics, err := s.analytics.ErrorPatternStats(r.Context(), userID, window)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	patterns := make([]ErrorPatternAggregate, 0, len(metrics))
	for _, m := range metrics {
		patterns = append(patterns, ErrorPatternAggregate{
			Code:       ErrorPatternCode(m.Code),
			Count:      m.Count,
			LastSeenAt: m.LastSeenAt,
		})
	}
	writeJSON(w, http.StatusOK, ErrorPatternStats{Window: Window(window), Patterns: patterns})
}

// GetProgressSeries handles GET /analytics/progress. Deferred to Fase 6
// (refresh-aggregates).
func (s *Server) GetProgressSeries(w http.ResponseWriter, _ *http.Request, _ GetProgressSeriesParams) {
	writeNotImplemented(w)
}

// GetAccessLog handles GET /me/access-log: the user's append-only audit (A9).
func (s *Server) GetAccessLog(w http.ResponseWriter, r *http.Request, params GetAccessLogParams) {
	userID, ok := httpx.UserFrom(r.Context())
	if !ok {
		writeDomainError(w, &domain.InternalError{Field: "user", Message: "missing user context"})
		return
	}

	limit, offset := page(params.Limit, params.Offset)
	events, total, err := s.identity.AccessLog(r.Context(), userID, limit, offset)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	items := make([]AccessLogEntry, 0, len(events))
	for _, event := range events {
		items = append(items, accessEventToWire(event))
	}
	writeJSON(w, http.StatusOK, AccessLog{Items: items, Total: total})
}

// DeleteData handles DELETE /me/data: it records a deletion request with the
// 30-day grace period (A9). A repeated call while one is pending is idempotent.
func (s *Server) DeleteData(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.UserFrom(r.Context())
	if !ok {
		writeDomainError(w, &domain.InternalError{Field: "user", Message: "missing user context"})
		return
	}

	if err := s.identity.RequestDeletion(r.Context(), userID); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// ExportData handles GET /me/data/export: the user's data portability (A9).
func (s *Server) ExportData(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpx.UserFrom(r.Context())
	if !ok {
		writeDomainError(w, &domain.InternalError{Field: "user", Message: "missing user context"})
		return
	}

	items, err := s.identity.Export(r.Context(), userID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	practices := make([]Practice, 0, len(items))
	for i := range items {
		practices = append(practices, practiceToWire(&items[i]))
	}
	writeJSON(w, http.StatusOK, DataExport{
		Email:       openapi_types.Email(s.identity.Email().String()),
		GeneratedAt: time.Now().UTC(),
		Practices:   practices,
		UserId:      openapi_types.UUID(userID),
	})
}

// writeJSON encodes body as the JSON response.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// toDomainID converts a parsed path UUID into a domain id.
func toDomainID(id openapi_types.UUID) (domain.ID, error) {
	parsed, err := domain.ParseID(id.String())
	if err != nil {
		return domain.ID{}, &domain.ValidationError{Field: "practice_id", Message: "invalid uuid"}
	}
	return parsed, nil
}

// page applies the pagination defaults of the contract (limit 20, offset 0).
func page(limit *Limit, offset *Offset) (int, int) {
	l, o := 20, 0
	if limit != nil {
		l = *limit
	}
	if offset != nil {
		o = *offset
	}
	return l, o
}

// targetRuleInputs converts wire rules into service inputs.
func targetRuleInputs(rules []TargetRule) []services.TargetRuleInput {
	out := make([]services.TargetRuleInput, 0, len(rules))
	for _, rule := range rules {
		in := services.TargetRuleInput{Verb: rule.Verb}
		if rule.Tense != nil {
			in.Tense = *rule.Tense
		}
		if rule.Note != nil {
			in.Note = *rule.Note
		}
		out = append(out, in)
	}
	return out
}

// practiceToWire converts the practice aggregate to its wire representation.
func practiceToWire(p *practice.Practice) Practice {
	return Practice{
		Id:          openapi_types.UUID(p.ID),
		UserId:      openapi_types.UUID(p.UserID),
		SourceText:  p.SourceText.String(),
		DraftText:   p.DraftText.String(),
		TargetRules: targetRulesToWire(p.TargetRules),
		Status:      PracticeStatus(p.Status),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// practiceDetailWire converts the practice aggregate to the detail wire type.
func practiceDetailWire(p *practice.Practice) PracticeDetail {
	return PracticeDetail{
		Id:          openapi_types.UUID(p.ID),
		UserId:      openapi_types.UUID(p.UserID),
		SourceText:  p.SourceText.String(),
		DraftText:   p.DraftText.String(),
		TargetRules: targetRulesToWire(p.TargetRules),
		Status:      PracticeStatus(p.Status),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// targetRulesToWire converts domain target rules to their wire representation.
func targetRulesToWire(rules []practice.TargetRule) []TargetRule {
	out := make([]TargetRule, 0, len(rules))
	for _, rule := range rules {
		wire := TargetRule{Verb: rule.Verb}
		if rule.Tense != "" {
			tense := rule.Tense
			wire.Tense = &tense
		}
		if rule.Note != "" {
			note := rule.Note
			wire.Note = &note
		}
		out = append(out, wire)
	}
	return out
}

// analysisToWire converts the analysis aggregate to its wire representation.
func analysisToWire(a *analysis.Analysis) *Analysis {
	fragments := make([]Fragment, 0, len(a.Fragments))
	for _, fragment := range a.Fragments {
		fragments = append(fragments, fragmentToWire(fragment))
	}
	return &Analysis{
		Id:           openapi_types.UUID(a.ID),
		PracticeId:   openapi_types.UUID(a.PracticeID),
		Model:        a.Model,
		ModelVersion: a.ModelVersion,
		Status:       AnalysisStatus(a.Status),
		Fragments:    fragments,
	}
}

// fragmentToWire converts an analysis fragment to its wire representation.
func fragmentToWire(f analysis.Fragment) Fragment {
	patterns := make([]ErrorPattern, 0, len(f.ErrorPatterns))
	for _, pattern := range f.ErrorPatterns {
		patterns = append(patterns, errorPatternToWire(pattern))
	}
	return Fragment{
		SourceEs:             f.SourceES,
		UserDraft:            f.UserDraft,
		Correction:           f.Correction,
		TargetVerbReview:     targetVerbReviewToWire(f.TargetVerbReview),
		LexicalClarification: lexicalClarificationToWire(f.LexicalClarification),
		GrammarExplanation:   grammarExplanationToWire(f.GrammarExplanation),
		ErrorPatterns:        patterns,
	}
}

// targetVerbReviewToWire converts the structured verb review to the wire type.
func targetVerbReviewToWire(r analysis.TargetVerbReview) TargetVerbReview {
	return TargetVerbReview{
		Verb:         r.Verb,
		CorrectForm:  r.CorrectForm,
		Rule:         r.Rule,
		Why:          r.Why,
		EsContrast:   r.ESContrast,
		Alternatives: nonNilStrings(r.Alternatives),
	}
}

// lexicalClarificationToWire converts the structured lexical note to the wire type.
func lexicalClarificationToWire(l analysis.LexicalClarification) LexicalClarification {
	return LexicalClarification{
		Term:         l.Term,
		Meaning:      l.Meaning,
		WhyWrong:     l.WhyWrong,
		Alternatives: nonNilStrings(l.Alternatives),
	}
}

// grammarExplanationToWire converts the structured grammar rule to the wire type.
func grammarExplanationToWire(g analysis.GrammarExplanation) GrammarExplanation {
	return GrammarExplanation{
		RuleName:       g.RuleName,
		Explanation:    g.Explanation,
		Construction:   g.Construction,
		Counterexample: g.Counterexample,
		Exception:      g.Exception,
		EsContrast:     g.ESContrast,
	}
}

// nonNilStrings guarantees a JSON array instead of null for a required list.
func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// errorPatternToWire converts a domain error pattern to its wire representation.
func errorPatternToWire(p domain.ErrorPattern) ErrorPattern {
	wire := ErrorPattern{
		Code:     ErrorPatternCode(p.Code),
		Severity: ErrorPatternSeverity(p.Severity),
	}
	if p.Note != "" {
		note := p.Note
		wire.Note = &note
	}
	return wire
}

// accessEventToWire converts a domain access event to its wire representation.
// Empty optional resource fields are omitted.
func accessEventToWire(event identity.AccessEvent) AccessLogEntry {
	entry := AccessLogEntry{
		Id:         openapi_types.UUID(event.ID),
		Action:     event.Action,
		OccurredAt: event.OccurredAt,
	}
	if event.ResourceType != "" {
		resourceType := event.ResourceType
		entry.ResourceType = &resourceType
	}
	if event.ResourceID != "" {
		resourceID := event.ResourceID
		entry.ResourceId = &resourceID
	}
	return entry
}
