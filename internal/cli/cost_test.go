package cli

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixtureSessionA is a small Pi session JSONL with two cost-carrying deepseek
// messages, one openai message, one message with usage but no cost (skipped),
// and a non-message line (skipped).
const fixtureSessionA = `{"type":"session","version":3,"id":"sess-a","timestamp":"2026-08-01T10:00:00Z"}
{"type":"message","id":"m1","parentId":null,"timestamp":"2026-08-01T10:00:00Z","message":{"role":"assistant","provider":"deepseek","model":"deepseek-v4-flash","usage":{"input":500,"output":100,"totalTokens":600,"cost":{"input":0.0001,"output":0.00002,"total":0.00012}}}}
{"type":"message","id":"m2","parentId":null,"timestamp":"2026-08-01T10:01:00Z","message":{"role":"assistant","provider":"deepseek","model":"deepseek-v4-flash","usage":{"input":300,"output":50,"totalTokens":350,"cost":{"input":0.00006,"output":0.00001,"total":0.00007}}}}
{"type":"message","id":"m3","parentId":null,"timestamp":"2026-08-01T10:02:00Z","message":{"role":"assistant","provider":"deepseek","model":"deepseek-v4-flash","usage":{"input":10,"output":10,"totalTokens":20}}}
{"type":"message","id":"m4","parentId":null,"timestamp":"2026-08-01T10:03:00Z","message":{"role":"assistant","provider":"openai","model":"openai/gpt-5.6-terra","usage":{"input":1000,"output":200,"totalTokens":1200,"cost":{"total":0.002}}}}
{"type":"model_change","id":"mc1","parentId":null,"timestamp":"2026-08-01T10:04:00Z","provider":"deepseek","modelId":"deepseek-v4-flash"}
`

// fixtureSessionB adds a second deepseek session file.
const fixtureSessionB = `{"type":"message","id":"m5","parentId":null,"timestamp":"2026-08-02T10:00:00Z","message":{"role":"assistant","provider":"deepseek","model":"deepseek-v4-flash","usage":{"input":100,"output":20,"totalTokens":120,"cost":{"total":0.0005}}}}
`

