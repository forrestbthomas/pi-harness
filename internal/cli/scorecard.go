package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

// exitScorecardFailed is the exit code for a failed quality gate: fail-below,
// baseline regression, or an incomplete run (some tasks errored). Documented
// in the --exit-codes table alongside codes 0-7.
const exitScorecardFailed = 8

// quickProfileAgentTimeout caps each task's agent timeout under
// --quick-profile, for cheap scheduled smoke runs.
const quickProfileAgentTimeout = 60

// scorecardUsage documents the `pi-run ci-benchmark` command.
const scorecardUsage = `Usage: pi-run ci-benchmark --providers <a,b> [--models <m1,m2>] [flags]

Run the benchmark suite against 2+ providers and gate on the scorecard.
  --providers <a,b>        Comma-separated providers (>= 2; order-significant).
  --models <m1,m2>         Optional per-provider model overrides (same order as --providers).
  --fail-below <rate>      Fail if any provider pass rate < rate (e.g. 0.8).
  --max-budget-usd <n>     Fail (exit 6) if total run cost >= n. PI_MAX_BUDGET_USD also applies.
  --baseline <path>        Previous scorecard/run JSON to diff pass rates against.
  --baseline-tolerance <n> Max allowed per-provider pass-rate drop vs baseline (default 0.05).
  --runs <n>               Repeat each provider suite n times; gate on median pass rate (default 1).
  --quick-profile          Cap per-task agent timeout at 60s (cheap, best-effort smoke run).
`

// scorecardOptions is the parsed result of ci-benchmark's owned flags.
type scorecardOptions struct {
	providers         []string
	models            []string // empty → each provider's defaultModel
	failBelow         float64  // -1 when --fail-below unset
	budgetFlag        string
	baselinePath      string
	baselineTolerance float64
	runs              int
	quickProfile      bool
}

// scorecardProvider is one aggregated per-provider row (schema §4.3).
type scorecardProvider struct {
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	Passed       int     `json:"passed"`
	Total        int     `json:"total"`
	Errors       int     `json:"errors"`
	PassRate     float64 `json:"passRate"`
	CostUSD      float64 `json:"costUsd"`
	AvgLatencyMs float64 `json:"avgLatencyMs"`
	Tokens       int     `json:"tokens"`
	// Cost-per-task metrics (spec §4.5), derived from the base fields by
	// derivePerTaskMetrics. The cost fields are agent-only: the Docker
	// benchmark grades deterministically, so there is no judge here and
	// judgeCostUsd is always 0 (and thus omitted from JSON).
	CostPerTaskUsd           float64 `json:"costPerTaskUsd"`
	CostPerSuccessfulTaskUsd float64 `json:"costPerSuccessfulTaskUsd"`
	TokensPerTask            int     `json:"tokensPerTask"`
	AgentCostUsd             float64 `json:"agentCostUsd"`
	JudgeCostUsd             float64 `json:"judgeCostUsd,omitempty"`
}

// scorecardGates captures the gate configuration in effect for the run.
type scorecardGates struct {
	FailBelow         *float64 `json:"failBelow,omitempty"`
	MaxBudgetUsd      float64  `json:"maxBudgetUsd,omitempty"`
	BaselineTolerance float64  `json:"baselineTolerance"`
}

// scorecardRegression is one per-provider baseline comparison that regressed.
type scorecardRegression struct {
	Provider  string  `json:"provider"`
	Baseline  float64 `json:"baseline"`
	Current   float64 `json:"current"`
	Tolerance float64 `json:"tolerance"`
}

// scorecardBaseline is the baseline block of the scorecard artifact.
type scorecardBaseline struct {
	Path        string                `json:"path"`
	Regressions []scorecardRegression `json:"regressions"`
}

