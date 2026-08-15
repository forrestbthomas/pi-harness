package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Provider is a single routing entry.
type Provider struct {
	Name         string            `json:"name"`                 // CLI name: openai | openrouter | ...
	KeyEnv       string            `json:"keyEnv"`               // env var / Bitwarden item holding the API key
	PiProvider   string            `json:"piProvider"`           // value passed to `pi --provider`
	DefaultModel string            `json:"defaultModel"`         // default `pi --model` value
	BaseURL      string            `json:"baseURL,omitempty"`    // optional provider base URL (OpenAI- or Anthropic-compatible)
	Keyless      bool              `json:"keyless,omitempty"`    // local provider is available without an API credential
	ModelTiers   map[string]string `json:"modelTiers,omitempty"` // tier name -> model id (fast|cheap; balanced is the defaultModel alias and is never stored)
}

// providerFile is the on-disk shape of providers.json.
type providerFile struct {
	Providers []Provider `json:"providers"`
}

// defaultProviders is the complete routing table shipped in every binary.
// providers.json can replace it in a repository checkout or through
// PI_RUN_PROVIDERS_FILE. openai is the default.
//
// ⚠ VERIFY BEFORE RELEASE: every tier model below except deepseek/deepseek-v4-pro
// (which already appears as a valid --model value) is a placeholder string that
// must be confirmed against the pi model catalog (`pi-run setup` /
// `pi update --models`) before v0.8.0 ships. If a string is wrong, fix the tier
// entry — the no-fallback invariant is unchanged; only the table content is
// corrected. `balanced` is the reserved defaultModel alias and never appears in
// a tier map. Omitted tiers are intentional and surface as strict errors
// (rule (b)) rather than silent fallback.
var defaultProviders = []Provider{
	{Name: "openai", KeyEnv: "OPENAI_API_KEY", PiProvider: "openai", DefaultModel: "openai/gpt-5.6-terra",
		// fast/cheap verified against the pi model catalog (2026-08-12):
		// gpt-5.4-mini and gpt-5-mini are real OpenAI model ids (the earlier
		// gpt-5.6-mini / gpt-5.1-mini placeholders do not exist — 400
		// model_not_found). balanced is the defaultModel alias, never stored.
		ModelTiers: map[string]string{"fast": "openai/gpt-5.4-mini", "cheap": "openai/gpt-5-mini"}},
	{Name: "openrouter", KeyEnv: "OPENROUTER_API_KEY", PiProvider: "openrouter", DefaultModel: "openai/gpt-5.6-terra",
		// fast served via openrouter (anthropic/claude-haiku-4.5 is in the
		// openrouter catalog); cheap keeps the deepseek flash line.
		ModelTiers: map[string]string{"fast": "anthropic/claude-haiku-4.5", "cheap": "deepseek/deepseek-v4-flash"}},
	{Name: "deepseek", KeyEnv: "DEEPSEEK_API_KEY", PiProvider: "deepseek", DefaultModel: "deepseek/deepseek-v4-flash",
		ModelTiers: map[string]string{"fast": "deepseek/deepseek-v4-pro"}}, // real catalog value; cheap omitted (the -flash default is already the cheapest line)
	{Name: "anthropic", KeyEnv: "ANTHROPIC_API_KEY", PiProvider: "anthropic", DefaultModel: "anthropic/claude-sonnet-4",
		// claude-haiku-4.5 is the real haiku line (the earlier
		// anthropic/claude-haiku-4 does not exist); cheap omitted (haiku is the
		// single fast/cheap line).
		ModelTiers: map[string]string{"fast": "anthropic/claude-haiku-4.5"}},
	{Name: "gemini", KeyEnv: "GEMINI_API_KEY", PiProvider: "gemini", DefaultModel: "gemini/gemini-2.5-pro",
		// gemini-2.5-flash / -lite are real Gemini model ids (verified in the
		// openrouter catalog as google/gemini-2.5-flash*).
		ModelTiers: map[string]string{"fast": "gemini/gemini-2.5-flash", "cheap": "gemini/gemini-2.5-flash-lite"}},
	{Name: "groq", KeyEnv: "GROQ_API_KEY", PiProvider: "groq", DefaultModel: "groq/llama-3.3-70b-versatile"},
	{Name: "local", KeyEnv: "LOCAL_API_KEY", PiProvider: "openai", DefaultModel: "local/model", BaseURL: "http://localhost:11434/v1"},
	// OpenAI-compatible cloud + local endpoints routed through pi's openai provider.
	{Name: "azure", KeyEnv: "AZURE_OPENAI_API_KEY", PiProvider: "openai", DefaultModel: "azure/gpt-5.6-terra", BaseURL: "https://<your-resource>.openai.azure.com/openai/v1"},
	{Name: "ollama", KeyEnv: "OLLAMA_API_KEY", PiProvider: "ollama", DefaultModel: "ollama/llama3.1", BaseURL: "http://localhost:11434/v1", Keyless: true},
	{Name: "mistral", KeyEnv: "MISTRAL_API_KEY", PiProvider: "openai", DefaultModel: "mistral/mistral-large-latest", BaseURL: "https://api.mistral.ai/v1"},
	{Name: "cohere", KeyEnv: "COHERE_API_KEY", PiProvider: "openai", DefaultModel: "cohere/command-r-plus", BaseURL: "https://api.cohere.com/compatibility/v1"},
	{Name: "together", KeyEnv: "TOGETHER_API_KEY", PiProvider: "openai", DefaultModel: "together/llama-3.3-70b-instruct", BaseURL: "https://api.together.xyz/v1"},
	{Name: "perplexity", KeyEnv: "PERPLEXITY_API_KEY", PiProvider: "openai", DefaultModel: "perplexity/sonar-pro", BaseURL: "https://api.perplexity.ai"},
	{Name: "fireworks", KeyEnv: "FIREWORKS_API_KEY", PiProvider: "openai", DefaultModel: "fireworks/llama-3.3-70b-instruct", BaseURL: "https://api.fireworks.ai/inference/v1"},
	{Name: "moonshot", KeyEnv: "MOONSHOT_API_KEY", PiProvider: "openai", DefaultModel: "moonshot/kimi-k2", BaseURL: "https://api.moonshot.cn/v1"},
	{Name: "xai", KeyEnv: "XAI_API_KEY", PiProvider: "openai", DefaultModel: "xai/grok-4", BaseURL: "https://api.x.ai/v1"},
	// AWS Bedrock speaks the Anthropic messages format; pi's anthropic provider
	// receives the key and base URL via ANTHROPIC_API_KEY / ANTHROPIC_BASE_URL.
	{Name: "bedrock", KeyEnv: "BEDROCK_API_KEY", PiProvider: "anthropic", DefaultModel: "bedrock/claude-sonnet-4", BaseURL: "https://bedrock-runtime.<region>.amazonaws.com/anthropic/v1"},
}

