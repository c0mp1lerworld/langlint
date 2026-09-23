// Command import is a one-off batch importer of manual practices (DEVLOG,
// checklist 09): it reads a JSON file of past practices, back-dates them to
// their real date and regenerates the AI analysis for each one through the
// existing LLMExtractor port, so the history "quedas al día" as if it had been
// produced by the application.
//
// It is a developer tool, not part of the HTTP API. Run it from apps/backend:
//
//	go run ./cmd/import -input ./import.json -reset
//	go run ./cmd/provisioner refresh-aggregates
//
// The importer only writes practices and analyses; analytics are rebuilt
// afterwards by refresh-aggregates (A4). The -reset flag truncates the
// practice/analytics data (not the audit tables) before importing.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/llm"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/adapters/postgres/repositories"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/ports"
	"github.com/c0mp1lerworld/langlint/backend/internal/api/services"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/analysis"
	"github.com/c0mp1lerworld/langlint/backend/internal/domain/practice"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/config"
	"github.com/c0mp1lerworld/langlint/backend/internal/shared/db"
	"github.com/c0mp1lerworld/langlint/backend/migrations"
)

// resetSQL truncates the practice and analytics data tables (not the audit
// tables access_events/deletion_requests). CASCADE handles the FK from analyses
// to practices.
const resetSQL = `TRUNCATE practices, analyses, error_metrics, quiz_attempts, study_sessions, outbox_events CASCADE`

// retryDelay is the pause between LLM attempts. gpt-4o-mini is mostly
// deterministic at temperature 0 but occasionally emits a fragment that fails
// the strict local validation, so a short backoff and a retry usually succeeds.
const retryDelay = 3 * time.Second

// maxImportRules mirrors practice.maxTargetRules (the domain limit of 9.5), so
// the import fails before touching the database or the LLM.
const maxImportRules = 5

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "import:", err)
		os.Exit(1)
	}
}

// importRule is the JSON shape of a target rule inside an import entry.
type importRule struct {
	Verb  string `json:"verb"`
	Tense string `json:"tense"`
	Note  string `json:"note"`
}

// importEntry is the JSON shape of one practice in the import file.
type importEntry struct {
	Date        string       `json:"date"`
	SourceText  string       `json:"source_text"`
	DraftText   string       `json:"draft_text"`
	TargetRules []importRule `json:"target_rules"`
}

// importPractice is the validated, domain-typed form of an import entry.
type importPractice struct {
	CreatedAt time.Time
	Source    practice.SourceText
	Draft     practice.DraftText
	Rules     []practice.TargetRule
}

