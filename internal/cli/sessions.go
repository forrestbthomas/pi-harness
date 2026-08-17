package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// sessionsUsage documents the sessions command. It is printed for
// `pi-run sessions --help` and on usage errors.
const sessionsUsage = `Usage: pi-run sessions [--recent <duration>] [--active] [--json] [--heal]

Observe agent sessions from the existing .pi/sessions transcript seam.

  --recent <duration>   Only sessions whose file mtime is within <duration>
                        (default: 24h). Examples: 24h, 30m, 7d.
  --active              Only sessions whose file mtime is within the liveness
                        window (default: 5m).
  --json                Machine-readable JSON output (one object per session).
  --heal                Scan recent session transcripts for repeated
                        connection/transport failures and write aggregated
                        connection-flap events to .pi/heal/events.jsonl
                        (the same seam the scorecard reads as selfHeal).
  --help                Show this help.

Exit codes: 0 ok · 2 usage error
`

// sessionSummary is one row of the fleet view.
type sessionSummary struct {
	ID        string
	Timestamp string // from the session header line
	Cwd       string
	Name      string // from the last session_info event, when present
	Mtime     time.Time
	Age       time.Duration // now - mtime
	Active    bool          // mtime within the liveness window
}

// flapEvent is an aggregated connection-flap observation for one session.
type flapEvent struct {
	Kind      string
	SessionID string
	Count     int
	Detail    string
}

// defaultRecentWindow is the default --recent filter for the fleet view.
const defaultRecentWindow = 24 * time.Hour

// defaultActiveWindow is the mtime liveness window for --active.
const defaultActiveWindow = 5 * time.Minute

// defaultFlapThreshold is the minimum number of connection-class errors within
// the flap window that produces one connection-flap event. It mirrors Pi's
// built-in retry.maxRetries (3), so a flap is "Pi's own retry budget was
// exhausted repeatedly." Tunable via PI_HEAL_FLAP_THRESHOLD.
const defaultFlapThreshold = 3

// defaultFlapWindow is the time window over which connection-class errors are
// aggregated per session. Tunable via PI_HEAL_FLAP_WINDOW.
const defaultFlapWindow = 10 * time.Minute

// runSessions implements `pi-run sessions`.
func runSessions(args []string) int {
	recent := defaultRecentWindow
	active := false
	jsonOut := false
	heal := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			fmt.Print(sessionsUsage)
			return 0
		case "--active":
			active = true
		case "--json":
			jsonOut = true
		case "--heal":
			heal = true
		case "--recent":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "pi-run: sessions: --recent requires a duration (e.g. 24h)\n\n%s", sessionsUsage)
				return 2
			}
			d, err := time.ParseDuration(args[i+1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "pi-run: sessions: invalid --recent duration %q\n\n%s", args[i+1], sessionsUsage)
				return 2
			}
			recent = d
			i++
		default:
			fmt.Fprintf(os.Stderr, "pi-run: sessions: unknown flag %q\n\n%s", args[i], sessionsUsage)
			return 2
		}
	}

	root := repoRoot()
	sessionsDir := filepath.Join(root, ".pi", "sessions")
	now := time.Now()

	all, err := listSessions(sessionsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: sessions: %v\n", err)
		return 1
	}

	// Fleet view (default and --active/--recent/--json).
	var rows []sessionSummary
	for _, s := range all {
		if active {
			if now.Sub(s.Mtime) > defaultActiveWindow {
				continue
			}
			rows = append(rows, s)
			continue
		}
		if now.Sub(s.Mtime) > recent {
			continue
		}
		rows = append(rows, s)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Mtime.After(rows[j].Mtime) })

	if heal {
		// --heal scans the recent window's transcripts (independent of the
		// fleet filter) and writes connection-flap events through the heal
		// seam. The scan is an explicit user action, so it bypasses the
		// PI_SELF_HEAL env gate and writes regardless.
		threshold, window := flapSettings()
		flaps := scanSessionsForFlaps(all, threshold, window, now)
		healDir := filepath.Join(root, ".pi", "heal")
		if err := writeFlapEvents(healDir, flaps); err != nil {
			fmt.Fprintf(os.Stderr, "pi-run: sessions: --heal: %v\n", err)
			return 1
		}
		fmt.Printf("sessions --heal: wrote %d connection-flap event(s)\n", len(flaps))
		return 0
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		for _, s := range rows {
			if err := enc.Encode(s); err != nil {
				fmt.Fprintf(os.Stderr, "pi-run: sessions: %v\n", err)
				return 1
			}
		}
		return 0
	}

	if len(rows) == 0 {
		fmt.Println("no sessions")
		return 0
	}
	for _, s := range rows {
		fmt.Printf("%s\t%s\t%s\t%s\n", s.ID, s.Mtime.Format(time.RFC3339), s.Age.Round(time.Second).String(), s.Cwd)
	}
	return 0
}

