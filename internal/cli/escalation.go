package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// escalationReport is the structured evidence packet written when a
// supervised run is terminated by the watchdog (output stall) or by the
// wall-clock timeout. It gives the operator everything needed to resume or
// reproduce without digging through a transcript (spec §"Escalation packet").
type escalationReport struct {
	SchemaVersion int    `json:"schemaVersion"`
	Timestamp     string `json:"timestamp"`
	Trigger       string `json:"trigger"` // "stall" | "timeout"
	Goal          string `json:"goal"`
	ExitCode      int    `json:"exitCode"`
	// ResumeHandle is the session id when the run persisted one (budget cap
	// active), else the literal "none (session not persisted)" — never a
	// fabricated session reference.
	ResumeHandle string           `json:"resumeHandle"`
	SideEffects  sideEffectLedger `json:"sideEffects"`
	PendingState pendingState     `json:"pendingState"`
	Evidence     stallEvidence    `json:"triggerEvidence"`
}

// sideEffectLedger is an append-only summary of what the run may have changed
// on disk (best-effort git status + diff stat), so a killed run's effects are
// visible without a full transcript.
type sideEffectLedger struct {
	GitStatus string `json:"gitStatus,omitempty"`
	GitDiff   string `json:"gitDiffStat,omitempty"`
}

// pendingState records durable in-progress state (e.g. a rebase) the next run
// must know about. Recovery of that state is the self-heal command's job; the
// watchdog only records it.
type pendingState struct {
	RebaseInProgress bool   `json:"rebaseInProgress"`
	Note             string `json:"note,omitempty"`
}

// writeEscalationPacket writes .pi/heal/<timestamp>-report.json under dir and
// prints a short human-readable summary to stderr. dir is the run's working
// directory (benchmark workspace or the launch cwd). Best-effort: a failure to
// write the report must never change the exit code.
func writeEscalationPacket(dir string, report escalationReport) error {
	healDir := filepath.Join(dir, ".pi", "heal")
	if err := os.MkdirAll(healDir, 0o755); err != nil {
		return err
	}
	name := fmt.Sprintf("%s-report.json", report.Timestamp)
	if _, err := os.Stat(filepath.Join(healDir, name)); err == nil {
		// Same-second collisions: append a suffix so we never overwrite
		// evidence (an append-only ledger).
		name = fmt.Sprintf("%s-%d-report.json", report.Timestamp, time.Now().UnixNano()%100000)
	}
	path := filepath.Join(healDir, name)
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return err
	}

	// Human-readable stderr summary (the evidence file path is the anchor).
	fmt.Fprintf(os.Stderr,
		"pi-run: watchdog: run terminated (%s, exit %d): %s\n"+
			"pi-run: watchdog: report: %s\n"+
			"pi-run: watchdog: resume: %s\n",
		report.Trigger, report.ExitCode, report.Goal,
		filepath.Join(healDir, name), report.ResumeHandle)
	return nil
}

// buildEscalationReport assembles the packet for a terminated supervised run.
// args is the pi argv (used to recover the goal and whether a session was
// persisted); wd may be nil for pure wall-clock timeouts without a watchdog.
func buildEscalationReport(dir string, args []string, trigger string, exitCode int, wd *watchdog, wallClock time.Duration) escalationReport {
	return escalationReport{
		SchemaVersion: 1,
		Timestamp:     time.Now().Format("2006-01-02T15-04-05"),
		Trigger:       trigger,
		Goal:          goalFromArgs(args),
		ExitCode:      exitCode,
		ResumeHandle:  resumeHandle(dir, args),
		SideEffects:   collectSideEffects(dir),
		PendingState:  collectPendingState(dir),
		Evidence:      evidenceFor(trigger, wd, wallClock),
	}
}

// evidenceFor builds the trigger-evidence block: the watchdog's observations
// for a stall, or the wall-clock bound for a timeout.
func evidenceFor(trigger string, wd *watchdog, wallClock time.Duration) stallEvidence {
	if wd != nil {
		return wd.evidence()
	}
	return stallEvidence{SilentSeconds: wallClock.Seconds()}
}

// goalFromArgs recovers the original goal (the print prompt) from pi's argv:
// the last positional argument (the message) after flags. Best-effort; a
// message that is entirely flags yields an empty goal rather than garbage.
func goalFromArgs(args []string) string {
	for i := len(args) - 1; i >= 0; i-- {
		a := args[i]
		if a == "" || strings.HasPrefix(a, "-") {
			continue
		}
		return a
	}
	return ""
}

// resumeHandle returns the session id when the run persisted one, else the
// literal "none (session not persisted)". Print runs persist a session only
// when a budget cap is active (piArgs omits --no-session then); the newest
// session file under dir/.pi/sessions is the handle. Best-effort: if a
// persisted run was killed before its session file was written, we say so
// rather than fabricate an id.
func resumeHandle(dir string, args []string) string {
	if sessionNotPersisted(args) {
		return "none (session not persisted)"
	}
	// A session was intended (budget cap active → no --no-session). Find the
	// newest session file for the resume handle.
	sessionsDir := filepath.Join(dir, ".pi", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return "persisted (session file not yet written)"
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "persisted (session file not yet written)"
	}
	sort.Strings(names)
	return names[len(names)-1]
}

// sessionNotPersisted reports whether pi was launched with --no-session (print
// runs without a budget cap), meaning there is no session to resume.
func sessionNotPersisted(args []string) bool {
	for _, a := range args {
		if a == "--no-session" {
			return true
		}
	}
	return false
}

// collectSideEffects snapshots git status (short) and diff --stat in dir,
// best-effort and bounded (5s each, context timeouts). A non-repo or gitless
// dir yields empty strings; the ledger is evidence, not a guarantee.
func collectSideEffects(dir string) sideEffectLedger {
	if dir == "" {
		dir = "."
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	status := boundedGit(ctx, dir, "status", "--short")
	diff := boundedGit(ctx, dir, "diff", "--stat")
	return sideEffectLedger{GitStatus: status, GitDiff: diff}
}

func boundedGit(ctx context.Context, dir string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "" // not a repo, no git, or timed out — leave the ledger empty
	}
	return strings.TrimSpace(string(out))
}

// collectPendingState records whether a rebase/merge is in progress in dir,
// via the worktree-safe git-path probes (self-heal owns the actual recovery;
// the watchdog only records state).
func collectPendingState(dir string) pendingState {
	if dir == "" {
		dir = "."
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// git rev-parse --git-path rebase-merge|rebase-apply resolves into the
	// worktree's git dir; a non-empty result means a rebase is in progress.
	for _, state := range []string{"rebase-merge", "rebase-apply"} {
		cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-path", state)
		cmd.Dir = dir
		if out, err := cmd.Output(); err == nil && strings.TrimSpace(string(out)) != "" {
			return pendingState{RebaseInProgress: true, Note: state}
		}
	}
	return pendingState{}
}

// logSelfHealEvent appends a JSON line to .pi/heal/events.jsonl under dir when
// --self-heal observability is enabled (PI_SELF_HEAL=1). Never fatal: event
// logging must not change exit codes.
func logSelfHealEvent(dir, kind, detail string) {
	if !selfHealEnabled() {
		return
	}
	healDir := filepath.Join(dir, ".pi", "heal")
	if err := os.MkdirAll(healDir, 0o755); err != nil {
		return
	}
	ev := map[string]string{
		"ts":     time.Now().Format(time.RFC3339),
		"kind":   kind,
		"detail": detail,
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(healDir, "events.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
}