// costFixtureRoot creates a temp HARNESS_ROOT with the given session files and
// clears ambient budget/provider/key env so tests are hermetic.
func costFixtureRoot(t *testing.T, sessions map[string]string) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	t.Setenv("PI_PROVIDER", "")
	t.Setenv("PI_MAX_BUDGET_USD", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("BW_GET", filepath.Join(t.TempDir(), "nonexistent-bw-get"))
	for name, content := range sessions {
		dir := filepath.Join(root, ".pi", "sessions")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func nearlyEqual(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}

func TestScanSessionFile(t *testing.T) {
	root := costFixtureRoot(t, map[string]string{"a.jsonl": fixtureSessionA})
	path := filepath.Join(root, ".pi", "sessions", "a.jsonl")
	samples, err := scanSessionFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 3 {
		t.Fatalf("got %d samples, want 3 (message without usage.cost and non-message lines skipped)", len(samples))
	}
	first := samples[0]
	if first.Provider != "deepseek" || first.Model != "deepseek-v4-flash" {
		t.Fatalf("got provider=%q model=%q", first.Provider, first.Model)
	}
	if first.InputTokens != 500 || first.OutputTokens != 100 || first.TotalTokens != 600 {
		t.Fatalf("got tokens in=%d out=%d total=%d", first.InputTokens, first.OutputTokens, first.TotalTokens)
	}
	if !nearlyEqual(first.CostUSD, 0.00012, 1e-12) {
		t.Fatalf("got cost %v, want 0.00012", first.CostUSD)
	}
	last := samples[2]
	if last.Provider != "openai" || !nearlyEqual(last.CostUSD, 0.002, 1e-12) {
		t.Fatalf("got provider=%q cost=%v", last.Provider, last.CostUSD)
	}
}

func TestScanSessionFileMissingFile(t *testing.T) {
	if _, err := scanSessionFile(filepath.Join(t.TempDir(), "nope.jsonl")); err == nil {
		t.Fatal("expected error for missing session file")
	}
}

func TestCollectCostReport(t *testing.T) {
	root := costFixtureRoot(t, map[string]string{
		"a.jsonl": fixtureSessionA,
		"b.jsonl": fixtureSessionB,
	})
	report, err := collectCostReport(root, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(report.Rows))
	}
	// Sorted by cost desc: openai (0.002) before deepseek (0.00069).
	first, second := report.Rows[0], report.Rows[1]
	if first.Provider != "openai" || !nearlyEqual(first.CostUSD, 0.002, 1e-12) {
		t.Fatalf("first row = %+v, want openai 0.002", first)
	}
	if first.TotalTokens != 1200 || first.Sessions != 1 {
		t.Fatalf("openai row tokens=%d sessions=%d", first.TotalTokens, first.Sessions)
	}
	if second.Provider != "deepseek" || !nearlyEqual(second.CostUSD, 0.00069, 1e-12) {
		t.Fatalf("second row = %+v, want deepseek 0.00069", second)
	}
	if second.InputTokens != 900 || second.OutputTokens != 170 || second.TotalTokens != 1070 || second.Sessions != 2 {
		t.Fatalf("deepseek row in=%d out=%d tokens=%d sessions=%d", second.InputTokens, second.OutputTokens, second.TotalTokens, second.Sessions)
	}
	if !nearlyEqual(report.TotalCostUSD, 0.00269, 1e-12) {
		t.Fatalf("total cost %v, want 0.00269", report.TotalCostUSD)
	}
	if report.TotalTokens != 2270 || report.Sessions != 2 {
		t.Fatalf("totals tokens=%d sessions=%d", report.TotalTokens, report.Sessions)
	}
}

func TestCollectCostReportNoSessionsDir(t *testing.T) {
	root := costFixtureRoot(t, nil)
	report, err := collectCostReport(root, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Rows) != 0 || report.TotalCostUSD != 0 || report.Sessions != 0 {
		t.Fatalf("expected empty report, got %+v", report)
	}
}

func TestRunCostJSON(t *testing.T) {
	costFixtureRoot(t, map[string]string{
		"a.jsonl": fixtureSessionA,
		"b.jsonl": fixtureSessionB,
	})
	code, out := captureRunStdout(t, []string{"cost", "--json"})
	if code != 0 {
		t.Fatalf("cost --json exit = %d, want 0; stderr: %s", code, out)
	}
	var report costReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if len(report.Rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(report.Rows))
	}
	if !nearlyEqual(report.TotalCostUSD, 0.00269, 1e-12) {
		t.Fatalf("total cost %v, want 0.00269", report.TotalCostUSD)
	}
	if report.TotalTokens != 2270 || report.Sessions != 2 {
		t.Fatalf("totals tokens=%d sessions=%d", report.TotalTokens, report.Sessions)
	}
	for _, row := range report.Rows {
		if row.Sessions < 1 || row.TotalTokens <= 0 {
			t.Fatalf("row %+v has empty aggregation fields", row)
		}
	}
}

func TestRunCostHumanTable(t *testing.T) {
	costFixtureRoot(t, map[string]string{"a.jsonl": fixtureSessionA})
	code, out := captureRunStdout(t, []string{"cost"})
	if code != 0 {
		t.Fatalf("cost exit = %d, want 0", code)
	}
	for _, want := range []string{"PROVIDER", "MODEL", "TOKENS", "COST (USD)", "SESSIONS", "deepseek", "$0.000190", "TOTAL"} {
		if !strings.Contains(out, want) {
			t.Fatalf("cost table missing %q:\n%s", want, out)
		}
	}
}

