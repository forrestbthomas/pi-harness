package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// scorecardFixtureRun builds a graded run result with the given task statuses
// and durations, aggregating the summary exactly like the live runner.
func scorecardFixtureRun(provider, model string, tasks []benchmarkTaskResult) benchmarkRunResult {
	return benchmarkRunResult{
		RunID:     "fixture-run",
		Provider:  provider,
		Model:     model,
		Timestamp: "2026-08-11T15:04:05Z",
		Tasks:     tasks,
		Summary:   aggregateBenchmarkResults(tasks),
	}
}

func TestScorecardProviderFromRunAggregation(t *testing.T) {
	results := []benchmarkTaskResult{
		{ID: "a", Status: "pass", Passed: true, Duration: 10},
		{ID: "b", Status: "fail", Duration: 5},
		{ID: "c", Status: "error", Duration: 20},
	}
	res := scorecardFixtureRun("openai", "openai/gpt-5.6-terra", results)
	row := scorecardProviderFromRun(res, 0.5, 1000)

	if row.Provider != "openai" || row.Model != "openai/gpt-5.6-terra" {
		t.Fatalf("row identity = %+v", row)
	}
	if row.Passed != 1 || row.Total != 3 || row.Errors != 1 {
		t.Fatalf("row counts = %+v, want passed=1 total=3 errors=1", row)
	}
	if !nearlyEqual(row.PassRate, 1.0/3.0, 1e-9) {
		t.Fatalf("passRate = %v, want 1/3", row.PassRate)
	}
	// avgLatencyMs = mean per-task durationSecs * 1000.
	if !nearlyEqual(row.AvgLatencyMs, (10+5+20)/3.0*1000, 1e-9) {
		t.Fatalf("avgLatencyMs = %v, want %v", row.AvgLatencyMs, (10+5+20)/3.0*1000)
	}
	if row.CostUSD != 0.5 || row.Tokens != 1000 {
		t.Fatalf("cost/tokens = %v/%d, want 0.5/1000", row.CostUSD, row.Tokens)
	}
}

func TestScorecardProviderFromRunEmptyTasks(t *testing.T) {
	res := scorecardFixtureRun("openai", "openai/gpt-5.6-terra", nil)
	row := scorecardProviderFromRun(res, 0, 0)
	if row.Total != 0 || row.PassRate != 0 || row.AvgLatencyMs != 0 {
		t.Fatalf("empty run row = %+v, want all zeros", row)
	}
}

func TestMedianFloats(t *testing.T) {
	if got := medianFloats(nil); got != 0 {
		t.Fatalf("empty median = %v, want 0", got)
	}
	if got := medianFloats([]float64{3}); got != 3 {
		t.Fatalf("single median = %v, want 3", got)
	}
	if got := medianFloats([]float64{0.8, 0.6}); got != 0.7 {
		t.Fatalf("even median = %v, want 0.7", got)
	}
	if got := medianFloats([]float64{0.6, 1.0, 0.8}); got != 0.8 {
		t.Fatalf("odd median = %v, want 0.8", got)
	}
	if got := medianInts([]int{1000, 2000}); got != 1500 {
		t.Fatalf("even int median = %d, want 1500", got)
	}
}

func TestCollapseScorecardRuns(t *testing.T) {
	rows := []scorecardProvider{
		{Provider: "openai", Model: "m", Passed: 4, Total: 5, PassRate: 0.8, CostUSD: 0.10, AvgLatencyMs: 100, Tokens: 1000},
		{Provider: "openai", Model: "m", Passed: 5, Total: 5, PassRate: 1.0, CostUSD: 0.20, AvgLatencyMs: 200, Tokens: 2000},
	}
	out := collapseScorecardRuns(rows)
	if !nearlyEqual(out.PassRate, 0.9, 1e-9) {
		t.Fatalf("median passRate = %v, want 0.9", out.PassRate)
	}
	// Passed/total come from the repeat closest to the median pass rate (tie
	// → earliest repeat).
	if out.Passed != 4 || out.Total != 5 {
		t.Fatalf("passed/total = %d/%d, want 4/5", out.Passed, out.Total)
	}
	if !nearlyEqual(out.CostUSD, 0.15, 1e-9) || !nearlyEqual(out.AvgLatencyMs, 150, 1e-9) || out.Tokens != 1500 {
		t.Fatalf("medians = cost %v lat %v tokens %d, want 0.15/150/1500", out.CostUSD, out.AvgLatencyMs, out.Tokens)
	}
}

