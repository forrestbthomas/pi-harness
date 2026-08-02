package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// repoRoot returns the harness repo root: HARNESS_ROOT env, else the parent of
// the resolved executable (binary lives at <root>/bin/pi-run).
func repoRoot() string {
	if r := os.Getenv("HARNESS_ROOT"); r != "" {
		return r
	}
	exe, err := os.Executable()
	if err != nil {
		return "."
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
	code, err := execPi(nodeVersion, []string{"update", "--models"}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return code
}
