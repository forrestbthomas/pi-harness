package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// childEnv builds the environment for spawned pi processes: the nvm node bin
// dir is prepended to PATH, NODE_OPTIONS forces IPv4-first DNS (the IPv6 route
// to pi.dev is broken on some networks and stalls fetch until timeouts), and
// extraEnv KEY_ENV=value pairs are appended. Any pre-existing NODE_OPTIONS is
// overridden, not duplicated.
func childEnv(binDir string, extraEnv []string) []string {
	env := make([]string, 0, len(os.Environ())+len(extraEnv)+2)
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "NODE_OPTIONS=") {
			continue // replaced below
		}
		env = append(env, kv)
	}
	env = append(env, "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	env = append(env, "NODE_OPTIONS=--dns-result-order=ipv4first")
	return append(env, extraEnv...)
}

// nodeBinDir returns the nvm node bin dir for the given version, verifying the
// node binary exists.
func nodeBinDir(home, version string) (string, error) {
	dir := filepath.Join(home, ".nvm", "versions", "node", version, "bin")
	if _, err := os.Stat(filepath.Join(dir, "node")); err != nil {
		return "", fmt.Errorf("node %s not found in %s (set PI_NODE_VERSION to override)", version, dir)
	}
	return dir, nil
}

// piArgs builds the argv for `pi` (minus the program name).
// mode: "chat", "print", or "resume". rest = pass-through flags and message positionals.
// persist is set when a budget cap is active: print runs then keep a session
// file so their spend can be recorded in the ledger (without a cap, print
// stays one-shot with --no-session).
// permissionMode is the harness-level permission tier (default|plan|
// acceptEdits|bypassPermissions). It is NOT a Pi flag: Pi has no
// --permission-mode. The tier is mapped to Pi's real tool-control surface:
//
//	plan             -> --tools read,grep,find,ls (read-only toolset)
//	acceptEdits      -> no extra flag (Pi's default already permits edits)
//	bypassPermissions -> --approve (trust project-local files)
//	default / empty  -> no flag
//
// pi runs with --offline so startup network ops (version check, changelog,
// catalog refresh) never hang on the flaky pi.dev endpoint; the stored model
// catalogs are used instead. `pi-run setup` is the explicit online path.
func piArgs(p Provider, model, mode string, rest []string, persist bool, permissionMode string) []string {
	args := []string{"--provider", p.PiProvider, "--model", model, "--offline"}
	switch mode {
	case "print":
		args = append(args, "-p")
		if !persist {
			args = append(args, "--no-session")
		}
	case "resume":
		args = append(args, "--continue")
	}
	switch permissionMode {
	case "plan":
		args = append(args, "--tools", "read,grep,find,ls")
	case "bypassPermissions":
		args = append(args, "--approve")
	case "", "default", "acceptEdits":
		// default: no flag; acceptEdits: Pi's default already permits edits.
	}
	return append(args, rest...)
}

// launchEnv returns the provider key, any provider-specific environment
// needed by the Pi child, and the non-interactive shell environment that
// prevents child bash tools from blocking on interactive prompts. BaseURL is
// meaningful for OpenAI-compatible providers (e.g. local, azure, mistral) via
// OPENAI_BASE_URL and for Anthropic-compatible providers (e.g. AWS Bedrock)
// via ANTHROPIC_BASE_URL; Anthropic-routed providers also receive the key
// under ANTHROPIC_API_KEY so pi's anthropic provider can authenticate when
// the entry's own keyEnv differs (e.g. BEDROCK_API_KEY).
func launchEnv(p Provider, key string) []string {
	env := []string{p.KeyEnv + "=" + key}
	switch p.PiProvider {
	case "anthropic":
		if p.KeyEnv != "ANTHROPIC_API_KEY" {
			env = append(env, "ANTHROPIC_API_KEY="+key)
		}
		if p.BaseURL != "" {
			env = append(env, "ANTHROPIC_BASE_URL="+p.BaseURL)
		}
	default:
		// OpenAI-compatible and any provider without a pi provider: base URL
		// travels via OPENAI_BASE_URL (legacy flat behavior).
		if p.BaseURL != "" {
			env = append(env, "OPENAI_BASE_URL="+p.BaseURL)
		}
	}
	// Non-interactive shell environment for every spawned pi process (and thus
	// every child bash tool it runs). Without these, commands like
	// `git rebase --continue` or `git commit` (no -m) open an interactive
	// editor in the agent's terminal and hang forever with no output — the
	// subagent never returns. GIT_EDITOR/SEQUENCE_EDITOR=true make git accept
	// the default message; GIT_TERMINAL_PROMPT=0 refuses credential prompts;
	// PAGER=cat prevents less/more pager hangs.
	env = append(env,
		"GIT_EDITOR=true",
		"GIT_SEQUENCE_EDITOR=true",
		"GIT_TERMINAL_PROMPT=0",
		"PAGER=cat",
	)
	return env
}

