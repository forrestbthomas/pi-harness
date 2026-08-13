package cli

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestExitWatchdogTerminated locks the exit-code contract: 9 = watchdog
// terminated (stall / process-group kill), distinct from generic 1.
func TestExitWatchdogTerminated(t *testing.T) {
	if exitWatchdogTerminated != 9 {
		t.Fatalf("exitWatchdogTerminated = %d, want 9", exitWatchdogTerminated)
	}
}

// TestStallTimeoutDefault proves the default silent window is 300s.
func TestStallTimeoutDefault(t *testing.T) {
	if got := stallTimeout(); got != 300*time.Second {
		t.Fatalf("stallTimeout() = %v, want 300s", got)
	}
}

// TestStallTimeoutEnvOverride proves PI_STALL_TIMEOUT_SECS wins, and that
// non-positive/garbage values fall back to the default instead of erroring.
func TestStallTimeoutEnvOverride(t *testing.T) {
	t.Setenv("PI_STALL_TIMEOUT_SECS", "7")
	if got := stallTimeout(); got != 7*time.Second {
		t.Fatalf("stallTimeout() = %v, want 7s", got)
	}
	t.Setenv("PI_STALL_TIMEOUT_SECS", "0")
	if got := stallTimeout(); got != 300*time.Second {
		t.Fatalf("stallTimeout(0) = %v, want default 300s", got)
	}
	t.Setenv("PI_STALL_TIMEOUT_SECS", "abc")
	if got := stallTimeout(); got != 300*time.Second {
		t.Fatalf("stallTimeout(abc) = %v, want default 300s", got)
	}
}

// TestWatchdogFiresOnSilence proves the watchdog closes stalled() when no
// bytes arrive within the window. The kill itself is the caller's job, so the
// test only checks the signal.
func TestWatchdogFiresOnSilence(t *testing.T) {
	pr, pw := io.Pipe()
	wd := newWatchdog(80*time.Millisecond, 10*time.Millisecond, 4242, pr, pw)
	wd.start()
	defer wd.stop()

	select {
	case <-wd.stalled():
		// expected: silent for 80ms → stalled
	case <-time.After(2 * time.Second):
		t.Fatal("watchdog did not fire on silence")
	}
}

// TestWatchdogResetsOnBytes proves a run emitting output is never flagged: the
// stall clock resets on every byte, so sustained activity outlives the window.
func TestWatchdogResetsOnBytes(t *testing.T) {
	pr, pw := io.Pipe()
	wd := newWatchdog(120*time.Millisecond, 10*time.Millisecond, 4242, pr, pw)
	wd.start()
	defer wd.stop()

	// Feed bytes every 40ms — well inside the 120ms window — for 400ms total.
	stopFeed := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(40 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopFeed:
				return
			case <-ticker.C:
				_, _ = io.WriteString(pw, "x")
			}
		}
	}()

	// While feeding: must NOT stall.
	select {
	case <-wd.stalled():
		t.Fatal("watchdog stalled while output was flowing")
	case <-time.After(400 * time.Millisecond):
		// expected
	}
	close(stopFeed)
	<-done

	// Evidence reflects the bytes we fed.
	ev := wd.evidence()
	if ev.BytesRead == 0 {
		t.Fatal("evidence.BytesRead = 0 despite bytes fed")
	}
	if ev.SilentSeconds < 0 {
		t.Fatalf("evidence.SilentSeconds = %v, want >= 0", ev.SilentSeconds)
	}

	// After the feeder stops, the window passes and the watchdog fires.
	select {
	case <-wd.stalled():
		// expected
	case <-time.After(2 * time.Second):
		t.Fatal("watchdog did not fire after feeding stopped")
	}
}