func TestCollapseScorecardRunsOddAndSingle(t *testing.T) {
	rows := []scorecardProvider{
		{Passed: 3, Total: 5, PassRate: 0.6, CostUSD: 1, AvgLatencyMs: 10, Tokens: 100},
		{Passed: 5, Total: 5, PassRate: 1.0, CostUSD: 3, AvgLatencyMs: 30, Tokens: 300},
		{Passed: 4, Total: 5, PassRate: 0.8, CostUSD: 2, AvgLatencyMs: 20, Tokens: 200},
	}
	out := collapseScorecardRuns(rows)
	if !nearlyEqual(out.PassRate, 0.8, 1e-9) || out.Passed != 4 || out.Total != 5 {
		t.Fatalf("odd collapse = %+v", out)
	}
	if !nearlyEqual(out.CostUSD, 2, 1e-9) || !nearlyEqual(out.AvgLatencyMs, 20, 1e-9) || out.Tokens != 200 {
		t.Fatalf("odd medians = %+v", out)
	}

	single := []scorecardProvider{{Provider: "openai", PassRate: 0.5, CostUSD: 1}}
	if got := collapseScorecardRuns(single); got.PassRate != 0.5 || got.CostUSD != 1 {
		t.Fatalf("single row collapse = %+v", got)
	}
	if got := collapseScorecardRuns(nil); got.Total != 0 {
		t.Fatalf("empty collapse = %+v", got)
	}
}

func TestEvaluateScorecardGates(t *testing.T) {
	passing := []scorecardProvider{
		{Provider: "openai", PassRate: 0.9, CostUSD: 1},
		{Provider: "deepseek", PassRate: 0.8, CostUSD: 2},
	}
	tests := []struct {
		name      string
		rows      []scorecardProvider
		repeats   []scorecardProvider
		failBelow float64
		baseline  map[string]float64
		tolerance float64
		budgetCap float64
		wantCode  int
		wantRegs  int
	}{
		{name: "all gates pass", rows: passing, failBelow: 0.7, tolerance: 0.05, budgetCap: 10, wantCode: 0},
		{name: "fail-below", rows: passing, failBelow: 0.85, tolerance: 0.05, wantCode: exitScorecardFailed},
		{name: "fail-below unset disabled", rows: passing, failBelow: -1, wantCode: 0},
		{name: "fail-below zero disabled", rows: passing, failBelow: 0, wantCode: 0},
		{name: "budget at cap", rows: passing, tolerance: 0.05, budgetCap: 3, wantCode: exitBudgetExceeded},
		{name: "budget boundary passes", rows: passing, tolerance: 0.05, budgetCap: 3.01, wantCode: 0},
		{name: "baseline regression", rows: passing, baseline: map[string]float64{"deepseek": 0.9}, tolerance: 0.05, wantCode: exitScorecardFailed, wantRegs: 1},
		{name: "baseline tolerance boundary passes", rows: passing, baseline: map[string]float64{"openai": 0.9, "deepseek": 0.85}, tolerance: 0.05, wantCode: 0},
		{name: "baseline provider only in current", rows: passing, baseline: map[string]float64{"other": 0.5}, tolerance: 0.05, wantCode: 0},
		{name: "baseline provider only in baseline", rows: passing, baseline: map[string]float64{"openai": 0.9, "gone": 1.0}, tolerance: 0.05, wantCode: 0},
		{name: "incomplete run", rows: passing, repeats: []scorecardProvider{{Provider: "openai", Errors: 1}}, failBelow: 0.7, tolerance: 0.05, wantCode: exitScorecardFailed},
		{name: "incomplete wins over budget", rows: passing, repeats: []scorecardProvider{{Errors: 1}}, tolerance: 0.05, budgetCap: 1, wantCode: exitScorecardFailed},
		{name: "regression wins over budget", rows: passing, baseline: map[string]float64{"deepseek": 0.9}, tolerance: 0.05, budgetCap: 1, wantCode: exitScorecardFailed, wantRegs: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repeats := tt.repeats
			if repeats == nil {
				repeats = tt.rows
			}
			st := evaluateScorecardGates(tt.rows, repeats, tt.failBelow, tt.baseline, tt.tolerance, tt.budgetCap)
			if got := scorecardExitCode(st); got != tt.wantCode {
				t.Fatalf("exit code = %d, want %d (status %+v)", got, tt.wantCode, st)
			}
			if len(st.Regressions) != tt.wantRegs {
				t.Fatalf("regressions = %d, want %d (status %+v)", len(st.Regressions), tt.wantRegs, st)
			}
		})
	}
}