// Providers is the active routing table, loaded from a configured JSON file
// when available, else the complete built-in defaults.
var Providers = defaultProviders

func init() {
	Providers = loadActiveProviders(repoRoot())
}

// loadActiveProviders loads the configured provider table. Explicit overrides
// and malformed tables warn loudly before falling back to built-in defaults;
// a missing default repository table remains silent for released binaries.
func loadActiveProviders(root string) []Provider {
	path, explicit := providerConfigPath(root)
	ps, err := loadProviders(path, explicit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: providers: warning: %v — using built-in provider defaults\n", err)
		return defaultProviders
	}
	if len(ps) == 0 {
		return defaultProviders
	}
	return ps
}

// providerConfigPath returns the provider table path and whether it was
// explicitly selected through PI_RUN_PROVIDERS_FILE.
func providerConfigPath(root string) (path string, explicit bool) {
	if path := os.Getenv("PI_RUN_PROVIDERS_FILE"); path != "" {
		return path, true
	}
	return filepath.Join(root, "providers.json"), false
}

// ProvidersFromJSON parses provider table JSON. Malformed modelTiers maps
// (type-level or semantic) fail the whole table so loadActiveProviders can
// warn and fall back to built-in defaults — never a partial table.
func ProvidersFromJSON(data []byte) ([]Provider, error) {
	var f providerFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse providers: %w", err)
	}
	if len(f.Providers) == 0 {
		return nil, fmt.Errorf("parse providers: no providers defined")
	}
	for _, p := range f.Providers {
		if err := validateTiers(p); err != nil {
			return nil, fmt.Errorf("parse providers: %w", err)
		}
	}
	return f.Providers, nil
}

// LoadProviders reads a repository provider table. A missing or unreadable
// repository table falls back to the complete provider set embedded in the binary.
func LoadProviders(path string) ([]Provider, error) {
	return loadProviders(path, false)
}