// flapSettings returns the connection-flap threshold and window, honoring the
// PI_HEAL_FLAP_THRESHOLD / PI_HEAL_FLAP_WINDOW env overrides.
func flapSettings() (int, time.Duration) {
	threshold := defaultFlapThreshold
	if v := os.Getenv("PI_HEAL_FLAP_THRESHOLD"); v != "" {
		if n, err := parseIntEnv(v); err == nil && n > 0 {
			threshold = n
		}
	}
	window := defaultFlapWindow
	if v := os.Getenv("PI_HEAL_FLAP_WINDOW"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			window = d
		}
	}
	return threshold, window
}

func parseIntEnv(v string) (int, error) {
	var n int
	_, err := fmt.Sscanf(v, "%d", &n)
	return n, err
}

// listSessions reads every *.jsonl file in dir and returns a summary per
// session, parsing the leading session header line for id/timestamp/cwd and
// the last session_info event for a human name. Malformed/unreadable files are
// skipped, never fatal.
func listSessions(dir string) ([]sessionSummary, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []sessionSummary
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, err := e.Info()
		if err != nil {
			continue
		}
		s := sessionSummary{Mtime: info.ModTime(), Age: time.Since(info.ModTime())}
		parseSessionHeader(path, &s)
		// A transcript that does not yield a session id (malformed header) is
		// not a session we can list — skip it rather than show an empty row.
		if s.ID == "" {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

// parseSessionHeader reads the first line of the transcript (the session
// header) and, if it parses, fills ID/Timestamp/Cwd; it also scans for the
// last session_info event to pick up a human name. Best-effort: failures leave
// the summary unchanged.
func parseSessionHeader(path string, s *sessionSummary) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	first := true
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var ev struct {
			Type      string `json:"type"`
			ID        string `json:"id"`
			Timestamp string `json:"timestamp"`
			Cwd       string `json:"cwd"`
			Name      string `json:"name"`
		}
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if first && ev.Type == "session" {
			s.ID = ev.ID
			s.Timestamp = ev.Timestamp
			s.Cwd = ev.Cwd
			first = false
		}
		if ev.Type == "session_info" && ev.Name != "" {
			s.Name = ev.Name
		}
	}
}

// filterSessions returns sessions whose mtime is at or after cutoff.
func filterSessions(all []sessionSummary, cutoff time.Time) []sessionSummary {
	var out []sessionSummary
	for _, s := range all {
		if !s.Mtime.Before(cutoff) {
			out = append(out, s)
		}
	}
	return out
}

// connectionErrorPattern matches the transport-class failures Pi itself
// treats as retryable (mirrors isRetryableAssistantError's connection subset).
var connectionErrorPattern = regexp.MustCompile(`(?i)connection.?error|connection.?lost|network.?error|socket hang up|fetch failed|reset before headers|other side closed|upstream.?connect|timed? out|timeout`)

