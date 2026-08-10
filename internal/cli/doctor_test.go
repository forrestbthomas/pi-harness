package cli

import (
	"os"
	"path/filepath"
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
