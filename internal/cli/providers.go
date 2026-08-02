package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Provider is a single routing entry.
type Provider struct {
	Name         string `json:"name"`              // CLI name: openai | openrouter | ...
	KeyEnv       string `json:"keyEnv"`            // env var / Bitwarden item holding the API key
	PiProvider   string `json:"piProvider"`        // value passed to `pi --provider`
	DefaultModel string `json:"defaultModel"`      // default `pi --model` value
	BaseURL      string `json:"baseURL,omitempty"` // optional provider base URL (e.g. local OpenAI-compatible)
}

// providerFile is the on-disk shape of providers.json.
type providerFile struct {
	Providers []Provider `json:"providers"`
}

// defaultProviders is the fallback routing table used when providers.json is
// missing (e.g. running tests without a repo root). openai is the default.
var defaultProviders = []Provider{
	{Name: "openai", KeyEnv: "OPENAI_API_KEY", PiProvider: "openai", DefaultModel: "openai/gpt-5.6-terra"},
	{Name: "openrouter", KeyEnv: "OPENROUTER_API_KEY", PiProvider: "openrouter", DefaultModel: "openai/gpt-5.6-terra"},
	{Name: "deepseek", KeyEnv: "DEEPSEEK_API_KEY", PiProvider: "deepseek", DefaultModel: "deepseek/deepseek-v4-flash"},
}

// Providers is the active routing table, loaded from providers.json when
// available, else the built-in defaults.
var Providers = defaultProviders

func init() {
	root := repoRoot()
	if root == "." {
		return // cannot locate providers.json; keep defaults
	}
	if ps, err := LoadProviders(filepath.Join(root, "providers.json")); err == nil && len(ps) > 0 {
		Providers = ps
	}
}

// ProvidersFromJSON parses provider table JSON.
func ProvidersFromJSON(data []byte) ([]Provider, error) {
	var f providerFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse providers: %w", err)
	}
	if len(f.Providers) == 0 {
		return nil, fmt.Errorf("parse providers: no providers defined")
	}
	return f.Providers, nil
}

// LoadProviders reads a provider table from path.
func LoadProviders(path string) ([]Provider, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ProvidersFromJSON(b)
}

// LookupProvider returns the provider named name, or an error.
func LookupProvider(name string) (Provider, error) {
	for _, p := range Providers {
		if p.Name == name {
			return p, nil
		}
	}
	return Provider{}, fmt.Errorf("unknown provider %q (want one of: %s)", name, providerNames())
}

// providerNames returns a comma-joined list of configured provider names.
func providerNames() string {
	names := make([]string, 0, len(Providers))
	for _, p := range Providers {
		names = append(names, p.Name)
	}
	return joinStrings(names, ", ")
}

func joinStrings(items []string, sep string) string {
	out := ""
	for i, s := range items {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

// ResolveProvider picks the provider from --provider, then PI_PROVIDER env,
// then the default (openai).
func ResolveProvider(flag string) (Provider, error) {
	name := flag
	if name == "" {
		name = os.Getenv("PI_PROVIDER")
	}
	if name == "" {
		name = "openai"
	}
	return LookupProvider(name)
}
