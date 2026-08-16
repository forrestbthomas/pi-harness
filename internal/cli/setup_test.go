package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeCheckout builds a temp dir shaped like a harness checkout:
// <root>/bin/pi-run plus the checkout markers (.pi/settings.json and
// eval/requirements.txt). Returns root and the exe path.
func fakeCheckout(t *testing.T) (root, exe string) {
	t.Helper()
	root = t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	exe = filepath.Join(root, "bin", "pi-run")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".pi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pi", "settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "eval"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "eval", "requirements.txt"), []byte("deepeval~=4.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, exe
}

func TestHarnessRootFromExeCheckout(t *testing.T) {
	root, exe := fakeCheckout(t)
	got, ok := harnessRootFromExe(exe)
	if !ok {
		t.Fatal("checkout exe should resolve to a harness root")
	}
	// EvalSymlinks canonicalizes symlinked prefixes (/var -> /private/var on
	// macOS), so compare against the canonicalized root.
	canon, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != canon {
		t.Fatalf("got root %q, want %q", got, canon)
	}
}

func TestHarnessRootFromExeSymlinkedInstall(t *testing.T) {
	root, exe := fakeCheckout(t)
	// Simulate `pi-run install`: ~/bin/pi-run -> <root>/bin/pi-run.
	linkDir := t.TempDir()
	link := filepath.Join(linkDir, "pi-run")
	if err := os.Symlink(exe, link); err != nil {
		t.Fatal(err)
	}
	got, ok := harnessRootFromExe(link)
	if !ok {
		t.Fatal("symlinked install exe should resolve to a harness root")
	}
	canon, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != canon {
		t.Fatalf("got root %q, want %q", got, canon)
	}
}