// TestWatchdogStopUnblocksPipe proves stop() closes the tee pipe so the
// reader goroutine never leaks a blocked Read after the child exits.
func TestWatchdogStopUnblocksPipe(t *testing.T) {
	pr, pw := io.Pipe()
	wd := newWatchdog(time.Hour, time.Millisecond, 4242, pr, pw)
	wd.start()
	wd.stop() // must not hang; closes pw → reader gets EOF
}

// TestWatchdogEvidenceZeroWithoutOutput proves a silent run reports honest
// evidence (no bytes, no last-output timestamp beyond start).
func TestWatchdogEvidenceZeroWithoutOutput(t *testing.T) {
	pr, pw := io.Pipe()
	wd := newWatchdog(time.Hour, time.Millisecond, 4242, pr, pw)
	wd.start()
	defer wd.stop()
	ev := wd.evidence()
	if ev.BytesRead != 0 {
		t.Fatalf("evidence.BytesRead = %d, want 0", ev.BytesRead)
	}
}

// TestStallTimeoutGatedOffByEnv proves PI_SELF_HEAL=1 enables observability
// and absence disables it (deterministic jobs must not emit events).
func TestSelfHealEnabled(t *testing.T) {
	t.Setenv("PI_SELF_HEAL", "1")
	if !selfHealEnabled() {
		t.Fatal("selfHealEnabled() = false with PI_SELF_HEAL=1")
	}
	os.Unsetenv("PI_SELF_HEAL")
	if selfHealEnabled() {
		t.Fatal("selfHealEnabled() = true without PI_SELF_HEAL")
	}
}

// TestGoalFromArgs proves the goal (print prompt) is recovered from pi's argv,
// skipping flags.
func TestGoalFromArgs(t *testing.T) {
	args := []string{"--provider", "openai", "--model", "openai/gpt-5.6-terra", "--offline", "-p", "--no-session", "fix the bug"}
	if got := goalFromArgs(args); got != "fix the bug" {
		t.Fatalf("goalFromArgs = %q, want %q", got, "fix the bug")
	}
	if got := goalFromArgs([]string{"--offline", "-p"}); got != "" {
		t.Fatalf("goalFromArgs(all flags) = %q, want empty", got)
	}
}

// TestSessionNotPersisted proves --no-session runs report the honest literal.
func TestSessionNotPersisted(t *testing.T) {
	if !sessionNotPersisted([]string{"-p", "--no-session", "hello"}) {
		t.Fatal("expected --no-session to mark the run as non-persisted")
	}
	if sessionNotPersisted([]string{"-p", "hello"}) {
		t.Fatal("run without --no-session must be treated as persisted")
	}
}

// TestResumeHandleLiteral proves a non-persisted run's resume handle is the
// exact literal from the spec, never a fabricated session reference.
func TestResumeHandleLiteral(t *testing.T) {
	dir := t.TempDir()
	got := resumeHandle(dir, []string{"-p", "--no-session", "hello"})
	if got != "none (session not persisted)" {
		t.Fatalf("resumeHandle = %q, want literal", got)
	}
}

