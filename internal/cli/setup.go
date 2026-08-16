package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// repoRoot returns the harness repo root: HARNESS_ROOT env, else the directory
// that actually contains a harness checkout. Two layouts are recognized:
//
//  1. A source checkout: the binary lives at <root>/bin/pi-run (possibly via
//     the ~/bin/pi-run install symlink), so the resolved executable's
//     parent-of-parent is <root>. This is only trusted when <root> carries the
//     checkout markers (.pi/settings.json + eval/requirements.txt).
//  2. A Homebrew-installed binary: it lives under
//     <prefix>/Cellar/pi-run/<v>/bin/pi-run, whose parent-of-parent is a
//     Cellar package dir with no harness markers. In that case (and whenever
//     the executable-derived root is not a checkout) fall back to the current
//     working directory, i.e. the project the user invoked pi-run from.
func repoRoot() string {
	if r := os.Getenv("HARNESS_ROOT"); r != "" {
		return r
	}
	if exe, err := os.Executable(); err == nil {
		if root, ok := harnessRootFromExe(exe); ok {
			return root
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

// harnessRootFromExe resolves the executable path (through symlinks) and, if
// the binary lives inside a harness checkout at <root>/bin/pi-run, returns
// <root>. It returns ok=false when the executable is not inside a checkout
// (e.g. a Homebrew-installed binary under <prefix>/Cellar/...) so the caller
// can fall back to the current working directory.
func harnessRootFromExe(exe string) (string, bool) {
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	root := filepath.Dir(filepath.Dir(exe))
	if isHarnessRoot(root) {
		return root, true
	}
	return "", false
}

// isHarnessRoot reports whether dir looks like a pi-harness checkout: it must
// contain the project settings and the eval suite that every checkout ships.
func isHarnessRoot(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, ".pi", "settings.json")); err != nil {
		return false
	}
	_, err := os.Stat(filepath.Join(dir, "eval", "requirements.txt"))
	return err == nil
}

// runCmd runs cmd with args in dir, streaming stdio; returns its exit code.
func runCmd(cmd string, args []string, dir string) (int, error) {
	c := exec.Command(cmd, args...)
	c.Dir = dir
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		if ee, ok := errors.AsType[*exec.ExitError](err); ok {
			return ee.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

// setupInstallDeps creates eval/.venv and installs eval/requirements.txt under
// root. It uses absolute paths so a Homebrew-installed binary (whose repoRoot
// may differ from the caller's cwd) never resolves the requirements file or the
// venv relative to cwd. Returns a process exit code (0 = ok) and an error for
// hard failures; callers print the error and return the code.
func setupInstallDeps(root string) (int, error) {
	venv := filepath.Join(root, "eval", ".venv")
	venvPython := filepath.Join(venv, "bin", "python")
	reqFile := filepath.Join(root, "eval", "requirements.txt")
	if _, err := os.Stat(reqFile); err != nil {
		return 1, fmt.Errorf("pi-run: setup: %s not found — run setup from the pi-harness checkout (current root: %s)", reqFile, root)
	}
	if _, err := os.Stat(venvPython); err != nil {
		fmt.Println("creating eval/.venv ...")
		// Absolute venv path: never create relative to the caller's cwd.
		if code, err := runCmd("python3", []string{"-m", "venv", venv}, root); err != nil || code != 0 {
			return code, err
		}
		if code, err := runCmd(venvPython, []string{"-m", "pip", "install", "--upgrade", "pip"}, root); err != nil || code != 0 {
			return code, err
		}
	}
	fmt.Println("installing eval/requirements.txt ...")
	if code, err := runCmd(venvPython, []string{"-m", "pip", "install", "-r", reqFile}, root); err != nil || code != 0 {
		if code != 0 {
			fmt.Fprintf(os.Stderr, "pi-run: setup: pip install failed (exit %d) — see output above\n", code)
		}
		return code, err
	}
	return 0, nil
}

// setupInstallOllamaExtension installs the Ollama provider extension into Pi's
// global agent dir (~/.pi/agent/extensions/, or $PI_AGENT_DIR/extensions) so
// `pi-run --provider ollama` works from ANY project, not just a harness
// checkout. Pi auto-discovers extensions from the project's .pi/extensions/
// (cwd) and the global agent dir; shipping the extension only in the harness
// repo made the provider "unknown" elsewhere (2026-08-16). Idempotent;
// best-effort (skips silently when the source files are absent, e.g. a
// Homebrew-installed binary invoked outside a checkout).
func setupInstallOllamaExtension(root string) {
	agentDir := os.Getenv("PI_AGENT_DIR")
	if agentDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		agentDir = filepath.Join(home, ".pi", "agent")
	}
	extDir := filepath.Join(root, ".pi", "extensions")
	dstDir := filepath.Join(agentDir, "extensions")
	files := []string{"ollama.ts", filepath.Join("lib", "ollama-catalog.ts")}
	installed := false
	for _, rel := range files {
		src := filepath.Join(extDir, rel)
		b, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		dst := filepath.Join(dstDir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			continue
		}
		if err := os.WriteFile(dst, b, 0o644); err != nil {
			continue
		}
		installed = true
	}
	if installed {
		fmt.Printf("installed Ollama provider extension -> %s\n", filepath.Join(dstDir, "ollama.ts"))
	}
}

// runSetup creates eval/.venv, installs Python deps, and refreshes model
// catalogs (fetches the deepseek catalog). Idempotent.
func runSetup() int {
	root := repoRoot()
	setupInstallOllamaExtension(root)
	if code, err := setupInstallDeps(root); err != nil || code != 0 {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return code
	}
	fmt.Println("refreshing model catalogs (pi update --models) ...")
	home, _ := os.UserHomeDir()
	nodeVersion, err := resolveNodeVersion(home)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: setup: %v\n", err)
		return 4
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
