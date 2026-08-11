package cli

import (
	"bufio"
	"bytes"
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

// exitBudgetExceeded is the exit code for a pre-flight budget-cap refusal.
// Documented in the --exit-codes table alongside codes 0-5.
const exitBudgetExceeded = 6

// costUsage documents the `pi-run cost` command.
const costUsage = `Usage: pi-run cost [--json] [--since <date>] [--reset]

Aggregate real spend from Pi session files (.pi/sessions/*.jsonl).
  --json         Machine-readable JSON output
  --since <date> Only count sessions modified at/after <date> (YYYY-MM-DD or RFC3339)
  --reset        Archive the spend ledger (.pi/cost-ledger.jsonl) and start a fresh budget period
`

// sessionLine is a single JSONL record from a Pi session file. Only "message"
// records carry usage/cost; provider/model live on the nested message body.
type sessionLine struct {
	Type    string      `json:"type"`
	Message *sessionMsg `json:"message"`
}

type sessionMsg struct {
	Provider string    `json:"provider"`
	Model    string    `json:"model"`
	Usage    *msgUsage `json:"usage"`
}

type msgUsage struct {
	Input       int      `json:"input"`
	Output      int      `json:"output"`
	TotalTokens int      `json:"totalTokens"`
	Cost        *msgCost `json:"cost"`
}

type msgCost struct {
	Total float64 `json:"total"`
}

// costSample is one per-message usage record with its provider/model.
type costSample struct {
	Provider     string
	Model        string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	CostUSD      float64
}

// costRow is one aggregated provider/model group in a cost report.
type costRow struct {
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	TotalTokens  int     `json:"totalTokens"`
	CostUSD      float64 `json:"costUsd"`
	Sessions     int     `json:"sessions"`
}

// costReport is the full aggregated result of a session scan.
type costReport struct {
	Rows         []costRow `json:"rows"`
	TotalCostUSD float64   `json:"totalCostUsd"`
	TotalTokens  int       `json:"totalTokens"`
	Sessions     int       `json:"sessions"`
}

// ledgerEntry is one append-only spend record. Mode is one of "chat", "print",
// "resume" (the pi-run command that produced the spend) or "backfill" (spend
// predating the ledger, captured on first use).
type ledgerEntry struct {
	TS           string  `json:"ts"` // RFC3339
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	CostUSD      float64 `json:"costUsd"`
	Mode         string  `json:"mode"`
}

// ledgerPath returns the append-only spend ledger path.
func ledgerPath(root string) string {
	return filepath.Join(root, ".pi", "cost-ledger.jsonl")
}

// resetMarkerPath returns the marker written by `pi-run cost --reset`.
func resetMarkerPath(root string) string {
	return filepath.Join(root, ".pi", "cost-ledger.reset")
}

// scanSessionFile returns usage samples from one session JSONL file. Lines
// that are not assistant "message" records, or whose message has no usage.cost
// (some providers/replies report no cost), are skipped. Malformed or truncated
// lines are skipped so a partially-written session never breaks the report.
func scanSessionFile(path string) ([]costSample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var samples []costSample
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			var rec sessionLine
			if json.Unmarshal(line, &rec) != nil {
				continue // unparseable line (schema drift / partial write)
			}
			if rec.Type != "message" || rec.Message == nil || rec.Message.Usage == nil || rec.Message.Usage.Cost == nil {
				continue
			}
			samples = append(samples, costSample{
				Provider:     rec.Message.Provider,
				Model:        rec.Message.Model,
				InputTokens:  rec.Message.Usage.Input,
				OutputTokens: rec.Message.Usage.Output,
				TotalTokens:  rec.Message.Usage.TotalTokens,
				CostUSD:      rec.Message.Usage.Cost.Total,
			})
		}
		if err != nil {
			break // EOF (or read error; best-effort report)
		}
	}
	return samples, nil
}

// sessionFiles returns every session JSONL under <root>/.pi/sessions,
// including subagent child sessions in nested directories. A missing sessions
// directory yields no files, not an error.
func sessionFiles(root string) ([]string, error) {
	dir := filepath.Join(root, ".pi", "sessions")
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []string
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil // skip unreadable entries; the report is best-effort
		}
		if strings.HasSuffix(d.Name(), ".jsonl") {
			files = append(files, path)
		}
		return nil
	})
	return files, nil
}