// TestResumeHandlePersisted finds the newest session file as the handle.
func TestResumeHandlePersisted(t *testing.T) {
	dir := t.TempDir()
	sess := dir + "/.pi/sessions"
	if err := os.MkdirAll(sess, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"a.jsonl", "z.jsonl"} {
		if err := os.WriteFile(sess+"/"+n, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := resumeHandle(dir, []string{"-p", "hello"}); got != "z.jsonl" {
		t.Fatalf("resumeHandle = %q, want newest z.jsonl", got)
	}
}

// TestResumeHandlePersistedButNotWritten covers the kill-before-session-write
// case: honest wording, never a fabricated id.
func TestResumeHandlePersistedButNotWritten(t *testing.T) {
	dir := t.TempDir() // no .pi/sessions yet
	got := resumeHandle(dir, []string{"-p", "hello"})
	if got != "persisted (session file not yet written)" {
		t.Fatalf("resumeHandle = %q, want honest not-yet-written wording", got)
	}
}

// TestWriteEscalationPacket proves the packet lands at .pi/heal/<ts>-report.json
// with the full evidence shape and a stderr summary (no hardcoded paths).
func TestWriteEscalationPacket(t *testing.T) {
	dir := t.TempDir()
	report := buildEscalationReport(dir, []string{"-p", "--no-session", "fix it"}, "stall", exitWatchdogTerminated, nil, 0)

	// Capture stderr to assert the human summary appears.
	var stderr strings.Builder
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	err := writeEscalationPacket(dir, report)
	_ = w.Close()
	os.Stderr = oldStderr
	if err != nil {
		t.Fatalf("writeEscalationPacket: %v", err)
	}
	_, _ = io.Copy(&stderr, r)

	// Packet file exists with correct shape.
	entries, err := os.ReadDir(dir + "/.pi/heal")
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected 1 packet file, got %v (%v)", entries, err)
	}
	if !strings.Contains(entries[0].Name(), "-report.json") {
		t.Fatalf("unexpected packet name %q", entries[0].Name())
	}
	data, err := os.ReadFile(dir + "/.pi/heal/" + entries[0].Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"trigger": "stall"`,
		`"exitCode": 9`,
		`"goal": "fix it"`,
		`"resumeHandle": "none (session not persisted)"`,
		`"schemaVersion": 1`,
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("packet missing %s:\n%s", want, data)
		}
	}
	// stderr summary present.
	if !strings.Contains(stderr.String(), "watchdog: run terminated (stall") {
		t.Fatalf("stderr summary missing: %q", stderr.String())
	}
}

// TestExecPiDirTimeoutStallTerminates is the hermetic end-to-end proof: a fake
// pi that hangs silently is killed by the output-stall watchdog, the run exits
// 9 (watchdog terminated), the escalation packet is written, and the process
// group is reaped. Uses a temp HOME with a fake node + fake pi script — no
// real node, no keys, no hardcoded paths.
func TestExecPiDirTimeoutStallTerminates(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns a real hanging child; skipped in -short")
	}
	home := t.TempDir()
	binDir := home + "/.nvm/versions/node/v99.0.0/bin"
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Fake node binary (nodeBinDir only stats it) and fake pi that hangs
	// forever with zero output — the impostor-package hang signature.
	if err := os.WriteFile(binDir+"/node", []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	piScript := "#!/bin/sh\ntrap '' TERM INT\nsleep 1000\n"
	if err := os.WriteFile(binDir+"/pi", []byte(piScript), 0o755); err != nil {
		t.Fatal(err)
	}

	ws := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PI_STALL_TIMEOUT_SECS", "1")  // 1s silent window — fast test
	t.Setenv("PI_WATCHDOG_GRACE_SECS", "0") // immediate SIGKILL — fast test

	code, err := execPiDirTimeout("v99.0.0", []string{"-p", "--no-session", "fix the bug"}, nil, ws, 30*time.Second)
	if err == nil {
		t.Fatalf("expected watchdog error, got nil (code %d)", code)
	}
	if code != exitWatchdogTerminated {
		t.Fatalf("exit code = %d, want %d (watchdog terminated)", code, exitWatchdogTerminated)
	}
	if !strings.Contains(err.Error(), "stalled") {
		t.Fatalf("error = %q, want stall mention", err.Error())
	}

	// Escalation packet written under the workspace's .pi/heal/.
	entries, err := os.ReadDir(ws + "/.pi/heal")
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected 1 escalation packet, got %v (%v)", entries, err)
	}
	data, err := os.ReadFile(ws + "/.pi/heal/" + entries[0].Name())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"trigger": "stall"`,
		`"exitCode": 9`,
		`"resumeHandle": "none (session not persisted)"`,
		`"goal": "fix the bug"`,
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("packet missing %s:\n%s", want, data)
		}
	}
}

