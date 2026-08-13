package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// selfHealUsage documents the self-heal command. It is printed for
// `pi-run self-heal --help` and on usage errors.
const selfHealUsage = `Usage: pi-run self-heal [--abort]

Detect and recover in-progress git state (e.g. a wedged rebase) left behind
by an interrupted agent run.

  --abort   Explicitly abort an in-progress rebase. Destructive: discards
            the rebase work. Never automatic.

Reported statuses:
  ok               no git operation in progress (or not a git repository)
  recovered        an in-progress rebase with no unresolved conflicts was
                   continued via GIT_EDITOR=true git rebase --continue
  needs-attention  a rebase is in progress but cannot be safely continued
                   (unresolved conflicts, or another git operation holds the
                   index lock). Git state is left untouched — conflicts are
                   never guessed, and --abort is never automatic.
  aborted          the in-progress rebase was aborted via --abort

Exit codes: 0 ok/recovered/aborted · 1 needs-attention · 2 usage error
`

// healStatus is the machine-readable state-machine outcome of a self-heal run.
type healStatus string

const (
	healOK             healStatus = "ok"
	healRecovered      healStatus = "recovered"
	healNeedsAttention healStatus = "needs-attention"
	healAborted        healStatus = "aborted"
)

// healResult is the outcome of a self-heal attempt plus a human-readable
// detail and, for needs-attention, the unresolved conflict paths.
type healResult struct {
	Status    healStatus
	Detail    string
	Conflicts []string
}

// runSelfHeal implements `pi-run self-heal`.
func runSelfHeal(args []string) int {
	abort := false
	for _, a := range args {
		switch a {
		case "--abort":
			abort = true
		case "--help", "-h":
			fmt.Print(selfHealUsage)
			return 0
		default:
			fmt.Fprintf(os.Stderr, "pi-run: self-heal: unknown flag or argument %q\n\n%s", a, selfHealUsage)
			return 2
		}
	}

	res := selfHealRepo(".", abort)
	if res.Status == healNeedsAttention {
		fmt.Printf("== pi-run self-heal ==\nstatus: %s\ndetail: %s\n", res.Status, res.Detail)
		if len(res.Conflicts) > 0 {
			fmt.Println("conflicts:")
			for _, p := range res.Conflicts {
				fmt.Printf("  - %s\n", p)
			}
		}
		return 1
	}
	fmt.Printf("== pi-run self-heal ==\nstatus: %s\ndetail: %s\n", res.Status, res.Detail)
	return 0
}

// selfHealRepo runs the recovery state machine in dir (the repo root the
// user invoked pi-run from). It is pure and side-effect-bounded: the only
// writes are the git commands themselves (rebase --continue / --abort), and
// only when the state machine decides they are safe.
func selfHealRepo(dir string, abort bool) healResult {
	gitDir, err := gitRevParse(dir, "--git-dir")
	if err != nil {
		// Not a git repository (or git unavailable): nothing to heal.
		return healResult{Status: healOK, Detail: "no git repository detected; nothing to heal"}
	}

	stateDir := rebaseStateDir(dir, gitDir)
	if stateDir == "" {
		if abort {
			return healResult{Status: healOK, Detail: "no rebase in progress; --abort had nothing to abort"}
		}
		return healResult{Status: healOK, Detail: "no git operation in progress"}
	}

	if abort {
		if _, err := gitIn(dir, nil, "rebase", "--abort"); err != nil {
			return healResult{Status: healNeedsAttention, Detail: "rebase --abort failed: " + err.Error()}
		}
		return healResult{Status: healAborted, Detail: "in-progress rebase aborted (destructive: rebase work discarded)"}
	}

	// Idempotency guard: if another git operation holds the index lock, a
	// concurrent run is active — refuse to recover rather than race it.
	if gitIndexLocked(dir, gitDir) {
		return healResult{Status: healNeedsAttention,
			Detail: "another git operation holds the index lock; rebase left untouched (refusing to recover while another run is active)"}
	}

	unmerged, err := gitUnmergedPaths(dir)
	if err != nil {
		return healResult{Status: healNeedsAttention, Detail: "could not inspect conflict state: " + err.Error()}
	}
	if len(unmerged) > 0 {
		return healResult{Status: healNeedsAttention,
			Detail:    "rebase has unresolved conflicts; left untouched (conflicts are never guessed)",
			Conflicts: unmerged,
		}
	}

	if _, err := gitIn(dir, setEnvVar(os.Environ(), "GIT_EDITOR", "true"), "rebase", "--continue"); err != nil {
		return healResult{Status: healNeedsAttention, Detail: "rebase --continue failed: " + err.Error()}
	}
	return healResult{Status: healRecovered,
		Detail: "continued in-progress rebase via GIT_EDITOR=true git rebase --continue"}
}

// gitIn runs git in dir with the given extra env and returns trimmed combined
// output. The caller owns whether an error is fatal.
func gitIn(dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// gitRevParse returns the trimmed output of `git rev-parse <arg>` in dir.
func gitRevParse(dir, arg string) (string, error) {
	return gitIn(dir, os.Environ(), "rev-parse", arg)
}

// rebaseStateDir returns the absolute path of an in-progress rebase state
// directory (rebase-merge or rebase-apply) that exists on disk, or "" when no
// rebase is in progress. Uses `git rev-parse --git-path` so worktree repos
// resolve to their per-worktree git dirs.
func rebaseStateDir(dir, gitDir string) string {
	for _, name := range []string{"rebase-merge", "rebase-apply"} {
		out, err := gitIn(dir, os.Environ(), "rev-parse", "--git-path", name)
		if err != nil || out == "" {
			continue
		}
		p := resolveGitDir(dir, out)
		if pathExists(p) {
			return p
		}
	}
	// git rev-parse --git-path can emit the state dir even when absent; the
	// existence check above is authoritative. Fall back to a direct probe of
	// the classic locations for robustness across git versions.
	base := resolveGitDir(dir, gitDir)
	for _, name := range []string{"rebase-merge", "rebase-apply"} {
		if pathExists(filepath.Join(base, name)) {
			return filepath.Join(base, name)
		}
	}
	return ""
}

// resolveGitDir joins a possibly-relative git dir path onto dir; absolute
// paths pass through unchanged.
func resolveGitDir(dir, gitDir string) string {
	if filepath.IsAbs(gitDir) {
		return gitDir
	}
	return filepath.Join(dir, gitDir)
}

// gitIndexLocked reports whether the repo's index lock exists, which git holds
// during any write operation — the signal that another run may be active.
func gitIndexLocked(dir, gitDir string) bool {
	base := resolveGitDir(dir, gitDir)
	_, err := os.Stat(filepath.Join(base, "index.lock"))
	return err == nil
}

// gitUnmergedPaths returns the unique unmerged (conflicted) paths from
// `git ls-files -u`, or nil when the index has no conflicts.
func gitUnmergedPaths(dir string) ([]string, error) {
	out, err := gitIn(dir, os.Environ(), "ls-files", "-u")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var paths []string
	seen := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		if i := strings.LastIndex(line, "\t"); i >= 0 {
			p := line[i+1:]
			if !seen[p] {
				seen[p] = true
				paths = append(paths, p)
			}
		}
	}
	return paths, nil
}

// setEnvVar returns env with any existing key= removed and key=val appended.
// os/exec passes duplicate keys through as-is and libc behavior varies, so the
// caller must not rely on last-wins; this guarantees a single occurrence.
func setEnvVar(env []string, key, val string) []string {
	var out []string
	prefix := key + "="
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			out = append(out, e)
		}
	}
	return append(out, prefix+val)
}
