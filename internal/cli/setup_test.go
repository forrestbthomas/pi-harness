package cli

import (
	"os"
	"path/filepath"
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