// collectCostReport scans session files under root and aggregates usage into a
// report. When since is non-zero, only session files modified at/after since
// are counted. Rows are sorted by cost (descending), then provider/model.
func collectCostReport(root string, since time.Time) (costReport, error) {
	files, err := sessionFiles(root)
	if err != nil {
		return costReport{}, err
	}
	rows := map[string]*costRow{}
	rowFiles := map[string]map[string]bool{}
	seenSessions := map[string]bool{}
	for _, f := range files {
		if !since.IsZero() {
			info, err := os.Stat(f)
			if err != nil || info.ModTime().Before(since) {
				continue
			}
		}
		samples, err := scanSessionFile(f)
		if err != nil || len(samples) == 0 {
			continue
		}
		seenSessions[f] = true
		for _, s := range samples {
			key := s.Provider + "\x00" + s.Model
			r := rows[key]
			if r == nil {
				r = &costRow{Provider: s.Provider, Model: s.Model}
				rows[key] = r
				rowFiles[key] = map[string]bool{}
			}
			r.InputTokens += s.InputTokens
			r.OutputTokens += s.OutputTokens
			r.TotalTokens += s.TotalTokens
			r.CostUSD += s.CostUSD
			rowFiles[key][f] = true
		}
	}
	report := costReport{}
	for key, r := range rows {
		r.Sessions = len(rowFiles[key])
		report.Rows = append(report.Rows, *r)
		report.TotalCostUSD += r.CostUSD
		report.TotalTokens += r.TotalTokens
	}
	report.Sessions = len(seenSessions)
	sort.Slice(report.Rows, func(i, j int) bool {
		if report.Rows[i].CostUSD != report.Rows[j].CostUSD {
			return report.Rows[i].CostUSD > report.Rows[j].CostUSD
		}
		return report.Rows[i].Provider+"/"+report.Rows[i].Model < report.Rows[j].Provider+"/"+report.Rows[j].Model
	})
	return report, nil
}

// parseSinceDate parses the --since value: YYYY-MM-DD or RFC3339. Empty means
// no filter.
func parseSinceDate(arg string) (time.Time, error) {
	if arg == "" {
		return time.Time{}, nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		if t, err := time.Parse(layout, arg); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid --since date %q: use YYYY-MM-DD or RFC3339", arg)
}

// runCost implements `pi-run cost`.
func runCost(args []string) int {
	jsonOut := false
	reset := false
	sinceArg := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			jsonOut = true
		case a == "--reset":
			reset = true
		case a == "--since":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "pi-run: cost: --since requires a date\n\n%s", costUsage)
				return 2
			}
			i++
			sinceArg = args[i]
		case strings.HasPrefix(a, "--since="):
			sinceArg = strings.TrimPrefix(a, "--since=")
		case a == "--help", a == "-h":
			fmt.Print(costUsage)
			return 0
		default:
			fmt.Fprintf(os.Stderr, "pi-run: cost: unknown flag or argument %q\n\n%s", a, costUsage)
			return 2
		}
	}

	root := repoRoot()
	if reset {
		archive, err := costReset(root)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pi-run: cost: --reset failed: %v\n", err)
			return 1
		}
		if archive != "" {
			fmt.Printf("archived spend ledger to %s\n", archive)
		}
		fmt.Println("cost tracking reset: budget now counts only sessions since the reset marker")
		return 0
	}

	since, err := parseSinceDate(sinceArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: cost: %v\n\n%s", err, costUsage)
		return 2
	}

	report, err := collectCostReport(root, since)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: cost: %v\n", err)
		return 1
	}
	if jsonOut {
		b, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(string(b))
		return 0
	}
	printCostTable(os.Stdout, report)
	return 0
}

// printCostTable renders the human-readable provider/model table.
func printCostTable(w io.Writer, report costReport) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PROVIDER\tMODEL\tTOKENS\tCOST (USD)\tSESSIONS")
	for _, row := range report.Rows {
		fmt.Fprintf(tw, "%s\t%s\t%d\t$%.6f\t%d\n", row.Provider, row.Model, row.TotalTokens, row.CostUSD, row.Sessions)
	}
	fmt.Fprintf(tw, "TOTAL\t\t%d\t$%.6f\t%d\n", report.TotalTokens, report.TotalCostUSD, report.Sessions)
	tw.Flush()
}