// scorecard is the full on-disk artifact written to
// eval/benchmark-results/scorecard-<run>.json (schema §4.3).
type scorecard struct {
	SchemaVersion int                 `json:"schemaVersion"`
	RunID         string              `json:"runId"`
	Timestamp     string              `json:"timestamp"`
	Suite         string              `json:"suite"`
	QuickProfile  bool                `json:"quickProfile"`
	Runs          int                 `json:"runs"`
	Gates         scorecardGates      `json:"gates"`
	BaselinePath  string              `json:"baselinePath,omitempty"`
	Providers     []scorecardProvider `json:"providers"`
	Baseline      *scorecardBaseline  `json:"baseline,omitempty"`
	Passed        bool                `json:"passed"`
}

// scorecardGateStatus captures which gates failed and the totals behind them.
type scorecardGateStatus struct {
	Incomplete     bool
	BelowFailFloor bool
	Regressions    []scorecardRegression
	BudgetExceeded bool
	// TotalCostUSD is the display total: the sum of the collapsed (median)
	// per-provider costs, shown in the scorecard table and budget diagnostic.
	TotalCostUSD float64
	// ActualSpendUSD is the gate's cost basis: the sum of EVERY raw per-run
	// row's cost delta across all providers and repeats. With --runs n this
	// equals the real spend (~n × median per provider), so the budget gate
	// catches a breach at the actual spend level even when the median rows
	// look cheap. Equal to TotalCostUSD when runs == 1.
	ActualSpendUSD float64
}

// splitList splits a comma-separated flag value into trimmed entries,
// rejecting empty entries so "--providers openai," cannot silently skip one.
func splitList(v string) ([]string, error) {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, fmt.Errorf("empty entry in list %q", v)
		}
		out = append(out, p)
	}
	return out, nil
}

// scorecardUsageError prints a usage error plus the command help.
func scorecardUsageError(msg string) (scorecardOptions, int) {
	fmt.Fprintf(os.Stderr, "pi-run: ci-benchmark: %s\n\n%s", msg, scorecardUsage)
	return scorecardOptions{}, 2
}