func TestRunCostSince(t *testing.T) {
	root := costFixtureRoot(t, map[string]string{
		"a.jsonl": fixtureSessionA,
		"b.jsonl": fixtureSessionB,
	})
	aPath := filepath.Join(root, ".pi", "sessions", "a.jsonl")
	bPath := filepath.Join(root, ".pi", "sessions", "b.jsonl")
	old := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(aPath, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(bPath, recent, recent); err != nil {
		t.Fatal(err)
	}

	// --since excludes a.jsonl (mtime 2026-07-01).
	code, out := captureRunStdout(t, []string{"cost", "--since", "2026-07-15", "--json"})
	if code != 0 {
		t.Fatalf("cost --since exit = %d, want 0", code)
	}
	var report costReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Rows) != 1 || report.Rows[0].Model != "deepseek-v4-flash" {
		t.Fatalf("expected only deepseek rows, got %+v", report.Rows)
	}
	if !nearlyEqual(report.TotalCostUSD, 0.0005, 1e-12) {
		t.Fatalf("total cost %v, want 0.0005 (b.jsonl only)", report.TotalCostUSD)
	}

	// --since after both files: empty report.
	code, out = captureRunStdout(t, []string{"cost", "--since", "2026-08-02", "--json"})
	if code != 0 {
		t.Fatalf("cost --since exit = %d, want 0", code)
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Rows) != 0 || report.Sessions != 0 {
		t.Fatalf("expected empty report, got %+v", report)
	}
}

func TestRunCostSinceRFC3339(t *testing.T) {
	costFixtureRoot(t, map[string]string{"a.jsonl": fixtureSessionA})
	code, out := captureRunStdout(t, []string{"cost", "--since=2026-08-01T10:00:00Z", "--json"})
	if code != 0 {
		t.Fatalf("cost --since=rfc3339 exit = %d, want 0; stderr: %s", code, out)
	}
	if !strings.Contains(out, "deepseek") {
		t.Fatalf("RFC3339 --since should include session a: %s", out)
	}
}

func TestRunCostFlagErrors(t *testing.T) {
	costFixtureRoot(t, map[string]string{"a.jsonl": fixtureSessionA})

	code, out := captureRunStderr(t, []string{"cost", "--since", "notadate"})
	if code != 2 || !strings.Contains(out, "invalid --since") {
		t.Fatalf("cost --since bad date exit=%d stderr=%q, want 2 with invalid --since", code, out)
	}
	code, out = captureRunStderr(t, []string{"cost", "--since"})
	if code != 2 || !strings.Contains(out, "--since requires a date") {
		t.Fatalf("cost --since w/o value exit=%d stderr=%q, want 2", code, out)
	}
	code, out = captureRunStderr(t, []string{"cost", "--bogus"})
	if code != 2 || !strings.Contains(out, "unknown flag") {
		t.Fatalf("cost --bogus exit=%d stderr=%q, want 2 with unknown flag", code, out)
	}
	code, out = captureRunStdout(t, []string{"cost", "--help"})
	if code != 0 || !strings.Contains(out, "Usage: pi-run cost") {
		t.Fatalf("cost --help exit=%d out=%q, want 0 with usage", code, out)
	}
}

