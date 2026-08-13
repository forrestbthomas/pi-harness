package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackendStatusBitwarden(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "bitwarden")
	fake := fakeScript(t, "bw_get", "#!/bin/sh\nprintf '%s\\n' 'unlocked'\n")
	t.Setenv("BW_GET", fake)
	be, err := newSecretBackend()
	if err != nil {
		t.Fatal(err)
	}
	status, err := be.Status()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != "unlocked" {
		t.Fatalf("got %q, want %q", status, "unlocked")
	}
}

func TestBackendStatusOnePassword(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "1password")
	opBin := fakeScript(t, "op", "#!/bin/sh\nprintf '%s\\n' 'account list output'\n")
	t.Setenv("PATH", filepath.Dir(opBin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	be, err := newSecretBackend()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := be.Status(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoctorNoKeyExitsZero(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HARNESS_ROOT", root)
	t.Setenv("PI_RUN_PERSONAL", "")
	t.Setenv("PI_SECRET_BACKEND", "env-only")
	for _, key := range supportedProviderKeyEnvs {
		t.Setenv(key, "")
	}

	nodeBin := filepath.Join(home, ".nvm", "versions", "node", "v24.0.0", "bin")
	if err := os.MkdirAll(nodeBin, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"node", "pi"} {
		if err := os.WriteFile(filepath.Join(nodeBin, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	venv := filepath.Join(root, "eval", ".venv", "bin")
	if err := os.MkdirAll(venv, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(venv, "python"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	code := runDoctor()
	w.Close()
	os.Stdout = oldStdout
	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("runDoctor() = %d, want 0 with required prerequisites and no keys", code)
	}
	if strings.Contains(string(output), "FAILURES FOUND") {
		t.Fatalf("runDoctor output contains unexpected failure summary: %s", output)
	}
	if !strings.Contains(string(output), "non-interactive launch env present (hang prevention)") {
		t.Fatalf("runDoctor output missing the non-interactive env check: %s", output)
	}
}

// TestMissingNonInteractiveEnv locks the regression-guard helper: it reports
// exactly which hang-prevention vars are absent from a launch env.
func TestMissingNonInteractiveEnv(t *testing.T) {
	full := []string{"GIT_EDITOR=true", "GIT_SEQUENCE_EDITOR=true", "GIT_TERMINAL_PROMPT=0", "PAGER=cat"}
	if missing := missingNonInteractiveEnv(full); len(missing) != 0 {
		t.Fatalf("expected no missing vars, got %v", missing)
	}
	noPager := []string{"GIT_EDITOR=true", "GIT_SEQUENCE_EDITOR=true", "GIT_TERMINAL_PROMPT=0"}
	if missing := missingNonInteractiveEnv(noPager); len(missing) != 1 || missing[0] != "PAGER=cat" {
		t.Fatalf("expected [PAGER=cat] missing, got %v", missing)
	}
	if missing := missingNonInteractiveEnv(nil); len(missing) != 4 {
		t.Fatalf("expected all 4 missing for empty env, got %v", missing)
	}
}

func TestDoctorMissingNodeFails(t *testing.T) {
	home := t.TempDir()
	root := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HARNESS_ROOT", root)
	t.Setenv("PI_RUN_PERSONAL", "")
	t.Setenv("PI_SECRET_BACKEND", "env-only")
	for _, key := range supportedProviderKeyEnvs {
		t.Setenv(key, "")
	}
	venv := filepath.Join(root, "eval", ".venv", "bin")
	if err := os.MkdirAll(venv, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(venv, "python"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	if code := runDoctor(); code != 1 {
		t.Fatalf("runDoctor() = %d, want 1 when nvm node is missing", code)
	}
}
