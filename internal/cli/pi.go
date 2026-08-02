package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
// mode: "chat" or "print". rest = pass-through flags and message positionals.
// pi runs with --offline so startup network ops (version check, changelog,
// catalog refresh) never hang on the flaky pi.dev endpoint; the stored model
// catalogs are used instead. `pi-run setup` is the explicit online path.
func piArgs(p Provider, model, mode string, rest []string) []string {
	args := []string{"--provider", p.PiProvider, "--model", model, "--offline"}
	if mode == "print" {
		args = append(args, "-p", "--no-session")
	}
	return append(args, rest...)
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
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), nil // pi printed its own errors; pass the code through
		}
		return 1, err
	}
	return 0, nil
}
