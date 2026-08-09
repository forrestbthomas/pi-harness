package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeBwGet writes an executable script that behaves like bw_get.
func fakeBwGet(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "bw_get")
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveSecretPrefersEnv(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-env-value")
	t.Setenv("BW_GET", fakeBwGet(t, "#!/bin/sh\nexit 3\n"))
	got, err := resolveSecret("OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sk-env-value" {
		t.Fatalf("got %q, want %q", got, "sk-env-value")
	}
}

func TestResolveSecretFallsBackToBwGet(t *testing.T) {
	t.Setenv("BW_GET", fakeBwGet(t, "#!/bin/sh\nprintf '%s\\n' 'sk-vault-value'\n"))
	got, err := resolveSecret("OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sk-vault-value" {
		t.Fatalf("got %q, want %q", got, "sk-vault-value")
	}
}

func TestResolveSecretFailsWhenBwGetFails(t *testing.T) {
	t.Setenv("BW_GET", fakeBwGet(t, "#!/bin/sh\nexit 1\n"))
	if _, err := resolveSecret("OPENAI_API_KEY"); err == nil {
		t.Fatal("expected error when bw_get exits non-zero")
	}
}

func TestResolveSecretFallsBackToOnePassword(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "1password")
	// Hermetic: clear ambient env key and force bw_get to fail if old code
	// ever falls back to it (prevents a real vault key from leaking into test
	// output, per the never-log-secrets rule).
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("BW_GET", filepath.Join(t.TempDir(), "nonexistent-bw-get"))
	// fake op on PATH
	opBin := fakeScript(t, "op", "#!/bin/sh\nprintf '%s\\n' 'sk-op-value'\n")
	t.Setenv("PATH", filepath.Dir(opBin)+string(os.PathListSeparator)+os.Getenv("PATH"))
	got, err := resolveSecret("OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "sk-op-value" {
		t.Fatalf("got %q, want %q", got, "sk-op-value")
	}
}

func TestResolveSecretEnvOnlyNoFallback(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "env-only")
	// Hermetic: clear ambient env key. Use a bw_get fake that WOULD return a
	// value if the code fell back to it, proving env-only truly does not fall
	// back (and keeping the test independent of any real vault).
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("BW_GET", fakeBwGet(t, "#!/bin/sh\nprintf '%s\\n' 'sk-vault-value'\n"))
	if _, err := resolveSecret("OPENAI_API_KEY"); err == nil {
		t.Fatal("expected error when env var unset with env-only backend")
	}
}