func TestEvaluateScorecardGatesTotals(t *testing.T) {
	rows := []scorecardProvider{
		{Provider: "openai", PassRate: 0.8, CostUSD: 0.0412},
		{Provider: "deepseek", PassRate: 0.8, CostUSD: 0.0021},
	}
	st := evaluateScorecardGates(rows, rows, -1, nil, 0.05, 0)
	if !nearlyEqual(st.TotalCostUSD, 0.0433, 1e-12) {
		t.Fatalf("total cost = %v, want 0.0433", st.TotalCostUSD)
	}
}

func TestQuickProfileTasks(t *testing.T) {
	tasks := []benchmarkTask{
		{ID: "a", TimeoutSecs: 300},
		{ID: "b", TimeoutSecs: 30},
	}
	got := quickProfileTasks(tasks)
	if got[0].TimeoutSecs != quickProfileAgentTimeout || got[1].TimeoutSecs != 30 {
		t.Fatalf("quick profile timeouts = %d/%d", got[0].TimeoutSecs, got[1].TimeoutSecs)
	}
	if tasks[0].TimeoutSecs != 300 {
		t.Fatalf("quickProfileTasks mutated the input: %d", tasks[0].TimeoutSecs)
	}
}

func TestScorecardRunTokensFromLedgerAttribution(t *testing.T) {
	root := costFixtureRoot(t, map[string]string{"run.jsonl": fixtureSessionB})
	start := time.Now().Add(-time.Minute)
	before, err := ledgerEntries(root)
	if err != nil {
		t.Fatal(err)
	}
	// preSpend 0 mirrors a fresh budget period: the run's session spend is
	// attributed to this provider/model with ledger mode "benchmark".
	post, err := recordRunSpend(root, start, "benchmark", "deepseek", "deepseek-v4-flash", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !nearlyEqual(post, 0.0005, 1e-12) {
		t.Fatalf("post spend = %v, want 0.0005", post)
	}
	if got := scorecardRunTokens(root, before, "deepseek", "deepseek-v4-flash"); got != 120 {
		t.Fatalf("tokens = %d, want 120 (input 100 + output 20)", got)
	}
	// Other providers/models get no attribution.
	if got := scorecardRunTokens(root, before, "openai", "openai/gpt-5.6-terra"); got != 0 {
		t.Fatalf("unexpected openai tokens = %d, want 0", got)
	}
	// A second snapshot sees no new entries → no double counting.
	if got := scorecardRunTokens(root, before, "deepseek", "deepseek-v4-flash"); got != 120 {
		t.Fatalf("repeat tokens = %d, want still 120", got)
	}
}

func TestParseBaselineScorecardShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scorecard.json")
	content := `{
  "schemaVersion": 1,
  "runId": "20260811T150405-openai-deepseek",
  "providers": [
    {"provider": "openai", "model": "openai/gpt-5.6-terra", "passRate": 1.0},
    {"provider": "deepseek", "model": "deepseek/deepseek-v4-flash", "passRate": 0.8}
  ],
  "passed": false
}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := parseBaseline(path)
	if err != nil {
		t.Fatal(err)
	}
	if !nearlyEqual(got["openai"], 1.0, 1e-12) || !nearlyEqual(got["deepseek"], 0.8, 1e-12) {
		t.Fatalf("baseline = %v", got)
	}
	if len(got) != 2 {
		t.Fatalf("baseline has %d providers, want 2", len(got))
	}
}

func TestParseBaselineRunShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "run.json")
	content := `{"runId":"x","provider":"openai","model":"openai/gpt-5.6-terra","dryRun":false,"timestamp":"2026-08-11T10:00:00Z","tasks":[],"summary":{"total":5,"passed":4,"failed":1,"errors":0,"score":0.8}}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := parseBaseline(path)
	if err != nil {
		t.Fatal(err)
	}
	if !nearlyEqual(got["openai"], 0.8, 1e-12) {
		t.Fatalf("baseline = %v, want openai 0.8", got)
	}
}