func TestRunCostReset(t *testing.T) {
	root := costFixtureRoot(t, nil)
	ledger := filepath.Join(root, ".pi", "cost-ledger.jsonl")
	content := `{"ts":"2026-08-01T10:00:00Z","provider":"deepseek","model":"deepseek-v4-flash","inputTokens":800,"outputTokens":150,"costUsd":0.00019,"mode":"chat"}` + "\n"
	if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledger, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := captureRunStdout(t, []string{"cost", "--reset"})
	if code != 0 {
		t.Fatalf("cost --reset exit = %d, want 0; stderr: %s", code, out)
	}
	if !strings.Contains(out, "archived spend ledger") {
		t.Fatalf("cost --reset output missing archive notice: %q", out)
	}
	if _, err := os.Stat(ledger); !os.IsNotExist(err) {
		t.Fatalf("ledger must be archived (moved), stat err = %v", err)
	}
	archives, err := filepath.Glob(filepath.Join(root, ".pi", "cost-ledger-*.archive.jsonl"))
	if err != nil || len(archives) != 1 {
		t.Fatalf("expected one archive file, got %v (err %v)", archives, err)
	}
	marker, err := os.ReadFile(filepath.Join(root, ".pi", "cost-ledger.reset"))
	if err != nil {
		t.Fatalf("reset marker missing: %v", err)
	}
	if _, err := time.Parse(time.RFC3339, strings.TrimSpace(string(marker))); err != nil {
		t.Fatalf("reset marker %q is not RFC3339: %v", marker, err)
	}
	if len(archives) > 0 {
		if got, err := os.ReadFile(archives[0]); err != nil || !strings.Contains(string(got), "0.00019") {
			t.Fatalf("archive must preserve the old ledger contents, got %q (err %v)", got, err)
		}
	}

	// A second --reset with no ledger must still write the marker and not fail.
	code, out = captureRunStdout(t, []string{"cost", "--reset"})
	if code != 0 || strings.Contains(out, "archived") {
		t.Fatalf("second reset exit=%d out=%q, want 0 and no archive notice", code, out)
	}
}

func TestBudgetPreflightFlag(t *testing.T) {
	costFixtureRoot(t, map[string]string{
		"a.jsonl": fixtureSessionA,
		"b.jsonl": fixtureSessionB,
	})
	// Cumulative session spend is 0.00269; cap 0.001 → refuse with exit 6
	// BEFORE any key resolution.
	code, out := captureRunStderr(t, []string{"print", "--max-budget-usd", "0.001", "hello"})
	if code != exitBudgetExceeded {
		t.Fatalf("print over budget exit = %d, want %d; stderr: %s", code, exitBudgetExceeded, out)
	}
	if !strings.Contains(out, "budget exceeded") || !strings.Contains(out, "--max-budget-usd") {
		t.Fatalf("budget refusal message unclear: %q", out)
	}

	// Cap above spend: proceed to key resolution (missing key → exit 3).
	code, _ = captureRunStderr(t, []string{"print", "--max-budget-usd", "10", "hello"})
	if code != 3 {
		t.Fatalf("print under budget exit = %d, want 3 (missing key)", code)
	}
}

func TestBudgetPreflightEnv(t *testing.T) {
	costFixtureRoot(t, map[string]string{"a.jsonl": fixtureSessionA})
	t.Setenv("PI_MAX_BUDGET_USD", "0.001")
	code, out := captureRunStderr(t, []string{"print", "hello"})
	if code != exitBudgetExceeded {
		t.Fatalf("env budget exit = %d, want %d; stderr: %s", code, exitBudgetExceeded, out)
	}
	if !strings.Contains(out, "budget exceeded") {
		t.Fatalf("env budget refusal message unclear: %q", out)
	}
}

func TestBudgetFlagOverridesEnv(t *testing.T) {
	costFixtureRoot(t, map[string]string{"a.jsonl": fixtureSessionA})
	t.Setenv("PI_MAX_BUDGET_USD", "0.001") // would refuse if used
	code, _ := captureRunStderr(t, []string{"print", "--max-budget-usd", "10", "hello"})
	if code != 3 {
		t.Fatalf("flag must override env: exit = %d, want 3 (missing key)", code)
	}
}

func TestBudgetInvalidFlag(t *testing.T) {
	costFixtureRoot(t, nil)
	code, out := captureRunStderr(t, []string{"print", "--max-budget-usd", "nope", "hello"})
	if code != 2 || !strings.Contains(out, "invalid budget cap") {
		t.Fatalf("invalid budget exit=%d stderr=%q, want 2", code, out)
	}
	code, out = captureRunStderr(t, []string{"print", "--max-budget-usd=-1", "hello"})
	if code != 2 || !strings.Contains(out, "invalid budget cap") {
		t.Fatalf("negative budget exit=%d stderr=%q, want 2", code, out)
	}
	t.Setenv("PI_MAX_BUDGET_USD", "oops")
	code, out = captureRunStderr(t, []string{"print", "hello"})
	if code != 2 || !strings.Contains(out, "invalid budget cap") {
		t.Fatalf("invalid env budget exit=%d stderr=%q, want 2", code, out)
	}
}