func run(args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("import", flag.ContinueOnError)
	flags.SetOutput(stderr)
	envFile := flags.String("env", ".env", "path to the .env file with the server and OpenAI settings")
	inputFile := flags.String("input", "import.json", "path to the JSON file with the practices to import")
	reset := flags.Bool("reset", false, "truncate the practice/analytics tables before importing")
	timeout := flags.Duration("timeout", 15*time.Minute, "per-practice LLM call timeout")
	attempts := flags.Int("attempts", 3, "per-practice LLM attempts before marking it failed")
	validateOnly := flags.Bool("validate", false, "parse and validate the import file without importing (dry-run)")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if err := config.LoadDotEnv(*envFile); err != nil {
		return err
	}
	serverCfg, err := config.LoadServerConfig()
	if err != nil {
		return err
	}
	openaiCfg, err := config.LoadOpenAIConfig()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(*inputFile)
	if err != nil {
		return err
	}
	entries, err := parseEntries(data)
	if err != nil {
		return err
	}
	practices, err := buildPractices(entries)
	if err != nil {
		return err
	}

	if *validateOnly {
		dates := make([]string, 0, len(practices))
		for _, p := range practices {
			dates = append(dates, p.CreatedAt.Format("2006-01-02"))
		}
		fmt.Fprintf(stdout, "OK: %d practices valid (dates: %s)\n", len(practices), strings.Join(dates, ", "))
		return nil
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, serverCfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := migrations.Up(pool); err != nil {
		return err
	}

	if *reset {
		if _, err := pool.Exec(ctx, resetSQL); err != nil {
			return err
		}
		fmt.Fprintln(stderr, "reset: testing data truncated")
	}

	practiceRepo := repositories.NewPracticeRepository(pool)
	analysisRepo := repositories.NewAnalysisRepository(pool)

	options := []option.RequestOption{
		option.WithAPIKey(openaiCfg.APIKey),
		option.WithBaseURL(llm.BaseURLOrDefault(openaiCfg.BaseURL)),
	}
	extractor := llm.NewOpenAIExtractor(openai.NewClient(options...), openaiCfg.Model, openaiCfg.ModelVersion)

	imported, failed := 0, 0
	for i, p := range practices {
		label := fmt.Sprintf("[%d/%d] %s", i+1, len(practices), p.CreatedAt.Format("2006-01-02"))

		practiceAgg, err := practice.NewPractice(serverCfg.UserID, p.Source, p.Draft, p.Rules, p.CreatedAt)
		if err != nil {
			return fmt.Errorf("entry %s: %w", label, err)
		}
		if err := practiceRepo.Save(ctx, practiceAgg); err != nil {
			return err
		}

		fragments, extractErr := extractWithRetry(ctx, extractor, practiceAgg, *attempts, *timeout, label, stderr)
		if extractErr != nil {
			if markFailed(practiceAgg, p.CreatedAt) == nil {
				_ = practiceRepo.Save(ctx, practiceAgg)
			}
			failed++
			fmt.Fprintf(stderr, "%s FAILED: %v\n", label, extractErr)
			continue
		}

		analysisAgg, err := analysis.NewAnalysis(practiceAgg.ID, extractor.Model(), extractor.ModelVersion(), p.CreatedAt)
		if err != nil {
			return err
		}
		if err := analysisAgg.Complete(fragments); err != nil {
			if markFailed(practiceAgg, p.CreatedAt) == nil {
				_ = practiceRepo.Save(ctx, practiceAgg)
			}
			failed++
			fmt.Fprintf(stderr, "%s FAILED: %v\n", label, err)
			continue
		}
		if err := analysisRepo.Save(ctx, analysisAgg); err != nil {
			return err
		}

		if err := markCompleted(practiceAgg, p.CreatedAt); err != nil {
			return err
		}
		if err := practiceRepo.Save(ctx, practiceAgg); err != nil {
			return err
		}

		imported++
		fmt.Fprintf(stdout, "%s imported (%d fragments)\n", label, len(fragments))
	}

	fmt.Fprintf(stdout, "done: imported %d, failed %d\n", imported, failed)
	return nil
}

// markCompleted transitions the practice draft -> analyzing -> completed at the
// historical timestamp, mirroring the application lifecycle.
func markCompleted(p *practice.Practice, at time.Time) error {
	if err := p.StartAnalysis(at); err != nil {
		return err
	}
	return p.MarkCompleted(at)
}

// markFailed transitions the practice draft -> analyzing -> failed at the
// historical timestamp.
func markFailed(p *practice.Practice, at time.Time) error {
	if err := p.StartAnalysis(at); err != nil {
		return err
	}
	return p.MarkFailed(at)
}

// extractWithRetry runs the anonymized extraction up to attempts times, pausing
// retryDelay between attempts. gpt-4o-mini occasionally emits a fragment that
// fails the strict local validation even at temperature 0, so a retry usually
// succeeds. It returns the fragments or the last error.
func extractWithRetry(ctx context.Context, extractor ports.LLMExtractor, p *practice.Practice, attempts int, timeout time.Duration, label string, stderr io.Writer) ([]analysis.Fragment, error) {
	var fragments []analysis.Fragment
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		llmCtx, cancel := context.WithTimeout(ctx, timeout)
		fragments, err = extractor.Extract(llmCtx, ports.ExtractRequest{
			PracticeID:  p.ID,
			SourceText:  services.AnonymizeSpanish(p.SourceText.String()),
			DraftText:   services.Anonymize(p.DraftText.String()),
			TargetRules: p.TargetRules,
		})
		cancel()

		if err == nil {
			return fragments, nil
		}
		if attempt < attempts {
			fmt.Fprintf(stderr, "%s attempt %d/%d failed (%v), retrying…\n", label, attempt, attempts, err)
			time.Sleep(retryDelay)
		}
	}
	return nil, err
}

// parseEntries unmarshals the import file into entries. The file must contain at
// least one practice.
func parseEntries(data []byte) ([]importEntry, error) {
	var entries []importEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse import file: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("import file must contain at least one practice")
	}
	return entries, nil
}

// buildPractices validates every entry against the domain value objects and
// returns them sorted by date (stable, preserving input order for equal dates).
func buildPractices(entries []importEntry) ([]importPractice, error) {
	out := make([]importPractice, 0, len(entries))
	for i, e := range entries {
		createdAt, err := parseDate(e.Date)
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", i+1, err)
		}
		source, err := practice.NewSourceText(e.SourceText)
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", i+1, err)
		}
		draft, err := practice.NewDraftText(e.DraftText)
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", i+1, err)
		}
		rules := make([]practice.TargetRule, 0, len(e.TargetRules))
		for _, r := range e.TargetRules {
			rule, err := practice.NewTargetRule(r.Verb, r.Tense, r.Note)
			if err != nil {
				return nil, fmt.Errorf("entry %d: %w", i+1, err)
			}
			rules = append(rules, rule)
		}
		if len(rules) == 0 {
			return nil, fmt.Errorf("entry %d: target_rules must contain at least one rule", i+1)
		}
		if len(rules) > maxImportRules {
			return nil, fmt.Errorf("entry %d: target_rules must contain at most 5 rules", i+1)
		}
		out = append(out, importPractice{CreatedAt: createdAt, Source: source, Draft: draft, Rules: rules})
	}

	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

// parseDate maps a YYYY-MM-DD day to noon UTC (the import has day-only
// precision).
func parseDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q (want YYYY-MM-DD)", raw)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, time.UTC), nil
}