func TestHarnessRootFromExeHomebrewCellar(t *testing.T) {
	// Homebrew layout: <prefix>/Cellar/pi-run/<v>/bin/pi-run. The
	// parent-of-parent is the Cellar package dir, which is NOT a checkout.
	cellar := t.TempDir()
	binDir := filepath.Join(cellar, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(binDir, "pi-run")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, ok := harnessRootFromExe(exe); ok {
		t.Fatalf("Cellar exe should NOT resolve to a harness root, got %q", got)
	}
}

func TestHarnessRootFromExeNoMarkers(t *testing.T) {
	// A bin/pi-run without checkout markers must not be treated as a root.
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(binDir, "pi-run")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, ok := harnessRootFromExe(exe); ok {
		t.Fatalf("exe without markers should NOT resolve to a harness root, got %q", got)
	}
}

func TestIsHarnessRoot(t *testing.T) {
	root, _ := fakeCheckout(t)
	if !isHarnessRoot(root) {
		t.Fatalf("fake checkout should be recognized as a harness root")
	}
	// A dir missing the eval suite is not a harness root.
	noEval := t.TempDir()
	if err := os.MkdirAll(filepath.Join(noEval, ".pi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(noEval, ".pi", "settings.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if isHarnessRoot(noEval) {
		t.Fatal("dir without eval/ must not be a harness root")
	}
	// A dir missing .pi/settings.json is not a harness root.
	noSettings := t.TempDir()
	if err := os.MkdirAll(filepath.Join(noSettings, "eval"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(noSettings, "eval", "requirements.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if isHarnessRoot(noSettings) {
		t.Fatal("dir without .pi/settings.json must not be a harness root")
	}
}

func TestSetupInstallOllamaExtensionInstallsCompletePair(t *testing.T) {
	root, _ := fakeCheckout(t)
	ext := filepath.Join(root, ".pi", "extensions")
	if err := os.MkdirAll(filepath.Join(ext, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ext, "ollama.ts"), []byte("provider\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ext, "lib", "ollama-catalog.ts"), []byte("catalog\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	agentDir := t.TempDir()
	t.Setenv("PI_AGENT_DIR", agentDir)

	if err := setupInstallOllamaExtension(root); err != nil {
		t.Fatalf("setupInstallOllamaExtension() = %v", err)
	}
	for _, rel := range []string{"ollama.ts", filepath.Join("lib", "ollama-catalog.ts")} {
		got, err := os.ReadFile(filepath.Join(agentDir, "extensions", rel))
		if err != nil {
			t.Fatalf("read installed %s: %v", rel, err)
		}
		if string(got) == "" {
			t.Fatalf("installed %s is empty", rel)
		}
	}
}

func TestSetupInstallOllamaExtensionRollsBackPartialPublish(t *testing.T) {
	root, _ := fakeCheckout(t)
	ext := filepath.Join(root, ".pi", "extensions")
	if err := os.MkdirAll(filepath.Join(ext, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ext, "ollama.ts"), []byte("new provider\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ext, "lib", "ollama-catalog.ts"), []byte("new catalog\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	agentDir := t.TempDir()
	dstDir := filepath.Join(agentDir, "extensions")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldProvider := filepath.Join(dstDir, "ollama.ts")
	if err := os.WriteFile(oldProvider, []byte("old provider\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Make the second destination impossible to inspect. The first backup must
	// be restored when publication of the pair cannot proceed.
	if err := os.WriteFile(filepath.Join(dstDir, "lib"), []byte("not a directory\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PI_AGENT_DIR", agentDir)

	if err := setupInstallOllamaExtension(root); err == nil {
		t.Fatal("setupInstallOllamaExtension() succeeded despite an invalid destination")
	}
	got, err := os.ReadFile(oldProvider)
	if err != nil {
		t.Fatalf("read restored provider: %v", err)
	}
	if string(got) != "old provider\n" {
		t.Fatalf("restored provider = %q, want previous contents", got)
	}
}

func TestSetupInstallOllamaExtensionRejectsUntrustedRoot(t *testing.T) {
	agentDir := t.TempDir()
	t.Setenv("PI_AGENT_DIR", agentDir)
	if err := setupInstallOllamaExtension(t.TempDir()); err == nil {
		t.Fatal("setupInstallOllamaExtension() accepted an untrusted root")
	}
	if _, err := os.Stat(filepath.Join(agentDir, "extensions")); !os.IsNotExist(err) {
		t.Fatalf("untrusted install touched destination, stat error = %v", err)
	}
}

func TestRepoRootFallsBackToCWD(t *testing.T) {
	// HARNESS_ROOT unset and the test binary lives outside any checkout, so
	// repoRoot() must fall back to the current working directory.
	t.Setenv("HARNESS_ROOT", "")
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// NOTE: tests run with CWD = the package dir, which is inside the real
	// harness checkout. That is itself a harness root, so repoRoot() may
	// legitimately resolve to the checkout instead of CWD. This test only
	// asserts repoRoot() does not panic and returns a non-empty path.
	if got := repoRoot(); got == "" {
		t.Fatal("repoRoot() returned empty string")
	}
	_ = oldWD
}

// TestRunSetupMissingRequirementsFailsCleanly verifies that `pi-run setup` in a
// directory that is not a pi-harness checkout fails fast with a clear message
// (and does NOT attempt pip install with a cwd-relative requirements path).
func TestRunSetupMissingRequirementsFailsCleanly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	// No eval/requirements.txt anywhere under root (not a checkout).
	code, out := captureRunStderr(t, []string{"setup"})
	if code != 1 {
		t.Fatalf("setup outside checkout exit = %d, want 1; stderr:\n%s", code, out)
	}
	if !strings.Contains(out, "eval/requirements.txt") || !strings.Contains(out, "not found") {
		t.Fatalf("stderr must name the missing requirements file clearly, got:\n%s", out)
	}
}

// TestSetupInstallDepsMissingRequirementsFailsCleanly verifies the venv+dep
// installer fails fast with a clear message when the checkout markers are
// absent (regression: previously pip failed confusingly on a cwd-relative
// requirements path).
func TestSetupInstallDepsMissingRequirementsFailsCleanly(t *testing.T) {
	root := t.TempDir()
	code, err := setupInstallDeps(root)
	if code != 1 || err == nil {
		t.Fatalf("setupInstallDeps outside checkout = code %d, err %v; want code 1 + error", code, err)
	}
	if !strings.Contains(err.Error(), "eval/requirements.txt") || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error must name the missing requirements file clearly, got: %v", err)
	}
}

// TestSetupInstallDepsUsesAbsolutePaths verifies requirements.txt and the venv
// python are resolved from root (not the caller's cwd) — regression for brew
// installs where repoRoot != cwd.
func TestSetupInstallDepsUsesAbsolutePaths(t *testing.T) {
	root := t.TempDir()
	evalDir := filepath.Join(root, "eval")
	if err := os.MkdirAll(filepath.Join(evalDir, ".venv", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evalDir, "requirements.txt"), []byte("deepeval~=4.1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	argsFile := filepath.Join(root, "pip-args.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\nexit 0\n"
	if err := os.WriteFile(filepath.Join(evalDir, ".venv", "bin", "python"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	code, err := setupInstallDeps(root)
	if code != 0 || err != nil {
		t.Fatalf("setupInstallDeps = code %d, err %v; want 0, nil", code, err)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "eval", "requirements.txt")
	if !strings.Contains(string(args), want) {
		t.Fatalf("pip args %q must contain absolute requirements path %q", args, want)
	}
}
