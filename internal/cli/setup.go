package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// repoRoot returns the harness repo root: HARNESS_ROOT env, else the parent of
// the resolved executable (binary lives at <root>/bin/pi-run). The executable
// path is resolved through symlinks so `~/bin/pi-run -> <root>/bin/pi-run`
// still resolves to <root>.
func repoRoot() string {
	if r := os.Getenv("HARNESS_ROOT"); r != "" {
		return r
	}
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Dir(filepath.Dir(exe))
}

// runCmd runs cmd with args in dir, streaming stdio; returns its exit code.
func runCmd(cmd string, args []string, dir string) (int, error) {
	c := exec.Command(cmd, args...)
	c.Dir = dir
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

// runSetup creates eval/.venv, installs Python deps, and refreshes model
// catalogs (fetches the deepseek catalog). Idempotent.
func runSetup() int {
	root := repoRoot()
	venv := filepath.Join(root, "eval", ".venv")
	venvPython := filepath.Join(venv, "bin", "python")
	if _, err := os.Stat(venvPython); err != nil {
		fmt.Println("creating eval/.venv ...")
		if code, err := runCmd("python3", []string{"-m", "venv", "eval/.venv"}, root); err != nil || code != 0 {
			return code
		}
		if code, err := runCmd(venvPython, []string{"-m", "pip", "install", "--upgrade", "pip"}, root); err != nil || code != 0 {
			return code
		}
	}
	fmt.Println("installing eval/requirements.txt ...")
	if code, err := runCmd(venvPython, []string{"-m", "pip", "install", "-r", "eval/requirements.txt"}, root); err != nil || code != 0 {
		return code
	}
	fmt.Println("refreshing model catalogs (pi update --models) ...")
	nodeVersion := os.Getenv("PI_NODE_VERSION")
	if nodeVersion == "" {
		nodeVersion = "v22.19.0"
	}
	// The pi.dev catalog endpoint is intermittently unreachable from some
	// networks (TLS connects, HTTP never responds), so the refresh can time
	// out even though the stored catalogs already resolve every default model.
	// Retry a few times; warn instead of failing setup on the final failure.
	const attempts = 3
	var refreshErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		code, err := execPi(nodeVersion, []string{"update", "--models"}, nil)
		if err == nil && code == 0 {
			refreshErr = nil
			break
		}
		if err != nil {
			refreshErr = err
		} else {
			refreshErr = fmt.Errorf("pi update --models exited with code %d", code)
		}
		if attempt < attempts {
			fmt.Printf("  model catalog refresh failed (attempt %d/%d); retrying ...\n", attempt, attempts)
			time.Sleep(3 * time.Second)
		}
	}
	if refreshErr != nil {
		fmt.Fprintf(os.Stderr,
			"warning: model catalog refresh failed after %d attempts (%v); stored catalogs still resolve the default models — run `pi-run doctor` to check\n",
			attempts, refreshErr)
	}
	return 0
}