func TestParseBaselineErrors(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.json")
	if _, err := parseBaseline(missing); err == nil {
		t.Fatal("missing baseline file must error")
	}

	garbage := filepath.Join(dir, "garbage.json")
	if err := os.WriteFile(garbage, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := parseBaseline(garbage); err == nil {
		t.Fatal("unparseable baseline must error")
	}

	emptyShape := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(emptyShape, []byte(`{"runId":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := parseBaseline(emptyShape); err == nil || !strings.Contains(err.Error(), "unrecognized shape") {
		t.Fatalf("empty-shape error = %v, want unrecognized shape", err)
	}

	badVersion := filepath.Join(dir, "badver.json")
	if err := os.WriteFile(badVersion, []byte(`{"schemaVersion": 2, "providers": [{"provider": "openai", "passRate": 1.0}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := parseBaseline(badVersion); err == nil || !strings.Contains(err.Error(), "schemaVersion") {
		t.Fatalf("bad-version error = %v, want schemaVersion error", err)
	}

	noProvider := filepath.Join(dir, "noprov.json")
	if err := os.WriteFile(noProvider, []byte(`{"summary": {"score": 0.5}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := parseBaseline(noProvider); err == nil || !strings.Contains(err.Error(), `"provider"`) {
		t.Fatalf("no-provider error = %v, want missing provider error", err)
	}
}

func TestParseScorecardArgsValid(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	opts, code := parseScorecardArgs([]string{
		"--providers", "openai, deepseek",
		"--models", "openai/gpt-5.6-terra,deepseek/deepseek-v4-flash",
		"--fail-below", "0.8",
		"--max-budget-usd", "5",
		"--baseline", "b.json",
		"--baseline-tolerance", "0.1",
		"--runs", "3",
		"--quick-profile",
	})
	if code != -1 {
		t.Fatalf("code = %d, want -1", code)
	}
	if len(opts.providers) != 2 || opts.providers[0] != "openai" || opts.providers[1] != "deepseek" {
		t.Fatalf("providers = %v", opts.providers)
	}
	if len(opts.models) != 2 || opts.models[1] != "deepseek/deepseek-v4-flash" {
		t.Fatalf("models = %v", opts.models)
	}
	if !nearlyEqual(opts.failBelow, 0.8, 1e-12) || opts.budgetFlag != "5" || opts.baselinePath != "b.json" {
		t.Fatalf("opts = %+v", opts)
	}
	if !nearlyEqual(opts.baselineTolerance, 0.1, 1e-12) || opts.runs != 3 || !opts.quickProfile {
		t.Fatalf("opts = %+v", opts)
	}
}

func TestParseScorecardArgsDefaults(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	opts, code := parseScorecardArgs([]string{"--providers", "openai,deepseek", "--max-budget-usd=2"})
	if code != -1 {
		t.Fatalf("code = %d, want -1", code)
	}
	if opts.runs != 1 || !nearlyEqual(opts.baselineTolerance, 0.05, 1e-12) || opts.failBelow != -1 {
		t.Fatalf("defaults wrong: %+v", opts)
	}
	if opts.models != nil {
		t.Fatalf("models should default to nil, got %v", opts.models)
	}
	if opts.budgetFlag != "2" {
		t.Fatalf("budget flag = %q, want 2", opts.budgetFlag)
	}
}

func TestParseScorecardArgsUsageErrors(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	tests := [][]string{
		nil,                       // --providers required
		{"--providers", "openai"}, // < 2 providers
		{"--providers", "openai,deepseek", "--models", "m1"}, // length mismatch
		{"--providers", "openai,deepseek", "--fail-below", "1.5"},
		{"--providers", "openai,deepseek", "--fail-below", "abc"},
		{"--providers", "openai,deepseek", "--fail-below"}, // missing value
		{"--providers", "openai,deepseek", "--runs", "0"},
		{"--providers", "openai,deepseek", "--runs", "x"},
		{"--providers", "openai,deepseek", "--baseline-tolerance", "-0.1"},
		{"--providers", "openai,", "x"}, // empty list entry
		{"--providers", "openai,frobnicate"},
		{"--providers", "openai,deepseek", "--bogus"},
	}
	for _, args := range tests {
		_, code := parseScorecardArgs(args)
		if code != 2 {
			t.Fatalf("args %v: code = %d, want 2", args, code)
		}
	}
}

func TestParseScorecardArgsHelp(t *testing.T) {
	_, code := parseScorecardArgs([]string{"--help"})
	if code != 0 {
		t.Fatalf("--help code = %d, want 0", code)
	}
}

func TestScorecardJSONRoundTrip(t *testing.T) {
	fb := 0.8
	sc := scorecard{
		SchemaVersion: 1,
		RunID:         "20260811T150405-openai-deepseek",
		Timestamp:     "2026-08-11T15:04:05Z",
		Suite:         "eval/benchmarks (5 tasks)",
		QuickProfile:  false,
		Runs:          1,
		Gates:         scorecardGates{FailBelow: &fb, MaxBudgetUsd: 5.0, BaselineTolerance: 0.05},
		BaselinePath:  "eval/benchmark-results/scorecard-old.json",
		Providers: []scorecardProvider{
			{Provider: "openai", Model: "openai/gpt-5.6-terra", Passed: 5, Total: 5, Errors: 0, PassRate: 1.0, CostUSD: 0.0412, AvgLatencyMs: 18734.5, Tokens: 128430},
			{Provider: "deepseek", Model: "deepseek/deepseek-v4-flash", Passed: 4, Total: 5, Errors: 1, PassRate: 0.8, CostUSD: 0.0021, AvgLatencyMs: 9430.2, Tokens: 45210},
		},
		Baseline: &scorecardBaseline{
			Path: "eval/benchmark-results/scorecard-old.json",
			Regressions: []scorecardRegression{
				{Provider: "deepseek", Baseline: 1.0, Current: 0.8, Tolerance: 0.05},
			},
		},
		Passed: false,
	}
	b, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	var got scorecard
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, sc) {
		t.Fatalf("round trip mismatch:\n got %+v\nwant %+v", got, sc)
	}
}

func TestScorecardJSONOmitEmptyGates(t *testing.T) {
	sc := scorecard{
		SchemaVersion: 1,
		RunID:         "x",
		Gates:         scorecardGates{BaselineTolerance: 0.05},
		Providers:     []scorecardProvider{{Provider: "openai", PassRate: 1.0}},
		Passed:        true,
	}
	b, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, "maxBudgetUsd") {
		t.Fatalf("unconfigured budget cap must be omitted:\n%s", s)
	}
	if strings.Contains(s, "baselinePath") || strings.Contains(s, "\"baseline\"") {
		t.Fatalf("no-baseline run must omit baseline fields:\n%s", s)
	}
	var got scorecard
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Gates.MaxBudgetUsd != 0 || got.BaselinePath != "" || got.Baseline != nil {
		t.Fatalf("round trip of omitted fields = %+v", got)
	}
}

func TestWriteScorecard(t *testing.T) {
	root := t.TempDir()
	sc := scorecard{
		SchemaVersion: 1,
		RunID:         "20260811T150405-openai-deepseek",
		Suite:         "eval/benchmarks (1 tasks)",
		Gates:         scorecardGates{BaselineTolerance: 0.05},
		Providers:     []scorecardProvider{{Provider: "openai", PassRate: 1.0}},
		Passed:        true,
	}
	path, err := writeScorecard(root, sc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, "benchmark-results") || !strings.Contains(filepath.Base(path), "scorecard-"+sc.RunID) {
		t.Fatalf("path = %q, want under benchmark-results/scorecard-<run>.json", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got scorecard
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.RunID != sc.RunID || got.Passed != true || len(got.Providers) != 1 {
		t.Fatalf("written scorecard mismatch: %+v", got)
	}
}

func TestPrintScorecardTable(t *testing.T) {
	sc := scorecard{
		Providers: []scorecardProvider{
			{Provider: "openai", Model: "openai/gpt-5.6-terra", Passed: 5, Total: 5, Errors: 0, PassRate: 1.0, CostUSD: 0.0412, AvgLatencyMs: 18734.5, Tokens: 128430},
		},
	}
	st := evaluateScorecardGates(sc.Providers, sc.Providers, -1, nil, 0.05, 0)
	var sb strings.Builder
	printScorecardTable(&sb, sc, st)
	for _, want := range []string{"PROVIDER", "MODEL", "PASS-RATE", "COST (USD)", "openai", "TOTAL"} {
		if !strings.Contains(sb.String(), want) {
			t.Fatalf("table missing %q:\n%s", want, sb.String())
		}
	}
}

func TestScorecardRunID(t *testing.T) {
	id := scorecardRunID([]string{"openai", "deepseek"})
	if !strings.HasSuffix(id, "-openai-deepseek") {
		t.Fatalf("run id = %q, want provider suffix", id)
	}
	if len(id) != 15+len("-openai-deepseek") {
		t.Fatalf("run id = %q, want timestamp prefix of 15 chars", id)
	}
}

func TestRunScorecardHelp(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	code, out := captureRunStdout(t, []string{"ci-benchmark", "--help"})
	if code != 0 {
		t.Fatalf("ci-benchmark --help exit = %d, want 0", code)
	}
	if !strings.Contains(out, "Usage: pi-run ci-benchmark") || !strings.Contains(out, "--providers") {
		t.Fatalf("ci-benchmark help missing usage:\n%s", out)
	}
}

func TestRunScorecardUsageErrors(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	for _, args := range [][]string{
		{"ci-benchmark"},
		{"ci-benchmark", "--providers", "openai"},
		{"ci-benchmark", "--providers", "openai,deepseek", "--models", "m1"},
		{"ci-benchmark", "--providers", "openai,deepseek", "--fail-below", "1.5"},
		{"ci-benchmark", "--providers", "openai,deepseek", "--runs", "0"},
		{"ci-benchmark", "--providers", "openai,frobnicate"},
		{"ci-benchmark", "--providers", "openai,deepseek", "--bogus"},
	} {
		code, out := captureRunStderr(t, args)
		if code != 2 {
			t.Fatalf("args %v: exit = %d, want 2; stderr: %s", args, code, out)
		}
	}
}

func TestRunScorecardBaselineBadFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	bad := filepath.Join(root, "baseline.json")
	if err := os.WriteFile(bad, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := captureRunStderr(t, []string{"ci-benchmark", "--providers", "openai,deepseek", "--baseline", bad})
	if code != 2 {
		t.Fatalf("bad baseline exit = %d, want 2; stderr: %s", code, out)
	}
	if !strings.Contains(out, "baseline") {
		t.Fatalf("bad baseline stderr missing baseline error: %s", out)
	}
}

func TestRunScorecardNoDocker(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	for _, key := range supportedProviderKeyEnvs {
		t.Setenv(key, "")
	}
	t.Setenv("PI_PROVIDER", "")
	t.Setenv("PI_MAX_BUDGET_USD", "")
	t.Setenv("BW_GET", filepath.Join(t.TempDir(), "nonexistent-bw-get"))
	writeBenchmarkTask(t, root, "demo", `{"id": "demo", "prompt": "Fix it."}`)
	// Hide docker from PATH: the first provider run must fail fast with exit 7
	// before any key/node resolution, exactly like eval --benchmark.
	t.Setenv("PATH", t.TempDir())
	code, out := captureRunStderr(t, []string{"ci-benchmark", "--providers", "openai,deepseek"})
	if code != 7 {
		t.Fatalf("ci-benchmark no-docker exit = %d, want 7; stderr:\n%s", code, out)
	}
	if !strings.Contains(out, "Docker") {
		t.Fatalf("stderr missing docker message:\n%s", out)
	}
}

func TestExitCodesTableDocumentsScorecard(t *testing.T) {
	if !strings.Contains(exitCodesText, "8  scorecard gate failed") {
		t.Fatalf("exit-codes table must document 8 = scorecard gate failed:\n%s", exitCodesText)
	}
	if !strings.Contains(usage, "8 scorecard gate failed") {
		t.Fatalf("usage exit-code line must mention 8 = scorecard gate failed")
	}
	if !strings.Contains(usage, "ci-benchmark") {
		t.Fatalf("usage must document the ci-benchmark command")
	}
}