// costReset archives the current spend ledger and starts a fresh budget
// period: the ledger is renamed to .pi/cost-ledger-<ts>.archive.jsonl and a
// reset marker records the timestamp. Session files are never deleted. Returns
// the archive path (empty if there was no ledger to archive).
func costReset(root string) (string, error) {
	now := time.Now().UTC()
	dir := filepath.Join(root, ".pi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	archive := ""
	if _, err := os.Stat(ledgerPath(root)); err == nil {
		archive = filepath.Join(dir, fmt.Sprintf("cost-ledger-%s.archive.jsonl", now.Format("20060102T150405Z")))
		if err := os.Rename(ledgerPath(root), archive); err != nil {
			return "", err
		}
	}
	if err := os.WriteFile(resetMarkerPath(root), []byte(now.Format(time.RFC3339)+"\n"), 0o644); err != nil {
		return "", err
	}
	return archive, nil
}

// resetTime returns the timestamp of the last `pi-run cost --reset` (zero if
// never reset).
func resetTime(root string) time.Time {
	b, err := os.ReadFile(resetMarkerPath(root))
	if err != nil {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(b)))
	if err != nil {
		return time.Time{}
	}
	return t
}

// ledgerEntries returns all parseable ledger entries. Malformed lines are
// skipped (the ledger is best-effort audit data). A missing ledger yields no
// entries, not an error.
func ledgerEntries(root string) ([]ledgerEntry, error) {
	b, err := os.ReadFile(ledgerPath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []ledgerEntry
	for _, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var e ledgerEntry
		if json.Unmarshal([]byte(line), &e) != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// ledgerSum returns the total USD recorded in the spend ledger.
func ledgerSum(root string) (float64, error) {
	entries, err := ledgerEntries(root)
	if err != nil {
		return 0, err
	}
	var total float64
	for _, e := range entries {
		total += e.CostUSD
	}
	return total, nil
}

// ledgerAppend appends entries to the spend ledger (O_APPEND single-write
// lines), creating the file if needed. Entries with an empty TS get the
// current UTC time.
func ledgerAppend(root string, entries []ledgerEntry) error {
	path := ledgerPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range entries {
		if entries[i].TS == "" {
			entries[i].TS = now
		}
		b, err := json.Marshal(entries[i])
		if err != nil {
			return err
		}
		if _, err := f.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return f.Sync()
}

// ledgerEmpty reports whether the ledger has no entries yet.
func ledgerEmpty(root string) (bool, error) {
	entries, err := ledgerEntries(root)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// sessionSpendSince returns the total USD recorded in session files with mtime
// at/after since.
func sessionSpendSince(root string, since time.Time) (float64, error) {
	files, err := sessionFiles(root)
	if err != nil {
		return 0, err
	}
	var total float64
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil || info.ModTime().Before(since) {
			continue
		}
		samples, err := scanSessionFile(f)
		if err != nil {
			continue
		}
		for _, s := range samples {
			total += s.CostUSD
		}
	}
	return total, nil
}

// currentSpend returns cumulative real spend: the larger of the ledger total
// and the session-file total since the last reset. The ledger preserves spend
// after sessions are cleaned up; the session scan catches spend that was never
// logged (e.g. pi runs launched outside pi-run). Taking the larger avoids
// double-counting records that appear in both.
func currentSpend(root string) (float64, error) {
	ledger, err := ledgerSum(root)
	if err != nil {
		return 0, err
	}
	sessions, err := sessionSpendSince(root, resetTime(root))
	if err != nil {
		return 0, err
	}
	if sessions > ledger {
		return sessions, nil
	}
	return ledger, nil
}

// resolveBudgetCap returns the configured spend cap: --max-budget-usd wins
// over the PI_MAX_BUDGET_USD env var. Empty or zero means no cap.
func resolveBudgetCap(flagVal string) (float64, error) {
	v := flagVal
	if v == "" {
		v = os.Getenv("PI_MAX_BUDGET_USD")
	}
	if v == "" {
		return 0, nil
	}
	capUSD, err := strconv.ParseFloat(v, 64)
	if err != nil || capUSD < 0 {
		return 0, fmt.Errorf("invalid budget cap %q: --max-budget-usd / PI_MAX_BUDGET_USD must be a non-negative number", v)
	}
	return capUSD, nil
}

// recordRunSpend appends ledger entries for a just-finished run and returns
// the post-run cumulative spend. Entries are attributed from session files
// written during the run (mtime >= start), grouped by provider/model. If the
// run left no attributable session files, a single entry for the launch
// provider/model carries the remaining spend delta with tokens unknown (0).
// On the ledger's first write, existing session spend (mtime between any
// reset marker and run start) is backfilled first so the ledger stays a
// durable record after sessions are cleaned up.
func recordRunSpend(root string, start time.Time, mode, provider, model string, preSpend float64) (float64, error) {
	postSpend, err := currentSpend(root)
	if err != nil {
		return 0, err
	}
	delta := postSpend - preSpend
	if delta <= 0 {
		return postSpend, nil
	}

	backfilled := 0.0
	if empty, err := ledgerEmpty(root); err != nil {
		return 0, err
	} else if empty {
		backfilled, err = ledgerBackfill(root, start)
		if err != nil {
			return 0, err
		}
	}

	entries := sessionAttribution(root, start, mode)
	if len(entries) == 0 && delta-backfilled > 0 {
		entries = []ledgerEntry{{Provider: provider, Model: model, CostUSD: delta - backfilled, Mode: mode}}
	}
	return postSpend, ledgerAppend(root, entries)
}

// sessionAttribution groups usage from session files written during a run
// (mtime >= start) by provider/model, for ledger recording. Zero-cost groups
// are dropped.
func sessionAttribution(root string, start time.Time, mode string) []ledgerEntry {
	files, err := sessionFiles(root)
	if err != nil {
		return nil
	}
	agg := map[string]*ledgerEntry{}
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil || info.ModTime().Before(start) {
			continue
		}
		samples, err := scanSessionFile(f)
		if err != nil {
			continue
		}
		for _, s := range samples {
			key := s.Provider + "\x00" + s.Model
			e := agg[key]
			if e == nil {
				e = &ledgerEntry{Provider: s.Provider, Model: s.Model, Mode: mode}
				agg[key] = e
			}
			e.InputTokens += s.InputTokens
			e.OutputTokens += s.OutputTokens
			e.CostUSD += s.CostUSD
		}
	}
	keys := make([]string, 0, len(agg))
	for k := range agg {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	entries := make([]ledgerEntry, 0, len(keys))
	for _, k := range keys {
		if e := agg[k]; e.CostUSD > 0 {
			entries = append(entries, *e)
		}
	}
	return entries
}

// ledgerBackfill records session spend that predates the ledger (sessions
// created before pi-run tracked spend, e.g. direct pi runs) as "backfill"
// entries, so the ledger remains a durable record after sessions are cleaned
// up. Only sessions with mtime between the reset marker (if any) and the run's
// start are included: pre-reset spend belongs to the archived period, and the
// run's own sessions are recorded separately via sessionAttribution. Returns
// the total USD backfilled.
func ledgerBackfill(root string, start time.Time) (float64, error) {
	since := resetTime(root)
	files, err := sessionFiles(root)
	if err != nil {
		return 0, err
	}
	agg := map[string]*ledgerEntry{}
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil || info.ModTime().Before(since) || !info.ModTime().Before(start) {
			continue
		}
		samples, err := scanSessionFile(f)
		if err != nil {
			continue
		}
		for _, s := range samples {
			key := s.Provider + "\x00" + s.Model
			e := agg[key]
			if e == nil {
				e = &ledgerEntry{
					TS:       info.ModTime().UTC().Format(time.RFC3339),
					Provider: s.Provider,
					Model:    s.Model,
					Mode:     "backfill",
				}
				agg[key] = e
			}
			e.InputTokens += s.InputTokens
			e.OutputTokens += s.OutputTokens
			e.CostUSD += s.CostUSD
		}
	}
	keys := make([]string, 0, len(agg))
	for k := range agg {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	entries := make([]ledgerEntry, 0, len(keys))
	var total float64
	for _, k := range keys {
		if e := agg[k]; e.CostUSD > 0 {
			entries = append(entries, *e)
			total += e.CostUSD
		}
	}
	if len(entries) == 0 {
		return 0, nil
	}
	return total, ledgerAppend(root, entries)
}
