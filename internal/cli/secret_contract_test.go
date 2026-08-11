package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// backendContractCase is one row of the cross-language PI_SECRET_BACKEND
// identifier contract: the env value, the canonical backend both languages
// must select, and whether the value is a known backend at all.
type backendContractCase struct {
	name      string
	envValue  string
	wantCanon string
	wantKnown bool
}

// secretBackendContractCases lists every supported identifier/alias plus an
// unknown value. The Go switch (secret.go) and Python (secret_backend.py)
// must agree on each row.
var secretBackendContractCases = []backendContractCase{
	{name: "empty", envValue: "", wantCanon: "bitwarden", wantKnown: true},
	{name: "bitwarden", envValue: "bitwarden", wantCanon: "bitwarden", wantKnown: true},
	{name: "bw alias", envValue: "bw", wantCanon: "bitwarden", wantKnown: true},
	{name: "1password", envValue: "1password", wantCanon: "1password", wantKnown: true},
	{name: "op alias", envValue: "op", wantCanon: "1password", wantKnown: true},
	{name: "env-only", envValue: "env-only", wantCanon: "env-only", wantKnown: true},
	{name: "env alias", envValue: "env", wantCanon: "env-only", wantKnown: true},
	{name: "unknown", envValue: "vault", wantCanon: "", wantKnown: false},
}

// TestSecretBackendContract verifies the Go side of the identifier contract.
func TestSecretBackendContract(t *testing.T) {
	for _, tc := range secretBackendContractCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PI_SECRET_BACKEND", tc.envValue)
			be, err := newSecretBackend()
			if tc.wantKnown {
				if err != nil {
					t.Fatalf("known backend %q: unexpected error %v", tc.envValue, err)
				}
				if be.Name() != tc.wantCanon {
					t.Fatalf("known backend %q: got %q, want %q", tc.envValue, be.Name(), tc.wantCanon)
				}
			} else {
				if err == nil {
					t.Fatalf("unknown backend %q: expected error, got backend %q", tc.envValue, be.Name())
				}
			}
		})
	}
}

// TestSecretBackendContractMatchesPython shells out to eval/secret_backend.py
// (stdlib-only, no deepeval) and asserts the Python selection logic agrees
// with Go for every supported and unsupported backend identifier. This is the
// cross-language contract test for the PI_SECRET_BACKEND contract; it is
// skipped when python3 is unavailable so the Go CI job stays hermetic.
func TestSecretBackendContractMatchesPython(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping cross-language secret-backend contract")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// file = <root>/internal/cli/secret_contract_test.go
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	script := filepath.Join(root, "eval", "secret_backend.py")
	if _, err := os.Stat(script); err != nil {
		t.Skipf("eval/secret_backend.py not found: %v", err)
	}

	for _, tc := range secretBackendContractCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(python, script, "canonical", tc.envValue)
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("python3 secret_backend.py canonical %q: %v", tc.envValue, err)
			}
			got := strings.TrimSpace(string(out))
			if tc.wantKnown {
				if got != tc.wantCanon {
					t.Fatalf("python selected %q for %q, want %q", got, tc.envValue, tc.wantCanon)
				}
			} else if got != "unknown" {
				t.Fatalf("python selected %q for unknown %q, want %q", got, tc.envValue, "unknown")
			}
		})
	}
}

// TestSecretBackendPythonResolveParity verifies that env-var-first resolution
// and unknown-backend behavior agree between Go and Python for a concrete key.
// The Python probe reports availability via exit code (never prints the value),
// so no secret material crosses a process boundary.
func TestSecretBackendPythonResolveParity(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available; skipping cross-language secret-backend contract")
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	script := filepath.Join(root, "eval", "secret_backend.py")
	if _, err := os.Stat(script); err != nil {
		t.Skipf("eval/secret_backend.py not found: %v", err)
	}

	// pythonResolves runs the availability probe and reports whether the key
	// resolved (exit 0) or was unavailable (exit 1).
	pythonResolves := func() (bool, error) {
		cmd := exec.Command(python, script, "resolve-available", "OPENAI_API_KEY")
		if err := cmd.Run(); err != nil {
			if ee, ok := errors.AsType[*exec.ExitError](err); ok && ee.ExitCode() == 1 {
				return false, nil
			}
			return false, err
		}
		return true, nil
	}

	t.Run("env var wins in both", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "sk-env-value")

		// Go resolves the env value.
		got, err := resolveSecret("OPENAI_API_KEY")
		if err != nil || got != "sk-env-value" {
			t.Fatalf("Go resolveSecret = %q, err %v; want env value", got, err)
		}

		// Python reports the key as available.
		resolved, err := pythonResolves()
		if err != nil {
			t.Fatalf("python probe: %v", err)
		}
		if !resolved {
			t.Fatal("python reported env value unavailable")
		}
	})

	t.Run("unknown backend unavailable in both", func(t *testing.T) {
		t.Setenv("PI_SECRET_BACKEND", "vault")
		t.Setenv("OPENAI_API_KEY", "")
		t.Setenv("BW_GET", filepath.Join(t.TempDir(), "missing-bw-get")) // hermetic: no vault

		// Go: unknown backend must error.
		if _, err := newSecretBackend(); err == nil {
			t.Fatal("Go accepted unknown backend")
		}

		// Python: unknown backend must report the key unavailable.
		resolved, err := pythonResolves()
		if err != nil {
			t.Fatalf("python probe: %v", err)
		}
		if resolved {
			t.Fatal("python resolved with unknown backend; want unavailable")
		}
	})
}