func TestSplitLaunchArgsBudget(t *testing.T) {
	p, m, b, _, _, rest := splitLaunchArgs([]string{"--max-budget-usd", "5.5", "hello"})
	if b != "5.5" || len(rest) != 1 || rest[0] != "hello" || p != "" || m != "" {
		t.Fatalf("got provider=%q model=%q budget=%q rest=%v", p, m, b, rest)
	}
	_, _, b, _, _, _ = splitLaunchArgs([]string{"--max-budget-usd=2", "hi"})
	if b != "2" {
		t.Fatalf("got budget=%q, want 2", b)
	}
	_, _, b, _, _, _ = splitLaunchArgs([]string{"hello"})
	if b != "" {
		t.Fatalf("no budget flag must leave budget empty, got %q", b)
	}
	// Budget flag must not leak into pass-through args.
	_, _, _, _, _, rest = splitLaunchArgs([]string{"--max-budget-usd", "1", "--tools", "read", "x"})
	if len(rest) != 3 || rest[0] != "--tools" || rest[1] != "read" {
		t.Fatalf("rest = %v, want pass-through preserved", rest)
	}
}

func TestSplitCostModeFlag(t *testing.T) {
	mode, rest := splitCostModeFlag([]string{"--cost-mode", "live-eval", "hello"})
	if mode != "live-eval" || len(rest) != 1 || rest[0] != "hello" {
		t.Fatalf("got mode=%q rest=%v, want live-eval + [hello]", mode, rest)
	}
	mode, rest = splitCostModeFlag([]string{"--cost-mode=benchmark", "hi"})
	if mode != "benchmark" || len(rest) != 1 || rest[0] != "hi" {
		t.Fatalf("got mode=%q rest=%v, want benchmark + [hi]", mode, rest)
	}
	// Absent flag → empty mode, args untouched.
	mode, rest = splitCostModeFlag([]string{"--tools", "read", "x"})
	if mode != "" || len(rest) != 3 || rest[0] != "--tools" {
		t.Fatalf("absent flag: got mode=%q rest=%v, want empty + pass-through preserved", mode, rest)
	}
	// --cost-mode must not leak into pass-through args.
	mode, rest = splitCostModeFlag([]string{"--cost-mode", "live-eval", "--tools", "read", "x"})
	if mode != "live-eval" || len(rest) != 3 || rest[0] != "--tools" || rest[1] != "read" || rest[2] != "x" {
		t.Fatalf("got mode=%q rest=%v, want live-eval + [--tools read x]", mode, rest)
	}
	// "--" ends flag parsing: the tail is preserved verbatim.
	mode, rest = splitCostModeFlag([]string{"--", "--cost-mode", "x"})
	if mode != "" || len(rest) != 3 || rest[1] != "--cost-mode" {
		t.Fatalf("-- must end flag parsing: got mode=%q rest=%v", mode, rest)
	}
}

func TestResolveCostMode(t *testing.T) {
	// Default when the flag is absent = the command name (today's behavior).
	for cmd, want := range map[string]string{"chat": "chat", "print": "print", "resume": "resume"} {
		got, err := resolveCostMode(cmd, "")
		if err != nil || got != want {
			t.Fatalf("resolveCostMode(%q, \"\") = %q, %v; want %q", cmd, got, err, want)
		}
	}
	// Every documented mode is accepted.
	for _, m := range []string{"chat", "print", "resume", "backfill", "benchmark", "live-eval"} {
		got, err := resolveCostMode("print", m)
		if err != nil || got != m {
			t.Fatalf("resolveCostMode(print, %q) = %q, %v; want %q", m, got, err, m)
		}
	}
	// Unknown mode is a usage error naming the valid set.
	if _, err := resolveCostMode("print", "bogus"); err == nil || !strings.Contains(err.Error(), "unknown cost mode") {
		t.Fatalf("unknown cost mode error = %v, want 'unknown cost mode'", err)
	}
}

