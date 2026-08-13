package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- fixture helpers (hermetic: isolated HOME, no system gitconfig) ---

// gitEnv returns an environment isolated from the user's real git config:
// GIT_CONFIG_NOSYSTEM=1 and HOME redirected to a throwaway dir.
func gitEnv(t *testing.T) []string {
	t.Helper()
	env := []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"HOME=" + t.TempDir(),
		"PATH=" + os.Getenv("PATH"),
	}
	return env
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not available on this runner: %v", err)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func gitInitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init", "-q", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
}

func commitAll(t *testing.T, dir, msg string) {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", msg)
}

// rebaseStateExists reports whether an in-progress rebase state dir exists
// (worktree-safe via git rev-parse --git-path).
func rebaseStateExists(t *testing.T, dir string) bool {
	t.Helper()
	for _, name := range []string{"rebase-merge", "rebase-apply"} {
		out := runGit(t, dir, "rev-parse", "--git-path", name)
		if out != "" {
			if _, err := os.Stat(filepath.Join(dir, out)); err == nil {
				return true
			}
		}
	}
	return false
}

// scaffoldPausedRebase creates a repo with an in-progress, conflict-free
// rebase: `git rebase -i --root` with the first todo action rewritten to
// "edit", which stops the rebase after applying the first commit with no
// conflicts. `git ls-files -u` is empty in this state.
func scaffoldPausedRebase(t *testing.T, dir string) {
	t.Helper()
	gitInitRepo(t, dir)
	writeFile(t, dir, "f.txt", "one\n")
	commitAll(t, dir, "c1")
	writeFile(t, dir, "f.txt", "two\n")
	commitAll(t, dir, "c2")

	// Sequence editor that rewrites the first "pick" to "edit" so rebase
	// stops after the first commit (paused, no conflicts).
	seq := filepath.Join(t.TempDir(), "seq.sh")
	script := "#!/bin/sh\nperl -pi -e 'if ($. == 1) { s/^pick/edit/ }' \"$1\"\n"
	if err := os.WriteFile(seq, []byte(script), 0o755); err != nil {
		t.Fatalf("write seq editor: %v", err)
	}

	cmd := exec.Command("git", "rebase", "-i", "--root")
	cmd.Dir = dir
	cmd.Env = append(gitEnv(t), "GIT_SEQUENCE_EDITOR="+seq, "GIT_EDITOR=true")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git rebase -i --root failed: %v\n%s", err, out)
	}
	if !rebaseStateExists(t, dir) {
		t.Fatal("expected in-progress rebase state after edit stop")
	}
	unmerged := runGit(t, dir, "ls-files", "-u")
	if unmerged != "" {
		t.Fatalf("expected no conflicts in paused rebase, got: %s", unmerged)
	}
}

// scaffoldConflictedRebase creates a repo with an in-progress rebase paused
// on a real conflict in f.txt.
func scaffoldConflictedRebase(t *testing.T, dir string) {
	t.Helper()
	gitInitRepo(t, dir)
	writeFile(t, dir, "f.txt", "base\n")
	commitAll(t, dir, "base")

	runGit(t, dir, "checkout", "-q", "-b", "side")
	writeFile(t, dir, "f.txt", "side\n")
	commitAll(t, dir, "side-change")

	runGit(t, dir, "checkout", "-q", "main")
	writeFile(t, dir, "f.txt", "main\n")
	commitAll(t, dir, "main-change")

	// Rebasing main onto side replays main-change onto side's f.txt -> conflict.
	out := runGitWithOutput(t, dir, "rebase", "side")
	if !strings.Contains(out, "CONFLICT") && !strings.Contains(out, "conflict") {
		t.Fatalf("expected a rebase conflict, got output:\n%s", out)
	}
	if !rebaseStateExists(t, dir) {
		t.Fatal("expected in-progress rebase state after conflict")
	}
	unmerged := runGit(t, dir, "ls-files", "-u")
	if !strings.Contains(unmerged, "f.txt") {
		t.Fatalf("expected f.txt unmerged, got:\n%s", unmerged)
	}
}

// runGitWithOutput is runGit but returns output on failure instead of
// failing the test (used where a non-zero exit is expected).
func runGitWithOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out))
	}
	return strings.TrimSpace(string(out))
}

// --- tests ---

func TestSelfHealNoGitState(t *testing.T) {
	dir := t.TempDir()
	gitInitRepo(t, dir)
	writeFile(t, dir, "f.txt", "hi\n")
	commitAll(t, dir, "c1")

	res := selfHealRepo(dir, false)
	if res.Status != healOK {
		t.Fatalf("status = %q, want %q (detail: %s)", res.Status, healOK, res.Detail)
	}
}

func TestSelfHealNonRepoDir(t *testing.T) {
	dir := t.TempDir() // no .git
	res := selfHealRepo(dir, false)
	if res.Status != healOK {
		t.Fatalf("status = %q, want %q (detail: %s)", res.Status, healOK, res.Detail)
	}
}

