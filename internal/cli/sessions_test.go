package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeSessionFile writes a minimal v3 session transcript: a session header
// line plus event lines, and sets the file mtime to mtime.
func writeSessionFile(t *testing.T, dir, id string, mtime time.Time, events ...string) string {
	t.Helper()
	path := filepath.Join(dir, "2026-08-16T12-00-00-000Z_"+id+".jsonl")
	header := `{"type":"session","version":3,"id":"` + id + `","timestamp":"2026-08-16T12:00:00.000Z","cwd":"/tmp/` + id + `"}`
	content := strings.Join(append([]string{header}, events...), "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write session file: %v", err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	return path
}

// errMsgEvent returns a transcript message event whose assistant message has
// stopReason "error" and the given errorMessage.
func errMsgEvent(id, errMsg string) string {
	m := map[string]any{
		"type":      "message",
		"id":        id,
		"timestamp": "2026-08-16T12:00:00.000Z",
		"message": map[string]any{
			"role":         "assistant",
			"stopReason":   "error",
			"errorMessage": errMsg,
		},
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func TestListSessionsActiveRecent(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	writeSessionFile(t, dir, "active-1", now.Add(-1*time.Minute))
	writeSessionFile(t, dir, "recent-2", now.Add(-2*time.Hour))
	writeSessionFile(t, dir, "old-3", now.Add(-48*time.Hour))

	all, err := listSessions(dir)
	if err != nil {
		t.Fatalf("listSessions: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(all))
	}

	active := filterSessions(all, now.Add(-5*time.Minute))
	if len(active) != 1 || active[0].ID != "active-1" {
		t.Fatalf("expected 1 active session (active-1), got %+v", active)
	}

	recent := filterSessions(all, now.Add(-24*time.Hour))
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent sessions, got %d", len(recent))
	}
}

func TestListSessionsSkipsMalformed(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	// Malformed header line: not valid JSON.
	bad := filepath.Join(dir, "2026-08-16T12-00-00-000Z_bad.jsonl")
	_ = os.WriteFile(bad, []byte("not json\n"), 0o600)
	_ = os.Chtimes(bad, now, now)
	writeSessionFile(t, dir, "good", now)

	all, err := listSessions(dir)
	if err != nil {
		t.Fatalf("listSessions: %v", err)
	}
	if len(all) != 1 || all[0].ID != "good" {
		t.Fatalf("expected only the good session, got %+v", all)
	}
}

func TestClassifyConnectionError(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{"Connection error.", true},
		{"Network connection lost.", true},
		{"502: {\"type\":\"server_error\",\"message\":\"fetch failed\"}", true},
		{"socket hang up", true},
		{"reset before headers", true},
		{"the request timed out", true},
		{"401 {\"error\":{\"type\":\"authentication_error\",\"message\":\"The API Key appears to be invalid\"}}", false},
		{"429 too many requests", false},
		{"insufficient_quota", false},
		{"model_not_found", false},
	}
	for _, c := range cases {
		if got := classifyConnectionError(c.msg); got != c.want {
			t.Errorf("classifyConnectionError(%q) = %v, want %v", c.msg, got, c.want)
		}
	}
}

func TestScanConnectionFlapsBoundary(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	// 3 connection errors within the window -> exactly 1 flap.
	sess := writeSessionFile(t, dir, "flappy", now,
		errMsgEvent("a", "Connection error."),
		errMsgEvent("b", "Network connection lost."),
		errMsgEvent("c", "fetch failed"),
	)
	flaps := scanSessionConnectionFlaps(sess, 3, 10*time.Minute, now.Add(time.Hour))
	if len(flaps) != 1 {
		t.Fatalf("expected 1 flap for 3 errors, got %d", len(flaps))
	}
	if flaps[0].Kind != "connection-flap" || flaps[0].SessionID != "flappy" || flaps[0].Count != 3 {
		t.Fatalf("unexpected flap: %+v", flaps[0])
	}
}

func TestScanConnectionFlapsBelowThreshold(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	sess := writeSessionFile(t, dir, "quiet", now,
		errMsgEvent("a", "Connection error."),
		errMsgEvent("b", "Connection error."),
	)
	flaps := scanSessionConnectionFlaps(sess, 3, 10*time.Minute, now.Add(time.Hour))
	if len(flaps) != 0 {
		t.Fatalf("expected 0 flaps for 2 errors < threshold 3, got %d", len(flaps))
	}
}

func TestScanConnectionFlapsIgnoresNonTransport(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	// 3 non-transport errors (auth) must not count as connection flaps.
	sess := writeSessionFile(t, dir, "authy", now,
		errMsgEvent("a", "401 {\"error\":{\"type\":\"authentication_error\"}}"),
		errMsgEvent("b", "429 too many requests"),
		errMsgEvent("c", "insufficient_quota"),
	)
	flaps := scanSessionConnectionFlaps(sess, 3, 10*time.Minute, now.Add(time.Hour))
	if len(flaps) != 0 {
		t.Fatalf("expected 0 flaps for non-transport errors, got %d", len(flaps))
	}
}

func TestScanConnectionFlapsWindowBoundary(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	// 3 errors but spread > window: the window is 10m; errors are ~6m apart
	// so they span 12m -> not 3 in-window -> no flap at threshold 3.
	// Build events with explicit timestamps to verify windowing.
	ev := func(ts, msg string) string {
		m := map[string]any{
			"type":      "message",
			"id":        "x",
			"timestamp": ts,
			"message": map[string]any{
				"role":         "assistant",
				"stopReason":   "error",
				"errorMessage": msg,
			},
		}
		b, _ := json.Marshal(m)
		return string(b)
	}
	sess := writeSessionFile(t, dir, "spread", now,
		ev("2026-08-16T12:00:00.000Z", "Connection error."),
		ev("2026-08-16T12:06:00.000Z", "Connection error."),
		ev("2026-08-16T12:12:00.000Z", "Connection error."),
	)
	flaps := scanSessionConnectionFlaps(sess, 3, 10*time.Minute, now.Add(time.Hour))
	if len(flaps) != 0 {
		t.Fatalf("expected 0 flaps for 3 errors spread beyond window, got %d", len(flaps))
	}
}

func TestWriteFlapEventsSeamFormat(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	sess := writeSessionFile(t, dir, "flappy", now,
		errMsgEvent("a", "Connection error."),
		errMsgEvent("b", "Network connection lost."),
		errMsgEvent("c", "fetch failed"),
	)
	flaps := scanSessionConnectionFlaps(sess, 3, 10*time.Minute, now.Add(time.Hour))
	if len(flaps) != 1 {
		t.Fatalf("setup: expected 1 flap, got %d", len(flaps))
	}

	healDir := filepath.Join(dir, ".pi", "heal")
	if err := writeFlapEvents(healDir, flaps); err != nil {
		t.Fatalf("writeFlapEvents: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(healDir, "events.jsonl"))
	if err != nil {
		t.Fatalf("read events.jsonl: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 event line, got %d: %s", len(lines), b)
	}
	var ev map[string]string
	if err := json.Unmarshal([]byte(lines[0]), &ev); err != nil {
		t.Fatalf("event not valid JSON: %v", err)
	}
	if ev["kind"] != "connection-flap" {
		t.Fatalf("expected kind connection-flap, got %q", ev["kind"])
	}
	if ev["ts"] == "" || ev["detail"] == "" {
		t.Fatalf("event missing ts/detail: %+v", ev)
	}
}