// execPi spawns `pi <args>` with the nvm node bin dir prepended to PATH and the
// given extra env (KEY_ENV=value pairs), in the current working directory.
func execPi(nodeVersion string, args []string, extraEnv []string) (int, error) {
	return execPiDir(nodeVersion, args, extraEnv, "")
}

// execPiDir is execPi with an explicit working directory for the child (used
// by the benchmark runner so the agent edits the task workspace). An empty dir
// runs in the caller's working directory.
func execPiDir(nodeVersion string, args []string, extraEnv []string, dir string) (int, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return 1, err
	}
	binDir, err := nodeBinDir(home, nodeVersion)
	if err != nil {
		return 4, err
	}
	// Use the absolute pi path: exec.Command resolves the binary against the
	// parent's PATH, not cmd.Env, so PATH alone is not enough.
	cmd := exec.Command(filepath.Join(binDir, "pi"), args...)
	cmd.Dir = dir
	cmd.Env = childEnv(binDir, extraEnv)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := errors.AsType[*exec.ExitError](err); ok {
			return ee.ExitCode(), nil // pi printed its own errors; pass the code through
		}
		return 1, err
	}
	return 0, nil
}

// execPiDirTimeout is execPiDir bounded by a wall-clock timeout. On expiry the
// child is killed (SIGKILL after the context deadline) so a hung pi process
// cannot block the caller forever — used by the benchmark runner for the agent
// step, which has no other bound. An empty dir runs in the caller's cwd.
func execPiDirTimeout(nodeVersion string, args []string, extraEnv []string, dir string, timeout time.Duration) (int, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return 1, err
	}
	binDir, err := nodeBinDir(home, nodeVersion)
	if err != nil {
		return 4, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(binDir, "pi"), args...)
	cmd.Dir = dir
	cmd.Env = childEnv(binDir, extraEnv)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return 1, fmt.Errorf("pi agent run timed out after %s", timeout)
		}
		if ee, ok := errors.AsType[*exec.ExitError](err); ok {
			return ee.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

// resolveNodeVersion returns the version of the latest Node installed via nvm
// (highest semver in ~/.nvm/versions/node/), or an error with a guided install
// hint when nvm is missing or no node is installed. PI_NODE_VERSION overrides.
func resolveNodeVersion(home string) (string, error) {
	if v := os.Getenv("PI_NODE_VERSION"); v != "" {
		return v, nil
	}
	dir := filepath.Join(home, ".nvm", "versions", "node")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("nvm not found: install it with `curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash`, or set PI_NODE_VERSION")
	}
	var versions []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if semverRe.MatchString(e.Name()) {
			versions = append(versions, e.Name())
		}
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("no Node installed via nvm: run `nvm install node`, or set PI_NODE_VERSION")
	}
	return maxSemver(versions), nil
}

// semverRe matches v<major>.<minor>.<patch> (e.g. v22.19.0).
var semverRe = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// maxSemver returns the highest semver string from a list of vX.Y.Z strings.
func maxSemver(versions []string) string {
	best := versions[0]
	for _, v := range versions[1:] {
		if semverGreater(v, best) {
			best = v
		}
	}
	return best
}

func semverGreater(a, b string) bool {
	pa, pb := strings.Split(strings.TrimPrefix(a, "v"), "."), strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := range 3 {
		na, _ := strconv.Atoi(pa[i])
		nb, _ := strconv.Atoi(pb[i])
		if na != nb {
			return na > nb
		}
	}
	return false
}
