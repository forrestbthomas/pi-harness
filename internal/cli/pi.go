package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

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
func piArgs(p Provider, model, mode string, rest []string) []string {
	args := []string{"--provider", p.PiProvider, "--model", model}
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
	path := binDir + string(os.PathListSeparator) + os.Getenv("PATH")
	cmd := exec.Command("pi", args...)
	cmd.Env = append(os.Environ(), append(extraEnv, "PATH="+path)...)
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
