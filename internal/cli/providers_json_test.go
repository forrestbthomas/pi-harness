package cli

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

const sampleProvidersJSON = `{
  "providers": [
    {"name": "openai", "keyEnv": "OPENAI_API_KEY", "piProvider": "openai", "defaultModel": "openai/gpt-5.6-terra"},
    {"name": "local", "keyEnv": "LOCAL_API_KEY", "piProvider": "openai", "defaultModel": "local/model", "baseURL": "http://localhost:11434/v1"}
  ]
}`

func TestProvidersFromJSON(t *testing.T) {
	ps, err := ProvidersFromJSON([]byte(sampleProvidersJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 {
		t.Fatalf("got %d providers, want 2", len(ps))
	}
	if ps[0].Name != "openai" || ps[0].KeyEnv != "OPENAI_API_KEY" {
		t.Fatalf("unexpected provider: %+v", ps[0])
	}
	if ps[1].BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("local provider should carry baseURL, got %+v", ps[1])
	}
}

func TestProvidersFromJSONInvalid(t *testing.T) {
	if _, err := ProvidersFromJSON([]byte(`{"providers": [`)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadProvidersFromRepoFile(t *testing.T) {
	// Load the real providers.json from the repo root. Derive the repo root
	// from this source file (internal/cli/providers_json_test.go) so the test
	// is hermetic even when the test binary runs from a temp dir.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	ps, err := LoadProviders(filepath.Join(root, "providers.json"))
	if err != nil {
		t.Fatalf("LoadProviders: %v", err)
	}
	if len(ps) < 3 {
		t.Fatalf("providers.json should define at least 3 providers, got %d", len(ps))
	}
}

func TestDefaultProvidersIncludeAllShippedProviders(t *testing.T) {
	want := map[string]bool{
		"openai": false, "openrouter": false, "deepseek": false,
		"anthropic": false, "gemini": false, "groq": false, "local": false,
	}
	for _, p := range defaultProviders {
		if _, ok := want[p.Name]; ok {
			want[p.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Fatalf("defaultProviders missing shipped provider %q", name)
		}
	}
	for _, p := range defaultProviders {
		if p.Name == "local" && p.BaseURL != "http://localhost:11434/v1" {
			t.Fatalf("local fallback baseURL = %q", p.BaseURL)
		}
	}
}

// TestDefaultProvidersMatchRepoProvidersJSON enforces that the embedded
// defaultProviders table and the repository providers.json stay in sync:
// providers.json is the file-based override source of truth, and the embedded
// table must cover the same catalog so released binaries route identically.
func TestDefaultProvidersMatchRepoProvidersJSON(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	ps, err := LoadProviders(filepath.Join(root, "providers.json"))
	if err != nil {
		t.Fatalf("LoadProviders: %v", err)
	}
	if len(ps) != len(defaultProviders) {
		t.Fatalf("providers.json has %d providers but embedded table has %d; keep them in sync", len(ps), len(defaultProviders))
	}
	jsonByName := make(map[string]Provider, len(ps))
	for _, p := range ps {
		jsonByName[p.Name] = p
	}
	for _, p := range defaultProviders {
		jp, ok := jsonByName[p.Name]
		if !ok {
			t.Fatalf("providers.json missing provider %q present in embedded table", p.Name)
		}
		if jp != p {
			t.Fatalf("providers.json entry %q (%+v) differs from embedded entry (%+v)", p.Name, jp, p)
		}
	}
}

func TestLoadProvidersMissingFileFallsBackToDefaults(t *testing.T) {
	ps, err := LoadProviders(filepath.Join(t.TempDir(), "missing-providers.json"))
	if err != nil {
		t.Fatalf("LoadProviders missing file: %v", err)
	}
	if len(ps) != len(defaultProviders) {
		t.Fatalf("got %d fallback providers, want %d", len(ps), len(defaultProviders))
	}
}

func TestLoadProvidersMissingExplicitOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "missing-providers.json")
	if _, err := loadProviders(override, true); err == nil {
		t.Fatal("loadProviders must return an error for a missing explicit override")
	}
}

func TestProviderConfigPathUsesEnvOverride(t *testing.T) {
	override := filepath.Join(t.TempDir(), "custom-providers.json")
	if err := os.WriteFile(override, []byte(sampleProvidersJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PI_RUN_PROVIDERS_FILE", override)
	path, explicit := providerConfigPath("/unused-root")
	if path != override || !explicit {
		t.Fatalf("providerConfigPath = (%q, %t), want (%q, true)", path, explicit, override)
	}
	ps, err := loadProviders(path, explicit)
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 || ps[1].Name != "local" {
		t.Fatalf("override table was not loaded: %+v", ps)
	}
}

func captureProvidersStderr(t *testing.T, fn func() []Provider) ([]Provider, string) {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Stderr = old
		_ = w.Close()
		_ = r.Close()
	})
	os.Stderr = w
	providers := fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stderr = old
	var output strings.Builder
	if _, err := io.Copy(&output, r); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return providers, output.String()
}

func TestLoadActiveProvidersExplicitMissingWarns(t *testing.T) {
	override := filepath.Join(t.TempDir(), "missing-providers.json")
	t.Setenv("PI_RUN_PROVIDERS_FILE", override)
	providers, output := captureProvidersStderr(t, func() []Provider {
		return loadActiveProviders(t.TempDir())
	})
	if !reflect.DeepEqual(providers, defaultProviders) {
		t.Fatalf("fallback providers = %v, want defaultProviders", providers)
	}
	for _, want := range []string{"warning", override, "read explicit providers file"} {
		if !strings.Contains(output, want) {
			t.Fatalf("warning %q must contain %q", output, want)
		}
	}
}

func TestLoadActiveProvidersMalformedExplicitWarns(t *testing.T) {
	override := filepath.Join(t.TempDir(), "malformed-providers.json")
	if err := os.WriteFile(override, []byte(`{"providers": [`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PI_RUN_PROVIDERS_FILE", override)
	providers, output := captureProvidersStderr(t, func() []Provider {
		return loadActiveProviders(t.TempDir())
	})
	if !reflect.DeepEqual(providers, defaultProviders) {
		t.Fatalf("fallback providers = %v, want defaultProviders", providers)
	}
	for _, want := range []string{"warning", override, "parse providers"} {
		if !strings.Contains(output, want) {
			t.Fatalf("warning %q must contain %q", output, want)
		}
	}
}

func TestLoadActiveProvidersMalformedDefaultWarns(t *testing.T) {
	t.Setenv("PI_RUN_PROVIDERS_FILE", "")
	root := t.TempDir()
	path := filepath.Join(root, "providers.json")
	if err := os.WriteFile(path, []byte(`{"providers": [`), 0o600); err != nil {
		t.Fatal(err)
	}
	providers, output := captureProvidersStderr(t, func() []Provider {
		return loadActiveProviders(root)
	})
	if !reflect.DeepEqual(providers, defaultProviders) {
		t.Fatalf("fallback providers = %v, want defaultProviders", providers)
	}
	for _, want := range []string{"warning", path, "parse providers"} {
		if !strings.Contains(output, want) {
			t.Fatalf("warning %q must contain %q", output, want)
		}
	}
}

func TestLoadActiveProvidersDefaultMissingSilent(t *testing.T) {
	t.Setenv("PI_RUN_PROVIDERS_FILE", "")
	providers, output := captureProvidersStderr(t, func() []Provider {
		return loadActiveProviders(t.TempDir())
	})
	if len(providers) != len(defaultProviders) {
		t.Fatalf("got %d providers, want %d defaults", len(providers), len(defaultProviders))
	}
	if output != "" {
		t.Fatalf("default missing provider file must be silent, got %q", output)
	}
}
