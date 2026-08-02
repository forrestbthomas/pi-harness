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