func TestPiArgsPrintPersistSessionWhenBudgeted(t *testing.T) {
	p, _ := LookupProvider("deepseek")
	got := piArgs(p, p.DefaultModel, "print", []string{"hello"}, true, "")
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "-p") {
		t.Fatalf("print args missing -p: %v", got)
	}
	if strings.Contains(joined, "--no-session") {
		t.Fatalf("budgeted print must keep a session file (no --no-session): %v", got)
	}
}

func TestLedgerAppendAndSum(t *testing.T) {
	root := costFixtureRoot(t, nil)
	if empty, err := ledgerEmpty(root); err != nil || !empty {
		t.Fatalf("fresh root must have an empty ledger (empty=%v err=%v)", empty, err)
	}
	entries := []ledgerEntry{
		{Provider: "deepseek", Model: "deepseek-v4-flash", InputTokens: 800, OutputTokens: 150, CostUSD: 0.00019, Mode: "chat"},
		{Provider: "openai", Model: "openai/gpt-5.6-terra", InputTokens: 1000, OutputTokens: 200, CostUSD: 0.002, Mode: "print"},
	}
	if err := ledgerAppend(root, entries); err != nil {
		t.Fatal(err)
	}
	if empty, err := ledgerEmpty(root); err != nil || empty {
		t.Fatalf("ledger must be non-empty after append (empty=%v err=%v)", empty, err)
	}
	total, err := ledgerSum(root)
	if err != nil {
		t.Fatal(err)
	}
	if !nearlyEqual(total, 0.00219, 1e-12) {
		t.Fatalf("ledger sum %v, want 0.00219", total)
	}
	all, err := ledgerEntries(root)
	if err != nil || len(all) != 2 {
		t.Fatalf("got %d entries (err %v), want 2", len(all), err)
	}
	for _, e := range all {
		if e.TS == "" {
			t.Fatalf("ledger entry missing ts: %+v", e)
		}
		if e.Mode != "chat" && e.Mode != "print" {
			t.Fatalf("unexpected mode %q", e.Mode)
		}
	}

	// Owner-only permissions: ledger file 0600, .pi dir 0700 (spend ledger is
	// pi-run-owned and attributed to sessions).
	lf, err := os.Stat(ledgerPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if got := lf.Mode().Perm(); got != 0o600 {
		t.Fatalf("ledger perms %o, want 0600", got)
	}
	df, err := os.Stat(filepath.Dir(ledgerPath(root)))
	if err != nil {
		t.Fatal(err)
	}
	if got := df.Mode().Perm(); got != 0o700 {
		t.Fatalf("ledger dir perms %o, want 0700", got)
	}
}

func TestCurrentSpendMaxSemantics(t *testing.T) {
	root := costFixtureRoot(t, map[string]string{"a.jsonl": fixtureSessionA})
	// Session spend = 0.00219. Ledger larger → ledger wins.
	ledger := filepath.Join(root, ".pi", "cost-ledger.jsonl")
	ledgerContent := `{"ts":"2026-08-01T10:00:00Z","provider":"deepseek","model":"deepseek-v4-flash","inputTokens":800,"outputTokens":150,"costUsd":0.005,"mode":"chat"}` + "\n"
	if err := os.MkdirAll(filepath.Dir(ledger), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ledger, []byte(ledgerContent), 0o644); err != nil {
		t.Fatal(err)
	}
	spend, err := currentSpend(root)
	if err != nil {
		t.Fatal(err)
	}
	if !nearlyEqual(spend, 0.005, 1e-12) {
		t.Fatalf("currentSpend %v, want 0.005 (ledger wins, no double count)", spend)
	}

	// Remove the ledger: sessions win.
	if err := os.Remove(ledger); err != nil {
		t.Fatal(err)
	}
	spend, err = currentSpend(root)
	if err != nil {
		t.Fatal(err)
	}
	if !nearlyEqual(spend, 0.00219, 1e-12) {
		t.Fatalf("currentSpend %v, want 0.00219 (sessions win)", spend)
	}
}

func TestRecordRunSpendBackfillAndAttribution(t *testing.T) {
	root := costFixtureRoot(t, map[string]string{
		"old.jsonl": fixtureSessionA, // pre-run session, cost 0.00219
		"new.jsonl": fixtureSessionB, // run session, cost 0.0005
	})
	old := filepath.Join(root, ".pi", "sessions", "old.jsonl")
	past := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}

	start := time.Now().Add(-time.Minute)
	preSpend := 0.00219 // pre-flight counted only the old session
	post, err := recordRunSpend(root, start, "print", "openai", "openai/gpt-5.6-terra", preSpend)
	if err != nil {
		t.Fatal(err)
	}
	if !nearlyEqual(post, 0.00269, 1e-12) {
		t.Fatalf("post spend %v, want 0.00269", post)
	}

	entries, err := ledgerEntries(root)
	if err != nil || len(entries) != 3 {
		t.Fatalf("got %d ledger entries (err %v), want 3 (2 backfill + 1 run)", len(entries), err)
	}
	byKey := map[string]ledgerEntry{}
	for _, e := range entries {
		byKey[e.Mode+"|"+e.Provider+"|"+e.Model] = e
	}
	backfillDS := byKey["backfill|deepseek|deepseek-v4-flash"]
	backfillOA := byKey["backfill|openai|openai/gpt-5.6-terra"]
	run := byKey["print|deepseek|deepseek-v4-flash"]
	if backfillDS.Mode == "" || backfillOA.Mode == "" || run.Mode == "" {
		t.Fatalf("expected backfill deepseek/openai + print entries, got %+v", entries)
	}
	// Old session (fixture A): deepseek 0.00019/950 tokens + openai 0.002/1200.
	if !nearlyEqual(backfillDS.CostUSD, 0.00019, 1e-12) || backfillDS.InputTokens+backfillDS.OutputTokens != 950 {
		t.Fatalf("backfill deepseek %+v, want cost 0.00019 tokens 950", backfillDS)
	}
	if !nearlyEqual(backfillOA.CostUSD, 0.002, 1e-12) || backfillOA.InputTokens+backfillOA.OutputTokens != 1200 {
		t.Fatalf("backfill openai %+v, want cost 0.002 tokens 1200", backfillOA)
	}
	// Run session (fixture B): deepseek 0.0005/120 tokens, mode print.
	if !nearlyEqual(run.CostUSD, 0.0005, 1e-12) || run.InputTokens != 100 || run.OutputTokens != 20 {
		t.Fatalf("run entry %+v, want cost 0.0005 in=100 out=20", run)
	}

	// The ledger now covers total spend: max() stays consistent.
	spend, err := currentSpend(root)
	if err != nil {
		t.Fatal(err)
	}
	if !nearlyEqual(spend, 0.00269, 1e-12) {
		t.Fatalf("currentSpend after recording %v, want 0.00269", spend)
	}
}

