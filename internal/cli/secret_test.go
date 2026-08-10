package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeScript writes an executable shell script and returns its path.
func fakeScript(t *testing.T, name, script string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSecretBackendSelectionBitwarden(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "")
	be, err := newSecretBackend()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if be.Name() != "bitwarden" {
		t.Fatalf("got %q, want %q", be.Name(), "bitwarden")
	}
}

func TestSecretBackendSelectionOnePassword(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "1password")
	be, err := newSecretBackend()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if be.Name() != "1password" {
		t.Fatalf("got %q, want %q", be.Name(), "1password")
	}
}

func TestSecretBackendSelectionEnvOnly(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "env-only")
	be, err := newSecretBackend()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if be.Name() != "env-only" {
		t.Fatalf("got %q, want %q", be.Name(), "env-only")
	}
}

func TestSecretBackendUnknown(t *testing.T) {
	t.Setenv("PI_SECRET_BACKEND", "vault")
	if _, err := newSecretBackend(); err == nil {
		t.Fatal("expected error for unknown backend")
	} else if !strings.Contains(err.Error(), "vault") {
		t.Fatalf("error should mention backend name, got: %v", err)
	}
}

func TestBitwardenResolveTimesOut(t *testing.T) {
	oldTimeout := secretCmdTimeout
	secretCmdTimeout = 20 * time.Millisecond
	t.Cleanup(func() { secretCmdTimeout = oldTimeout })
	t.Setenv("BW_GET", fakeScript(t, "bw_get", "#!/bin/sh\nwhile :; do :; done\n"))

	started := time.Now()
	_, err := (&bitwardenBackend{}).Resolve("OPENAI_API_KEY")
	if err == nil {
		t.Fatal("expected timed out secret lookup to fail")
	}
	if !strings.Contains(err.Error(), "bitwarden") || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("timeout error must identify only the backend and timeout, got: %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("secret lookup took %s, want bounded completion", elapsed)
	}
}

func TestOnePasswordResolveTimesOut(t *testing.T) {
	oldTimeout := secretCmdTimeout
	secretCmdTimeout = 20 * time.Millisecond
	t.Cleanup(func() { secretCmdTimeout = oldTimeout })
	opBin := fakeScript(t, "op", "#!/bin/sh\nwhile :; do :; done\n")
	t.Setenv("PATH", filepath.Dir(opBin)+string(os.PathListSeparator)+os.Getenv("PATH"))

	started := time.Now()
	_, err := (&onePasswordBackend{}).Resolve("OPENAI_API_KEY")
	if err == nil {
		t.Fatal("expected timed out secret lookup to fail")
	}
	if !strings.Contains(err.Error(), "1password") || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("timeout error must identify only the backend and timeout, got: %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("secret lookup took %s, want bounded completion", elapsed)
	}
}