// loadProviders reads a provider table. Explicit overrides and malformed
// repository tables return errors so configuration mistakes are not silent;
// only a missing or unreadable default repository table falls back to defaults.
func loadProviders(path string, explicit bool) ([]Provider, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if explicit {
			return nil, fmt.Errorf("read explicit providers file %q: %w", path, err)
		}
		return defaultProviders, nil
	}
	ps, err := ProvidersFromJSON(b)
	if err != nil {
		return nil, fmt.Errorf("load providers file %q: %w", path, err)
	}
	return ps, nil
}

// providerRequiresCredential reports whether launch paths must resolve the
// provider's configured secret before starting Pi. Keyless is explicit so no
// provider name receives special-case behavior.
func providerRequiresCredential(p Provider) bool {
	return !p.Keyless
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
	var out strings.Builder
	for i, s := range items {
		if i > 0 {
			out.WriteString(sep)
		}
		out.WriteString(s)
	}
	return out.String()
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

// knownModelTiers is the closed set of tier names. balanced is the
// default-model alias and is never stored in a tier map.
var knownModelTiers = map[string]bool{"fast": true, "balanced": true, "cheap": true}

// validateTiers checks a provider's modelTiers map. Any violation fails the
// whole table (the caller warns and falls back to built-in defaults): unknown
// tier keys (a typo like "fasst" would otherwise be silently inert — the
// user's fast run would quietly run balanced), the reserved "balanced" key
// (it is an alias for defaultModel, storing it is a config bug), and empty
// values. There is no partial tolerance.
func validateTiers(p Provider) error {
	for tier, model := range p.ModelTiers {
		switch {
		case tier == "balanced":
			return fmt.Errorf("provider %q: reserved tier %q must not be stored in modelTiers (balanced is the defaultModel alias)", p.Name, tier)
		case !knownModelTiers[tier]:
			return fmt.Errorf("provider %q: unknown model tier %q (valid: fast, cheap)", p.Name, tier)
		case strings.TrimSpace(model) == "":
			return fmt.Errorf("provider %q: model tier %q has an empty model", p.Name, tier)
		}
	}
	return nil
}

// resolveModelTier returns the model id for a requested tier of provider p.
//
//	tier == ""         -> p.DefaultModel            (no tier requested)
//	tier == "balanced" -> p.DefaultModel            (reserved alias)
//	tier in p.ModelTiers -> that model id
//	tier known, not mapped -> error listing the tiers p DOES offer
//	tier unknown           -> error listing the valid tier names
//
// There is deliberately NO case that returns a different tier/model than the
// one requested: an unavailable tier is an error, never a fallback.
func resolveModelTier(p Provider, tier string) (string, error) {
	if tier == "" || tier == "balanced" {
		return p.DefaultModel, nil
	}
	if model, ok := p.ModelTiers[tier]; ok {
		return model, nil
	}
	if knownModelTiers[tier] {
		return "", fmt.Errorf("provider %q has no model for tier %q (available: %s)", p.Name, tier, joinStrings(availableTiers(p), ", "))
	}
	return "", fmt.Errorf("unknown model tier %q (valid: fast, balanced, cheap)", tier)
}

// availableTiers returns the sorted, deduped tier names a provider offers;
// "balanced" is always present (it is universally valid).
func availableTiers(p Provider) []string {
	tiers := []string{"balanced"}
	for t := range p.ModelTiers {
		if t == "" || t == "balanced" {
			continue
		}
		tiers = append(tiers, t)
	}
	sort.Strings(tiers)
	return tiers
}

// resolveLaunchModel is the runLaunch wiring: (c) flag+flag conflict, (a)
// unknown tier, (b) unmapped tier, and the existing --model / default-model
// behavior. Rules in order:
//  1. tier != "" && modelFlag != "" -> conflict error (exit 2, rule (c)).
//  2. tier != "" -> resolveModelTier(p, tier) (rules (a)/(b) surface here).
//  3. else modelFlag != "" -> modelFlag; else p.DefaultModel (today's
//     behavior). runLaunch arranges that the env tier is never passed here
//     alongside an explicit --model (rule (c')), so this branch only fires
//     for the flag+flag case or in direct tests.
func resolveLaunchModel(p Provider, tier, modelFlag string) (string, error) {
	if tier != "" && modelFlag != "" {
		return "", fmt.Errorf("--model-tier and --model are mutually exclusive; pick one")
	}
	if tier != "" {
		return resolveModelTier(p, tier)
	}
	if modelFlag != "" {
		return modelFlag, nil
	}
	return p.DefaultModel, nil
}