func TestRecordRunSpendNoDeltaWritesNothing(t *testing.T) {
	root := costFixtureRoot(t, map[string]string{"a.jsonl": fixtureSessionA})
	start := time.Now().Add(-time.Minute)
	// preSpend already includes the session → run added no new spend.
	post, err := recordRunSpend(root, start, "chat", "deepseek", "deepseek-v4-flash", 0.00219)
	if err != nil {
		t.Fatal(err)
	}
	if !nearlyEqual(post, 0.00219, 1e-12) {
		t.Fatalf("post %v, want 0.00219", post)
	}
	if _, err := os.Stat(filepath.Join(root, ".pi", "cost-ledger.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("no-delta run must not create a ledger, stat err = %v", err)
	}
}

func TestRecordRunSpendLiveEvalMode(t *testing.T) {
	root := costFixtureRoot(t, map[string]string{"run.jsonl": fixtureSessionB})
	start := time.Now().Add(-time.Minute)
	post, err := recordRunSpend(root, start, "live-eval", "deepseek", "deepseek-v4-flash", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !nearlyEqual(post, 0.0005, 1e-12) {
		t.Fatalf("post spend = %v, want 0.0005", post)
	}
	entries, err := ledgerEntries(root)
	if err != nil || len(entries) != 1 {
		t.Fatalf("got %d ledger entries (err %v), want 1", len(entries), err)
	}
	e := entries[0]
	if e.Mode != "live-eval" {
		t.Fatalf("ledger mode = %q, want live-eval; entry %+v", e.Mode, e)
	}
	if e.Provider != "deepseek" || e.Model != "deepseek-v4-flash" || !nearlyEqual(e.CostUSD, 0.0005, 1e-12) {
		t.Fatalf("entry = %+v, want deepseek/deepseek-v4-flash cost 0.0005", e)
	}
}

func TestCostResetStartsFreshBudgetPeriod(t *testing.T) {
	root := costFixtureRoot(t, map[string]string{"old.jsonl": fixtureSessionA})
	old := filepath.Join(root, ".pi", "sessions", "old.jsonl")
	past := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	if code, _ := captureRunStdout(t, []string{"cost", "--reset"}); code != 0 {
		t.Fatalf("cost --reset exit = %d, want 0", code)
	}
	// Pre-reset sessions (mtime < marker) no longer count toward the budget.
	spend, err := currentSpend(root)
	if err != nil {
		t.Fatal(err)
	}
	if spend != 0 {
		t.Fatalf("currentSpend after reset %v, want 0 (pre-reset spend excluded)", spend)
	}
	// A fresh session after the reset counts again.
	if err := os.WriteFile(filepath.Join(root, ".pi", "sessions", "new.jsonl"), []byte(fixtureSessionB), 0o644); err != nil {
		t.Fatal(err)
	}
	spend, err = currentSpend(root)
	if err != nil {
		t.Fatal(err)
	}
	if !nearlyEqual(spend, 0.0005, 1e-12) {
		t.Fatalf("currentSpend after new session %v, want 0.0005", spend)
	}
}

func TestBudgetAfterResetAllowsRun(t *testing.T) {
	root := costFixtureRoot(t, map[string]string{"old.jsonl": fixtureSessionA})
	past := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	if err := os.Chtimes(filepath.Join(root, ".pi", "sessions", "old.jsonl"), past, past); err != nil {
		t.Fatal(err)
	}
	if code, _ := captureRunStdout(t, []string{"cost", "--reset"}); code != 0 {
		t.Fatalf("cost --reset exit = %d, want 0", code)
	}
	// After reset, spend is 0 → a tiny cap passes and the run proceeds to key
	// resolution (missing key → exit 3), proving pre-flight did not refuse.
	code, _ := captureRunStderr(t, []string{"print", "--max-budget-usd", "0.0001", "hello"})
	if code != 3 {
		t.Fatalf("post-reset print exit = %d, want 3 (proceeded past budget)", code)
	}
}

func TestExitCodesTableDocumentsBudget(t *testing.T) {
	if !strings.Contains(exitCodesText, "6  budget exceeded") {
		t.Fatalf("exit-codes table must document 6 = budget exceeded:\n%s", exitCodesText)
	}
	if !strings.Contains(usage, "6 budget exceeded") {
		t.Fatalf("usage exit-code line must mention 6 = budget exceeded")
	}
}

func TestUsageMentionsCostAndBudget(t *testing.T) {
	for _, want := range []string{"cost", "--max-budget-usd", "--json", "--since", "--reset"} {
		if !strings.Contains(usage, want) {
			t.Fatalf("usage missing %q", want)
		}
	}
}