func TestSelfHealRecoversPausedRebase(t *testing.T) {
	dir := t.TempDir()
	scaffoldPausedRebase(t, dir)

	res := selfHealRepo(dir, false)
	if res.Status != healRecovered {
		t.Fatalf("status = %q, want %q (detail: %s)", res.Status, healRecovered, res.Detail)
	}
	if rebaseStateExists(t, dir) {
		t.Fatal("rebase state should be gone after recovery")
	}
	// The rebase must have completed: both commits are reachable.
	log := runGit(t, dir, "log", "--oneline")
	if got := strings.Count(log, "\n"); got < 1 {
		t.Fatalf("expected rebase to complete (>=2 commits), log:\n%s", log)
	}
}

func TestSelfHealNeedsAttentionOnConflict(t *testing.T) {
	dir := t.TempDir()
	scaffoldConflictedRebase(t, dir)

	res := selfHealRepo(dir, false)
	if res.Status != healNeedsAttention {
		t.Fatalf("status = %q, want %q (detail: %s)", res.Status, healNeedsAttention, res.Detail)
	}
	if len(res.Conflicts) != 1 || res.Conflicts[0] != "f.txt" {
		t.Fatalf("conflicts = %v, want [f.txt]", res.Conflicts)
	}
	// Git state must be untouched: rebase still in progress.
	if !rebaseStateExists(t, dir) {
		t.Fatal("rebase state must be left untouched on conflict")
	}
}

func TestSelfHealAbortExplicit(t *testing.T) {
	dir := t.TempDir()
	scaffoldConflictedRebase(t, dir)

	res := selfHealRepo(dir, true)
	if res.Status != healAborted {
		t.Fatalf("status = %q, want %q (detail: %s)", res.Status, healAborted, res.Detail)
	}
	if rebaseStateExists(t, dir) {
		t.Fatal("rebase state should be gone after --abort")
	}
}

func TestSelfHealAbortWithNothingToAbort(t *testing.T) {
	dir := t.TempDir()
	gitInitRepo(t, dir)
	writeFile(t, dir, "f.txt", "hi\n")
	commitAll(t, dir, "c1")

	res := selfHealRepo(dir, true)
	if res.Status != healOK {
		t.Fatalf("status = %q, want %q (detail: %s)", res.Status, healOK, res.Detail)
	}
}

func TestSelfHealRefusesWhenIndexLocked(t *testing.T) {
	dir := t.TempDir()
	scaffoldPausedRebase(t, dir)

	// Simulate a concurrent run: hold the index lock.
	gitDir := runGit(t, dir, "rev-parse", "--git-dir")
	if err := os.WriteFile(filepath.Join(dir, gitDir, "index.lock"), []byte(""), 0o644); err != nil {
		t.Fatalf("create index.lock: %v", err)
	}

	res := selfHealRepo(dir, false)
	if res.Status != healNeedsAttention {
		t.Fatalf("status = %q, want %q (detail: %s)", res.Status, healNeedsAttention, res.Detail)
	}
	if !strings.Contains(res.Detail, "index lock") {
		t.Fatalf("detail should mention the index lock, got: %s", res.Detail)
	}
	if !rebaseStateExists(t, dir) {
		t.Fatal("rebase must be left untouched while locked")
	}
}

// --- CLI surface tests ---

func TestRunSelfHealDispatch(t *testing.T) {
	// Running against the harness checkout (a clean git repo with no rebase)
	// must be "ok" with exit 0 and no side effects.
	if code := Run([]string{"self-heal"}); code != 0 {
		t.Fatalf("self-heal exit = %d, want 0", code)
	}
}

func TestRunSelfHealUnknownFlag(t *testing.T) {
	if code := Run([]string{"self-heal", "--frobnicate"}); code != 2 {
		t.Fatalf("unknown flag exit = %d, want 2", code)
	}
}

func TestSelfHealUsageMentionsAbort(t *testing.T) {
	if !strings.Contains(selfHealUsage, "--abort") {
		t.Fatal("self-heal usage must document --abort")
	}
	if !strings.Contains(selfHealUsage, "needs-attention") {
		t.Fatal("self-heal usage must document needs-attention status")
	}
}

func TestUsageMentionsSelfHealAndExitCode9(t *testing.T) {
	if !strings.Contains(usage, "self-heal") {
		t.Fatal("usage must list the self-heal command")
	}
	if !strings.Contains(usage, "9 watchdog terminated") {
		t.Fatal("usage exit-code line must include 9 watchdog terminated")
	}
	if !strings.Contains(exitCodesText, "9  watchdog terminated") {
		t.Fatal("exit-codes table must include code 9")
	}
}

func TestExitCodesCommandShows9(t *testing.T) {
	if !strings.Contains(exitCodesText, "9  watchdog terminated (stall/group-kill timeout)") {
		t.Fatal("--exit-codes output must document code 9 semantics")
	}
}