// parseScorecardArgs validates ci-benchmark's flags. A returned exitCode >= 0
// means parsing failed (or --help) and the caller should return it; -1 means
// "run the command". Usage errors exit 2 (provider resolution, list lengths,
// rate ranges, run counts). The --baseline file itself is parsed later in
// runScorecard (it needs no flags, only the path).
func parseScorecardArgs(args []string) (opts scorecardOptions, exitCode int) {
	opts.runs = 1
	opts.baselineTolerance = 0.05
	opts.failBelow = -1 // unset
	i := 0
	for i < len(args) {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			fmt.Print(scorecardUsage)
			return opts, 0
		case arg == "--providers":
			if i+1 >= len(args) {
				return scorecardUsageError("--providers requires a comma-separated list")
			}
			i++
			list, err := splitList(args[i])
			if err != nil {
				return scorecardUsageError(err.Error())
			}
			opts.providers = list
		case strings.HasPrefix(arg, "--providers="):
			list, err := splitList(strings.TrimPrefix(arg, "--providers="))
			if err != nil {
				return scorecardUsageError(err.Error())
			}
			opts.providers = list
		case arg == "--models":
			if i+1 >= len(args) {
				return scorecardUsageError("--models requires a comma-separated list")
			}
			i++
			list, err := splitList(args[i])
			if err != nil {
				return scorecardUsageError(err.Error())
			}
			opts.models = list
		case strings.HasPrefix(arg, "--models="):
			list, err := splitList(strings.TrimPrefix(arg, "--models="))
			if err != nil {
				return scorecardUsageError(err.Error())
			}
			opts.models = list
		case arg == "--fail-below":
			if i+1 >= len(args) {
				return scorecardUsageError("--fail-below requires a rate in [0,1]")
			}
			i++
			f, err := strconv.ParseFloat(args[i], 64)
			if err != nil || f < 0 || f > 1 {
				return scorecardUsageError(fmt.Sprintf("invalid --fail-below %q: must be a rate in [0,1]", args[i]))
			}
			opts.failBelow = f
		case strings.HasPrefix(arg, "--fail-below="):
			v := strings.TrimPrefix(arg, "--fail-below=")
			f, err := strconv.ParseFloat(v, 64)
			if err != nil || f < 0 || f > 1 {
				return scorecardUsageError(fmt.Sprintf("invalid --fail-below %q: must be a rate in [0,1]", v))
			}
			opts.failBelow = f
		case arg == "--max-budget-usd":
			if i+1 >= len(args) {
				return scorecardUsageError("--max-budget-usd requires a number")
			}
			i++
			opts.budgetFlag = args[i]
		case strings.HasPrefix(arg, "--max-budget-usd="):
			opts.budgetFlag = strings.TrimPrefix(arg, "--max-budget-usd=")
		case arg == "--baseline":
			if i+1 >= len(args) {
				return scorecardUsageError("--baseline requires a file path")
			}
			i++
			opts.baselinePath = args[i]
		case strings.HasPrefix(arg, "--baseline="):
			opts.baselinePath = strings.TrimPrefix(arg, "--baseline=")
		case arg == "--baseline-tolerance":
			if i+1 >= len(args) {
				return scorecardUsageError("--baseline-tolerance requires a rate in [0,1]")
			}
			i++
			f, err := strconv.ParseFloat(args[i], 64)
			if err != nil || f < 0 || f > 1 {
				return scorecardUsageError(fmt.Sprintf("invalid --baseline-tolerance %q: must be in [0,1]", args[i]))
			}
			opts.baselineTolerance = f
		case strings.HasPrefix(arg, "--baseline-tolerance="):
			v := strings.TrimPrefix(arg, "--baseline-tolerance=")
			f, err := strconv.ParseFloat(v, 64)
			if err != nil || f < 0 || f > 1 {
				return scorecardUsageError(fmt.Sprintf("invalid --baseline-tolerance %q: must be in [0,1]", v))
			}
			opts.baselineTolerance = f
		case arg == "--runs":
			if i+1 >= len(args) {
				return scorecardUsageError("--runs requires a positive integer")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 1 {
				return scorecardUsageError(fmt.Sprintf("invalid --runs %q: must be >= 1", args[i]))
			}
			opts.runs = n
		case strings.HasPrefix(arg, "--runs="):
			v := strings.TrimPrefix(arg, "--runs=")
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				return scorecardUsageError(fmt.Sprintf("invalid --runs %q: must be >= 1", v))
			}
			opts.runs = n
		case arg == "--quick-profile":
			opts.quickProfile = true
		default:
			return scorecardUsageError(fmt.Sprintf("unknown flag or argument %q", arg))
		}
		i++
	}
	// Cross-flag validation (usage errors exit 2).
	if len(opts.providers) == 0 {
		return scorecardUsageError("--providers is required (comma-separated, >= 2 providers)")
	}
	if len(opts.providers) < 2 {
		return scorecardUsageError("--providers requires at least 2 providers (the scorecard compares providers)")
	}
	if len(opts.models) > 0 && len(opts.models) != len(opts.providers) {
		return scorecardUsageError(fmt.Sprintf("--models has %d entries but --providers has %d", len(opts.models), len(opts.providers)))
	}
	for _, name := range opts.providers {
		if _, err := ResolveProvider(name); err != nil {
			return scorecardUsageError(err.Error())
		}
	}
	seen := make(map[string]bool, len(opts.providers))
	for _, name := range opts.providers {
		if seen[name] {
			return scorecardUsageError(fmt.Sprintf("duplicate provider %q in --providers", name))
		}
		seen[name] = true
	}
	return opts, -1
}