// classifyConnectionError reports whether an error message is a connection/
// transport failure (as opposed to auth/quota/model/HTTP 4xx-from-provider
// classes, which are deterministic and not connection flaps).
func classifyConnectionError(msg string) bool {
	// Explicitly exclude deterministic provider errors that happen to contain
	// a matched word (e.g. a 401 body mentioning "connection" is not a flap).
	if strings.Contains(msg, "authentication_error") || strings.Contains(msg, "insufficient_quota") || strings.Contains(msg, "rate_limit") || strings.Contains(msg, "too many requests") || strings.Contains(msg, "model_not_found") {
		return false
	}
	return connectionErrorPattern.MatchString(msg)
}

// scanSessionsForFlaps runs the aggregation over all sessions.
func scanSessionsForFlaps(all []sessionSummary, threshold int, window time.Duration, now time.Time) []flapEvent {
	var out []flapEvent
	for _, s := range all {
		if s.ID == "" {
			continue
		}
		out = append(out, scanSessionConnectionFlaps(filepath.Join(repoRoot(), ".pi", "sessions", s.ID+".jsonl"), threshold, window, now)...)
	}
	return out
}

// scanSessionConnectionFlaps scans one session transcript for connection-class
// error events and returns exactly one flapEvent when at least threshold
// connection errors occur within a sliding window of size window. The
// transcript filename is derived from the session id: sessions live in
// .pi/sessions/<ISO-timestamp>_<id>.jsonl, so we search for the file whose
// parsed id matches. To keep this pure and testable, path is the file to scan
// (callers resolve the id -> path mapping; scanSessionsForFlaps uses a direct
// glob fallback).
func scanSessionConnectionFlaps(path string, threshold int, window time.Duration, now time.Time) []flapEvent {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var errTimes []time.Time
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var ev struct {
			Type      string `json:"type"`
			Timestamp string `json:"timestamp"`
			Message   struct {
				StopReason   string `json:"stopReason"`
				ErrorMessage string `json:"errorMessage"`
			} `json:"message"`
		}
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if ev.Type != "message" || ev.Message.StopReason != "error" {
			continue
		}
		if !classifyConnectionError(ev.Message.ErrorMessage) {
			continue
		}
		if ts, err := time.Parse(time.RFC3339, ev.Timestamp); err == nil {
			errTimes = append(errTimes, ts)
		} else {
			// Unparsable timestamp: assume now for windowing purposes.
			errTimes = append(errTimes, now)
		}
	}

	for _, t0 := range errTimes {
		count := 0
		for _, t := range errTimes {
			if !t.Before(t0) && t.Sub(t0) <= window {
				count++
			}
		}
		if count >= threshold {
			return []flapEvent{{
				Kind:      "connection-flap",
				SessionID: sessionIDFromPath(path),
				Count:     count,
				Detail:    fmt.Sprintf("session %s: %d connection failures within %s", sessionIDFromPath(path), count, window),
			}}
		}
	}
	return nil
}

// sessionIDFromPath extracts the session id from a transcript filename of the
// form <ISO-timestamp>_<id>.jsonl: the id is everything after the first
// underscore up to .jsonl.
func sessionIDFromPath(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".jsonl")
	if i := strings.Index(base, "_"); i >= 0 {
		return base[i+1:]
	}
	return base
}

// writeFlapEvents appends flap events to <healDir>/events.jsonl in the same
// {"ts","kind","detail"} line format the scorecard's readSelfHealEvents
// consumes. It mirrors logSelfHealEvent's permissions (0700 dir, 0600 file).
func writeFlapEvents(healDir string, flaps []flapEvent) error {
	if len(flaps) == 0 {
		return nil
	}
	if err := os.MkdirAll(healDir, 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(healDir, "events.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, fl := range flaps {
		ev := map[string]string{
			"ts":     time.Now().Format(time.RFC3339),
			"kind":   fl.Kind,
			"detail": fl.Detail,
		}
		b, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return nil
}