// TestExecPiDirTimeoutWallClockTerminates is the sibling proof for the
// wall-clock path: a fake pi that produces output but never exits is killed by
// the timeout, exit 9, packet trigger "timeout".
func TestExecPiDirTimeoutWallClockTerminates(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns a real hanging child; skipped in -short")
	}
	home := t.TempDir()
	binDir := home + "/.nvm/versions/node/v99.0.0/bin"
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binDir+"/node", []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Fake pi that prints a line every 200ms (so it never stalls) but never
	// exits — the wall clock must be the terminator, not the stall detector.
	piScript := "#!/bin/sh\ntrap '' TERM INT\nwhile true; do echo tick; sleep 0.2; done\n"
	if err := os.WriteFile(binDir+"/pi", []byte(piScript), 0o755); err != nil {
		t.Fatal(err)
	}

	ws := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PI_STALL_TIMEOUT_SECS", "300") // stall window far above the wall clock
	t.Setenv("PI_WATCHDOG_GRACE_SECS", "0")  // immediate SIGKILL — fast test

	code, err := execPiDirTimeout("v99.0.0", []string{"-p", "--no-session", "work forever"}, nil, ws, 1200*time.Millisecond)
	if err == nil {
		t.Fatalf("expected watchdog error, got nil (code %d)", code)
	}
	if code != exitWatchdogTerminated {
		t.Fatalf("exit code = %d, want %d", code, exitWatchdogTerminated)
	}
	entries, err := os.ReadDir(ws + "/.pi/heal")
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected 1 escalation packet, got %v (%v)", entries, err)
	}
	data, _ := os.ReadFile(ws + "/.pi/heal/" + entries[0].Name())
	if !strings.Contains(string(data), `"trigger": "timeout"`) {
		t.Fatalf("expected timeout trigger:\n%s", data)
	}
}

