package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
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
// pi runs with --offline so startup network ops (version check, changelog,
// catalog refresh) never hang on the flaky pi.dev endpoint; the stored model
// catalogs are used instead. `pi-run setup` is the explicit online path.
func piArgs(p Provider, model, mode string, rest []string, persist bool) []string {
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
	return append(args, rest...)
}

// launchEnv returns the provider key and any provider-specific environment
// needed by the Pi child. BaseURL is currently meaningful for OpenAI-compatible
// providers, including the built-in local provider.
func launchEnv(p Provider, key string) []string {
	env := []string{p.KeyEnv + "=" + key}
	if p.BaseURL != "" {
		env = append(env, "OPENAI_BASE_URL="+p.BaseURL)
	}
	return env
}

// execPi spawns `pi <args>` with the nvm node bin dir prepended to PATH and the
// given extra env (KEY_ENV=value pairs). Returns pi's exit code.
func execPi(nodeVersion string, args []string, extraEnv []string) (int, error) {
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