// runScorecard implements `pi-run ci-benchmark`: run the benchmark suite
// against each provider sequentially, aggregate a per-provider scorecard, gate
// on fail-below / baseline regression / budget, write the JSON artifact, and
// print a human table.
func runScorecard(args []string) int {
	opts, parseCode := parseScorecardArgs(args)
	if parseCode >= 0 {
		return parseCode
	}
	root := repoRoot()

	budgetCap, err := resolveBudgetCap(opts.budgetFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: ci-benchmark: %v\n\n%s", err, scorecardUsage)
		return 2
	}
	var baseline map[string]float64
	if opts.baselinePath != "" {
		baseline, err = parseBaseline(opts.baselinePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pi-run: ci-benchmark: %v\n\n%s", err, scorecardUsage)
			return 2
		}
	}

	tasks, errs := loadBenchmarkTasks(root, "")
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "pi-run: ci-benchmark: %v\n", e)
		}
		return 1
	}

	// Providers run sequentially so per-provider cost attribution stays clean.
	rows := make([]scorecardProvider, 0, len(opts.providers))
	var repeats []scorecardProvider
	for idx, name := range opts.providers {
		p, _ := ResolveProvider(name) // already validated in parseScorecardArgs
		model := p.DefaultModel
		if len(opts.models) > 0 {
			model = opts.models[idx]
		}
		row, rs, err := runScorecardProvider(tasks, p, model, root, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pi-run: ci-benchmark: %v\n", err)
			return benchmarkErrorCode(err)
		}
		rows = append(rows, row)
		repeats = append(repeats, rs...)
	}

	st := evaluateScorecardGates(rows, repeats, opts.failBelow, baseline, opts.baselineTolerance, budgetCap)
	code := scorecardExitCode(st)
	if baseline != nil {
		for bprov := range baseline {
			found := false
			for _, r := range rows {
				if r.Provider == bprov {
					found = true
					break
				}
			}
			if !found {
				fmt.Fprintf(os.Stderr, "pi-run: ci-benchmark: warning: baseline provider %s missing from this run — not gated\n", bprov)
			}
		}
	}

	sc := buildScorecard(opts, rows, tasks, baseline, budgetCap, st, code)
	path, err := writeScorecard(root, sc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: ci-benchmark: %v\n", err)
		return 1
	}
	fmt.Printf("scorecard written to %s\n", path)
	if _, err := writeScorecardLatest(root, sc); err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: ci-benchmark: %v\n", err)
		return 1
	}

	printScorecardTable(os.Stdout, sc, st)

	// Gate diagnostics, in the §4.4 ordering (incomplete, fail-below,
	// regression, budget).
	if st.Incomplete {
		fmt.Fprintln(os.Stderr, "pi-run: ci-benchmark: gate failed: run incomplete (some tasks errored)")
	}
	if st.BelowFailFloor {
		fmt.Fprintf(os.Stderr, "pi-run: ci-benchmark: gate failed: a provider pass rate is below %.2f\n", opts.failBelow)
	}
	for _, r := range st.Regressions {
		fmt.Fprintf(os.Stderr, "pi-run: ci-benchmark: gate failed: %s regressed %.0f%% -> %.0f%% (tolerance %.2f)\n", r.Provider, r.Baseline*100, r.Current*100, r.Tolerance)
	}
	if st.BudgetExceeded {
		fmt.Fprintf(os.Stderr, "pi-run: ci-benchmark: budget exceeded: total run cost $%.6f >= cap $%.6f\n", st.ActualSpendUSD, budgetCap)
	}
	return code
}

