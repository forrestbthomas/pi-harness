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

// defaultProviders is the complete routing table shipped in every binary.
// providers.json can replace it in a repository checkout or through
// PI_RUN_PROVIDERS_FILE. openai is the default.
var defaultProviders = []Provider{
	{Name: "openai", KeyEnv: "OPENAI_API_KEY", PiProvider: "openai", DefaultModel: "openai/gpt-5.6-terra"},
	{Name: "openrouter", KeyEnv: "OPENROUTER_API_KEY", PiProvider: "openrouter", DefaultModel: "openai/gpt-5.6-terra"},
	{Name: "deepseek", KeyEnv: "DEEPSEEK_API_KEY", PiProvider: "deepseek", DefaultModel: "deepseek/deepseek-v4-flash"},
	{Name: "anthropic", KeyEnv: "ANTHROPIC_API_KEY", PiProvider: "anthropic", DefaultModel: "anthropic/claude-sonnet-4"},
	{Name: "gemini", KeyEnv: "GEMINI_API_KEY", PiProvider: "gemini", DefaultModel: "gemini/gemini-2.5-pro"},
	{Name: "groq", KeyEnv: "GROQ_API_KEY", PiProvider: "groq", DefaultModel: "groq/llama-3.3-70b-versatile"},
	{Name: "local", KeyEnv: "LOCAL_API_KEY", PiProvider: "openai", DefaultModel: "local/model", BaseURL: "http://localhost:11434/v1"},
}

// Providers is the active routing table, loaded from a configured JSON file
// when available, else the complete built-in defaults.
var Providers = defaultProviders

func init() {
	if ps, err := LoadProviders(providerConfigPath(repoRoot())); err == nil && len(ps) > 0 {
		Providers = ps
	}
}

// providerConfigPath returns the explicit provider table override when set,
// otherwise the providers.json file under root.
func providerConfigPath(root string) string {
	if path := os.Getenv("PI_RUN_PROVIDERS_FILE"); path != "" {
		return path
	}
	return filepath.Join(root, "providers.json")
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

// LoadProviders reads a provider table from path. A missing or unreadable
// table falls back to the complete provider set embedded in the binary.
func LoadProviders(path string) ([]Provider, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return defaultProviders, nil
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
