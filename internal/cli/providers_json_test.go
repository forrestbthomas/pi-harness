package cli

import (
	"os"
	"path/filepath"
	"runtime"
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