// TestExecPiDirTimeoutExitCodePassthrough proves a fake pi that exits on its
// own passes its exit code through unchanged (no watchdog involvement).
func TestExecPiDirTimeoutExitCodePassthrough(t *testing.T) {
	home := t.TempDir()
	binDir := home + "/.nvm/versions/node/v99.0.0/bin"
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binDir+"/node", []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binDir+"/pi", []byte("#!/bin/sh\necho done\nexit 7\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ws := t.TempDir()
	t.Setenv("HOME", home)
	code, err := execPiDirTimeout("v99.0.0", []string{"-p", "--no-session", "exit now"}, nil, ws, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 7 {
		t.Fatalf("exit code = %d, want passthrough 7", code)
	}
	if _, statErr := os.Stat(ws + "/.pi/heal"); !os.IsNotExist(statErr) {
		t.Fatalf("no escalation packet expected on clean exit, but .pi/heal exists")
	}
}

// TestWatchdogGraceDefault proves the SIGTERM→SIGKILL grace defaults to 10s.
func TestWatchdogGraceDefault(t *testing.T) {
	if got := watchdogGrace(); got != 10*time.Second {
		t.Fatalf("watchdogGrace() = %v, want 10s", got)
	}
	t.Setenv("PI_WATCHDOG_GRACE_SECS", "2")
	if got := watchdogGrace(); got != 2*time.Second {
		t.Fatalf("watchdogGrace(2) = %v, want 2s", got)
	}
	t.Setenv("PI_WATCHDOG_GRACE_SECS", "0")
	if got := watchdogGrace(); got != 0 {
		t.Fatalf("watchdogGrace(0) = %v, want 0 (immediate SIGKILL)", got)
	}
}

// TestExitCodesTableDocumentsWatchdog proves the documented exit-code table
// carries the new code 9 (watchdog terminated) so operators can grep CI for it.
func TestExitCodesTableDocumentsWatchdog(t *testing.T) {
	if !strings.Contains(exitCodesText, "9  watchdog terminated") {
		t.Fatalf("exitCodesText missing code 9 watchdog terminated:\n%s", exitCodesText)
	}
	if !strings.Contains(usage, "9 watchdog terminated") {
		t.Fatalf("usage missing code 9 watchdog terminated")
	}
}

// TestCollectSideEffectsInRealRepo proves the escalation packet's git ledger
// captures real uncommitted changes in a fixture repo (side-effect ledger).
func TestCollectSideEffectsInRealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=testvalue", "GIT_AUTHOR_EMAIL=t@testvalue", "GIT_COMMITTER_NAME=testvalue", "GIT_COMMITTER_EMAIL=t@testvalue")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	git("init", "-q", ".")
	git("config", "user.name", "testvalue")
	git("config", "user.email", "t@testvalue")
	if err := os.WriteFile(dir+"/f.txt", []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "f.txt")
	git("commit", "-qm", "init")
	if err := os.WriteFile(dir+"/f.txt", []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ledger := collectSideEffects(dir)
	if !strings.Contains(ledger.GitStatus, "f.txt") {
		t.Fatalf("git status ledger missing f.txt: %q", ledger.GitStatus)
	}
	if !strings.Contains(ledger.GitDiff, "f.txt") {
		t.Fatalf("git diff ledger missing f.txt: %q", ledger.GitDiff)
	}
}

// TestCollectPendingStateNoRebase proves a plain git repo with NO rebase in
// progress reports RebaseInProgress=false — git rev-parse --git-path prints
// the state path even when absent, so the existence check must be the gate.
func TestCollectPendingStateNoRebase(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=testvalue", "GIT_AUTHOR_EMAIL=t@testvalue",
		"GIT_COMMITTER_NAME=testvalue", "GIT_COMMITTER_EMAIL=t@testvalue",
		"GIT_EDITOR=true")
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	git("init", "-q", ".")
	git("config", "user.name", "testvalue")
	git("config", "user.email", "t@testvalue")
	if err := os.WriteFile(dir+"/f.txt", []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "f.txt")
	git("commit", "-qm", "base")

	ps := collectPendingState(dir)
	if ps.RebaseInProgress {
		t.Fatal("collectPendingState reported a rebase in a clean repo (false positive)")
	}
}

// TestCollectPendingStateRebase proves the escalation packet records an
// in-progress rebase (pending state) for the self-heal command to recover. A
// deliberate conflict guarantees the rebase stays in progress.
func TestCollectPendingStateRebase(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=testvalue", "GIT_AUTHOR_EMAIL=t@testvalue",
		"GIT_COMMITTER_NAME=testvalue", "GIT_COMMITTER_EMAIL=t@testvalue",
		"GIT_EDITOR=true")
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	git("init", "-q", ".")
	git("config", "user.name", "testvalue")
	git("config", "user.email", "t@testvalue")
	// base commit on main
	if err := os.WriteFile(dir+"/f.txt", []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "f.txt")
	git("commit", "-qm", "base")
	// feature branch changes f.txt, then main changes the same line -> conflict
	git("checkout", "-qb", "feature")
	if err := os.WriteFile(dir+"/f.txt", []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "f.txt")
	git("commit", "-qm", "feature change")
	git("checkout", "-q", "main")
	if err := os.WriteFile(dir+"/f.txt", []byte("mainline\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "f.txt")
	git("commit", "-qm", "main change")
	git("checkout", "-q", "feature")

	// Rebase feature onto main: f.txt conflicts -> rebase stays in progress.
	cmd := exec.Command("git", "rebase", "main")
	cmd.Dir = dir
	cmd.Env = env
	_ = cmd.Run() // expected non-zero (conflict); GIT_EDITOR=true + GIT_TERMINAL_PROMPT=0 keep it non-interactive

	ps := collectPendingState(dir)
	if !ps.RebaseInProgress {
		t.Fatal("collectPendingState did not detect in-progress rebase after conflict")
	}
}