// runScorecardProvider executes one provider's suite opts.runs times and
// returns the collapsed median row plus every raw per-run row (the
// incomplete-run gate reads the raw rows so an errored repeat cannot hide
// behind the median collapse). Per repeat: snapshot spend, run the provider
// benchmark, record the spend delta with ledger mode "benchmark", and compute
// the row from the graded result.
func runScorecardProvider(tasks []benchmarkTask, p Provider, model, root string, opts scorecardOptions) (scorecardProvider, []scorecardProvider, error) {
	var repeats []scorecardProvider
	for i := 0; i < opts.runs; i++ {
		runTasks := tasks
		if opts.quickProfile {
			runTasks = quickProfileTasks(tasks)
		}
		pre, err := currentSpend(root)
		if err != nil {
			return scorecardProvider{}, repeats, err
		}
		// Cost attribution note: benchmark agent runs use --no-session
		// (persist=false in piArgs), so in CI they write no session files and
		// the ledger cost delta is ~0 — the budget gate is then primarily
		// meaningful for local runs where sessions persist, or when --runs
		// repeats accumulate attributable spend. This matches the cost spec's
		// no-estimation rule; the pass-rate and baseline gates are unaffected.
		start := time.Now()
		res, err := runProviderBenchmark(runTasks, evalOptions{provider: p.Name, model: model}, root)
		if err != nil {
			return scorecardProvider{}, repeats, err
		}
		beforeEntries, err := ledgerEntries(root)
		if err != nil {
			return scorecardProvider{}, repeats, err
		}
		post, rerr := recordRunSpend(root, start, "benchmark", p.Name, model, pre)
		if rerr != nil {
			return scorecardProvider{}, repeats, rerr
		}
		row := scorecardProviderFromRun(res, post-pre, scorecardRunTokens(root, beforeEntries, p.Name, model))
		repeats = append(repeats, row)
	}
	return collapseScorecardRuns(repeats), repeats, nil
}

// quickProfileTasks caps each task's agent timeout at 60s for cheap scheduled
// smoke runs. The returned slice copies task values (TimeoutSecs may change);
// the underlying task data is shared.
func quickProfileTasks(tasks []benchmarkTask) []benchmarkTask {
	out := make([]benchmarkTask, len(tasks))
	for i, t := range tasks {
		out[i] = t
		if t.TimeoutSecs > quickProfileAgentTimeout {
			out[i].TimeoutSecs = quickProfileAgentTimeout
		}
	}
	return out
}

// scorecardProviderFromRun converts one graded provider run into an aggregated
// scorecard row: pass rate and errors from the run summary, avg latency from
// the mean per-task durationSecs × 1000, cost and tokens from the ledger.
func scorecardProviderFromRun(res benchmarkRunResult, costUSD float64, tokens int) scorecardProvider {
	row := scorecardProvider{
		Provider: res.Provider,
		Model:    res.Model,
		Passed:   res.Summary.Passed,
		Total:    res.Summary.Total,
		Errors:   res.Summary.Errors,
		PassRate: res.Summary.Score,
		CostUSD:  costUSD,
		Tokens:   tokens,
	}
	var durSum float64
	for _, tr := range res.Tasks {
		durSum += tr.Duration
	}
	if len(res.Tasks) > 0 {
		row.AvgLatencyMs = durSum / float64(len(res.Tasks)) * 1000
	}
	row.derivePerTaskMetrics()
	return row
}

// derivePerTaskMetrics populates the §4.5 cost-per-task fields from the row's
// base fields: costPerTaskUsd = costUsd / total, tokensPerTask = tokens /
// total, costPerSuccessfulTaskUsd = costUsd / passed, agentCostUsd = the
// existing costUsd (stable name for the agent/judge split). Division by zero
// is guarded: any zero denominator yields 0. judgeCostUsd stays 0 — the Go
// scorecard grades with deterministic Docker tests, so there is no judge and
// the field is omitted from JSON via omitempty.
func (r *scorecardProvider) derivePerTaskMetrics() {
	r.AgentCostUsd = r.CostUSD
	r.JudgeCostUsd = 0
	r.CostPerTaskUsd = 0
	r.TokensPerTask = 0
	r.CostPerSuccessfulTaskUsd = 0
	if r.Total > 0 {
		r.CostPerTaskUsd = r.CostUSD / float64(r.Total)
		r.TokensPerTask = r.Tokens / r.Total
	}
	if r.Passed > 0 {
		r.CostPerSuccessfulTaskUsd = r.CostUSD / float64(r.Passed)
	}
}

// ledgerEntryCounts indexes ledger entries by value so scorecardRunTokens can
// tell which entries a recordRunSpend call appended.
func ledgerEntryCounts(entries []ledgerEntry) map[ledgerEntry]int {
	counts := make(map[ledgerEntry]int, len(entries))
	for _, e := range entries {
		counts[e]++
	}
	return counts
}

// scorecardRunTokens returns the input+output tokens the ledger attributes to
// this provider/model's run: entries with mode "benchmark" for the
// provider/model that were appended since the pre-run snapshot (beforeEntries).
// Runs that leave no session files carry a fallback ledger entry with unknown
// (0) tokens, matching the cost spec's no-estimation rule.
func scorecardRunTokens(root string, beforeEntries []ledgerEntry, provider, model string) int {
	before := ledgerEntryCounts(beforeEntries)
	after, err := ledgerEntries(root)
	if err != nil {
		return 0
	}
	tokens := 0
	counts := map[ledgerEntry]int{}
	for _, e := range after {
		if e.Mode != "benchmark" || e.Provider != provider || e.Model != model {
			continue
		}
		counts[e]++
		if counts[e] > before[e] {
			tokens += e.InputTokens + e.OutputTokens
		}
	}
	return tokens
}

// medianFloats returns the median of vals: the middle value for an odd count,
// the average of the two middle values for an even count. Empty input yields 0.
func medianFloats(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := append([]float64(nil), vals...)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

// medianInts is medianFloats for integers (the even midpoint rounds down).
func medianInts(vals []int) int {
	if len(vals) == 0 {
		return 0
	}
	sorted := append([]int(nil), vals...)
	sort.Ints(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

// collapseScorecardRuns reduces n per-run rows into one median row: each
// metric reports its median across the runs (flakiness mitigation). passed/
// total/errors come from the repeat whose pass rate is closest to the median
// pass rate so the row stays self-consistent; ties pick the earliest repeat.
//
// Display note: with runs > 1 the reported passed/total come from that
// representative repeat while PassRate is the median, so the table can show
// e.g. PASSED 4 TOTAL 5 PASS-RATE 0.90 (4/5 != 0.90). This is display-only;
// the gate reads PassRate directly and is unaffected.
func collapseScorecardRuns(rows []scorecardProvider) scorecardProvider {
	if len(rows) == 0 {
		return scorecardProvider{}
	}
	if len(rows) == 1 {
		return rows[0]
	}
	passRates := make([]float64, 0, len(rows))
	costs := make([]float64, 0, len(rows))
	lats := make([]float64, 0, len(rows))
	tokens := make([]int, 0, len(rows))
	for _, r := range rows {
		passRates = append(passRates, r.PassRate)
		costs = append(costs, r.CostUSD)
		lats = append(lats, r.AvgLatencyMs)
		tokens = append(tokens, r.Tokens)
	}
	median := medianFloats(passRates)
	best := rows[0]
	for _, r := range rows {
		if absFloat(r.PassRate-median) < absFloat(best.PassRate-median) {
			best = r
		}
	}
	out := best
	out.PassRate = median
	out.CostUSD = medianFloats(costs)
	out.AvgLatencyMs = medianFloats(lats)
	out.Tokens = medianInts(tokens)
	// The median collapse replaced the base fields, so the derived per-task
	// metrics must be recomputed from them (a stale representative repeat's
	// values would otherwise survive the collapse).
	out.derivePerTaskMetrics()
	return out
}

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// baselineRegressions compares current rows against the baseline pass rates: a
// provider present in both sides regresses when current < baseline - tolerance
// (strict, so the tolerance boundary passes). Providers present in only one
// side are reported to the caller but never fail the gate.
func baselineRegressions(baseline map[string]float64, rows []scorecardProvider, tolerance float64) []scorecardRegression {
	var regs []scorecardRegression
	for _, r := range rows {
		b, ok := baseline[r.Provider]
		if !ok {
			continue
		}
		if r.PassRate < b-tolerance {
			regs = append(regs, scorecardRegression{
				Provider:  r.Provider,
				Baseline:  b,
				Current:   r.PassRate,
				Tolerance: tolerance,
			})
		}
	}
	return regs
}

// evaluateScorecardGates applies the ci-benchmark gates (§4.4) to the collapsed
// per-provider rows. repeats carries every per-run row across providers so an
// errored repeat never hides behind the median collapse, and so the budget gate
// sees the ACTUAL spend (sum of all repeats) rather than the median rows.
// failBelow <= 0 disables the fail-below gate, baseline nil disables the
// regression gate, and budgetCap <= 0 disables the budget gate.
func evaluateScorecardGates(rows []scorecardProvider, repeats []scorecardProvider, failBelow float64, baseline map[string]float64, tolerance float64, budgetCap float64) scorecardGateStatus {
	var st scorecardGateStatus
	for _, r := range rows {
		st.TotalCostUSD += r.CostUSD
	}
	for _, r := range repeats {
		st.ActualSpendUSD += r.CostUSD
		if r.Errors > 0 {
			st.Incomplete = true
		}
	}
	for _, r := range rows {
		if failBelow > 0 && r.PassRate < failBelow {
			st.BelowFailFloor = true
		}
	}
	if baseline != nil {
		st.Regressions = baselineRegressions(baseline, rows, tolerance)
	}
	// Gate on ACTUAL spend (all repeats) so --runs n cannot hide a real cost
	// breach behind the median collapse. TotalCostUSD stays the display total.
	if budgetCap > 0 && st.ActualSpendUSD >= budgetCap {
		st.BudgetExceeded = true
	}
	return st
}

// scorecardExitCode maps the gate status to the documented exit code: 8 for
// the quality gates (incomplete, fail-below, baseline regression), 6 for the
// budget gate, 0 otherwise. Ordering matches §4.4 so CI can distinguish causes.
func scorecardExitCode(st scorecardGateStatus) int {
	switch {
	case st.Incomplete, st.BelowFailFloor, len(st.Regressions) > 0:
		return exitScorecardFailed
	case st.BudgetExceeded:
		return exitBudgetExceeded
	default:
		return 0
	}
}

// scorecardNow is a package-level seam so tests can pin the scorecard
// timestamp and run ID. Production behavior is unchanged.
var scorecardNow = time.Now

// buildScorecard assembles the scorecard artifact from the run data.
func buildScorecard(opts scorecardOptions, rows []scorecardProvider, tasks []benchmarkTask, baseline map[string]float64, budgetCap float64, st scorecardGateStatus, code int) scorecard {
	sc := scorecard{
		SchemaVersion: 1,
		RunID:         scorecardRunID(opts.providers),
		Timestamp:     scorecardNow().UTC().Format(time.RFC3339),
		Suite:         fmt.Sprintf("eval/benchmarks (%d tasks)", len(tasks)),
		QuickProfile:  opts.quickProfile,
		Runs:          opts.runs,
		Gates: scorecardGates{
			MaxBudgetUsd:      budgetCap,
			BaselineTolerance: opts.baselineTolerance,
		},
		BaselinePath: opts.baselinePath,
		Providers:    rows,
		Passed:       code == 0,
	}
	if opts.failBelow >= 0 {
		fb := opts.failBelow
		sc.Gates.FailBelow = &fb
	}
	if baseline != nil {
		sc.Baseline = &scorecardBaseline{Path: opts.baselinePath, Regressions: st.Regressions}
	}
	return sc
}

// scorecardRunID builds the scorecard filename stem: timestamp + joined
// provider names, e.g. 20260811T150405-openai-deepseek.
func scorecardRunID(providers []string) string {
	return scorecardNow().Format("20060102T150405") + "-" + strings.Join(providers, "-")
}

// writeScorecard writes the scorecard JSON under eval/benchmark-results/
// (gitignored) and returns its path.
func writeScorecard(root string, sc scorecard) (string, error) {
	dir := filepath.Join(root, "eval", "benchmark-results")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	path := filepath.Join(dir, "scorecard-"+sc.RunID+".json")
	b, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// writeScorecardLatest mirrors writeScorecard but writes the fixed-name
// pointer file eval/benchmark-results/scorecard-latest.json (byte-identical to
// the most recent scorecard-<run>.json). The CLI owns the artifact directory,
// so the latest pointer is hermetic-testable and needs no shell glue.
func writeScorecardLatest(root string, sc scorecard) (string, error) {
	dir := filepath.Join(root, "eval", "benchmark-results")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	path := filepath.Join(dir, "scorecard-latest.json")
	b, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// printScorecardTable renders the human-readable per-provider scorecard table.
func printScorecardTable(w io.Writer, sc scorecard, st scorecardGateStatus) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PROVIDER\tMODEL\tPASSED\tTOTAL\tERRORS\tPASS-RATE\tCOST (USD)\tAVG LATENCY (MS)\tTOKENS")
	for _, row := range sc.Providers {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%d\t%.2f\t$%.6f\t%.1f\t%d\n",
			row.Provider, row.Model, row.Passed, row.Total, row.Errors, row.PassRate, row.CostUSD, row.AvgLatencyMs, row.Tokens)
	}
	var passed, total, errors, tokens int
	for _, row := range sc.Providers {
		passed += row.Passed
		total += row.Total
		errors += row.Errors
		tokens += row.Tokens
	}
	rate := 0.0
	if total > 0 {
		rate = float64(passed) / float64(total)
	}
	fmt.Fprintf(tw, "TOTAL\t\t%d\t%d\t%d\t%.2f\t$%.6f\t-\t%d\n", passed, total, errors, rate, st.TotalCostUSD, tokens)
	tw.Flush()
}

// baselineFile is the union shape used to sniff --baseline files: either a
// prior scorecard (schemaVersion 1, providers[].{provider, passRate}) or a
// prior per-provider run JSON (the shipped benchmarkRunResult shape
// {provider, summary.score}).
type baselineFile struct {
	SchemaVersion int `json:"schemaVersion"`
	Providers     []struct {
		Provider string  `json:"provider"`
		PassRate float64 `json:"passRate"`
	} `json:"providers"`
	Provider string            `json:"provider"`
	Summary  *benchmarkSummary `json:"summary"`
}

// parseBaseline reads a prior scorecard or per-provider run JSON and returns
// per-provider pass rates. The scorecard shape wins when both are present.
func parseBaseline(path string) (map[string]float64, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("baseline %s: %v", path, err)
	}
	var f baselineFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("baseline %s: %v", path, err)
	}
	baseline := map[string]float64{}
	if len(f.Providers) > 0 {
		if f.SchemaVersion != 0 && f.SchemaVersion != 1 {
			return nil, fmt.Errorf("baseline %s: unsupported scorecard schemaVersion %d (want 1)", path, f.SchemaVersion)
		}
		for _, p := range f.Providers {
			if p.Provider == "" {
				return nil, fmt.Errorf("baseline %s: scorecard provider entry missing \"provider\"", path)
			}
			baseline[p.Provider] = p.PassRate
		}
		return baseline, nil
	}
	if f.Summary != nil {
		if f.Provider == "" {
			return nil, fmt.Errorf("baseline %s: run JSON missing \"provider\"", path)
		}
		baseline[f.Provider] = f.Summary.Score
		return baseline, nil
	}
	return nil, fmt.Errorf("baseline %s: unrecognized shape: expected a scorecard (schemaVersion 1 with providers[]) or a benchmark run JSON (provider + summary.score)", path)
}
