package cli

import (
	"reflect"
	"strings"
	"testing"
)

func TestLookupProviderKnown(t *testing.T) {
	for _, name := range []string{"openai", "openrouter", "deepseek", "azure", "ollama", "bedrock"} {
		if _, err := LookupProvider(name); err != nil {
			t.Fatalf("LookupProvider(%q): %v", name, err)
		}
	}
}

// TestExpandedCatalogIncludesNewProviders pins the exact shape of the 10
// providers added for provider breadth (P1).
func TestExpandedCatalogIncludesNewProviders(t *testing.T) {
	want := map[string]Provider{
		"azure":      {Name: "azure", KeyEnv: "AZURE_OPENAI_API_KEY", PiProvider: "openai", DefaultModel: "azure/gpt-5.6-terra", BaseURL: "https://<your-resource>.openai.azure.com/openai/v1"},
		"ollama":     {Name: "ollama", KeyEnv: "OLLAMA_API_KEY", PiProvider: "openai", DefaultModel: "ollama/llama3.1", BaseURL: "http://localhost:11434/v1"},
		"mistral":    {Name: "mistral", KeyEnv: "MISTRAL_API_KEY", PiProvider: "openai", DefaultModel: "mistral/mistral-large-latest", BaseURL: "https://api.mistral.ai/v1"},
		"cohere":     {Name: "cohere", KeyEnv: "COHERE_API_KEY", PiProvider: "openai", DefaultModel: "cohere/command-r-plus", BaseURL: "https://api.cohere.com/compatibility/v1"},
		"together":   {Name: "together", KeyEnv: "TOGETHER_API_KEY", PiProvider: "openai", DefaultModel: "together/llama-3.3-70b-instruct", BaseURL: "https://api.together.xyz/v1"},
		"perplexity": {Name: "perplexity", KeyEnv: "PERPLEXITY_API_KEY", PiProvider: "openai", DefaultModel: "perplexity/sonar-pro", BaseURL: "https://api.perplexity.ai"},
		"fireworks":  {Name: "fireworks", KeyEnv: "FIREWORKS_API_KEY", PiProvider: "openai", DefaultModel: "fireworks/llama-3.3-70b-instruct", BaseURL: "https://api.fireworks.ai/inference/v1"},
		"moonshot":   {Name: "moonshot", KeyEnv: "MOONSHOT_API_KEY", PiProvider: "openai", DefaultModel: "moonshot/kimi-k2", BaseURL: "https://api.moonshot.cn/v1"},
		"xai":        {Name: "xai", KeyEnv: "XAI_API_KEY", PiProvider: "openai", DefaultModel: "xai/grok-4", BaseURL: "https://api.x.ai/v1"},
		"bedrock":    {Name: "bedrock", KeyEnv: "BEDROCK_API_KEY", PiProvider: "anthropic", DefaultModel: "bedrock/claude-sonnet-4", BaseURL: "https://bedrock-runtime.<region>.amazonaws.com/anthropic/v1"},
	}
	byName := make(map[string]Provider, len(defaultProviders))
	for _, p := range defaultProviders {
		byName[p.Name] = p
	}
	for name, want := range want {
		got, ok := byName[name]
		if !ok {
			t.Fatalf("defaultProviders missing provider %q", name)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("provider %q = %+v, want %+v", name, got, want)
		}
	}
}

// TestProviderNamesAndKeyEnvsUnique guards the routing table against duplicate
// names (ambiguous --provider) and duplicate key env vars (ambiguous key
// resolution).
func TestProviderNamesAndKeyEnvsUnique(t *testing.T) {
	names := make(map[string]bool, len(defaultProviders))
	keyEnvs := make(map[string]string, len(defaultProviders)) // keyEnv -> provider name
	for _, p := range defaultProviders {
		if names[p.Name] {
			t.Fatalf("duplicate provider name %q", p.Name)
		}
		names[p.Name] = true
		if prev, ok := keyEnvs[p.KeyEnv]; ok {
			t.Fatalf("keyEnv %q shared by %q and %q", p.KeyEnv, prev, p.Name)
		}
		keyEnvs[p.KeyEnv] = p.Name
		if p.Name == "" || p.KeyEnv == "" || p.PiProvider == "" || p.DefaultModel == "" {
			t.Fatalf("provider %+v has empty required field", p)
		}
	}
}

// TestSupportedProviderKeyEnvsCoverCatalog makes every catalog keyEnv eligible
// for the eval no-key skip guard (mirrors eval/conftest.py's list).
func TestSupportedProviderKeyEnvsCoverCatalog(t *testing.T) {
	covered := make(map[string]bool, len(supportedProviderKeyEnvs))
	for _, k := range supportedProviderKeyEnvs {
		covered[k] = true
	}
	for _, p := range defaultProviders {
		if !covered[p.KeyEnv] {
			t.Fatalf("keyEnv %q (provider %q) missing from supportedProviderKeyEnvs", p.KeyEnv, p.Name)
		}
	}
}

func TestLookupProviderUnknown(t *testing.T) {
	if _, err := LookupProvider("nope"); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestResolveProviderDefaultsToOpenAI(t *testing.T) {
	t.Setenv("PI_PROVIDER", "")
	p, err := ResolveProvider("")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "openai" || p.DefaultModel != "openai/gpt-5.6-terra" {
		t.Fatalf("unexpected default provider: %+v", p)
	}
}

func TestResolveProviderEnvFallback(t *testing.T) {
	t.Setenv("PI_PROVIDER", "deepseek")
	p, err := ResolveProvider("")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "deepseek" || p.DefaultModel != "deepseek/deepseek-v4-flash" {
		t.Fatalf("unexpected provider: %+v", p)
	}
}

func TestResolveProviderFlagWins(t *testing.T) {
	t.Setenv("PI_PROVIDER", "deepseek")
	p, err := ResolveProvider("openrouter")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "openrouter" {
		t.Fatalf("flag should override env, got %q", p.Name)
	}
}

func TestResolveProviderUnknown(t *testing.T) {
	if _, err := ResolveProvider("nope"); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestAnyProviderKeyEnvIncludesLocal(t *testing.T) {
	for _, key := range supportedProviderKeyEnvs {
		t.Setenv(key, "")
	}
	t.Setenv("LOCAL_API_KEY", "testvalue")
	if !anyProviderKeyEnv() {
		t.Fatal("LOCAL_API_KEY must make provider key availability true")
	}
}

func TestResolveModelTier(t *testing.T) {
	openai, err := LookupProvider("openai")
	if err != nil {
		t.Fatal(err)
	}
	deepseek, err := LookupProvider("deepseek")
	if err != nil {
		t.Fatal(err)
	}
	gemini, err := LookupProvider("gemini")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		p       Provider
		tier    string
		want    string
		wantErr bool
		errText string
	}{
		{"empty defaults to default model", openai, "", "openai/gpt-5.6-terra", false, ""},
		{"balanced aliases default model", openai, "balanced", "openai/gpt-5.6-terra", false, ""},
		{"openai fast maps", openai, "fast", "openai/gpt-5.4-mini", false, ""},
		{"openai cheap maps", openai, "cheap", "openai/gpt-5-mini", false, ""},
		{"gemini cheap maps", gemini, "cheap", "gemini/gemini-2.5-flash-lite", false, ""},
		{"deepseek fast maps", deepseek, "fast", "deepseek/deepseek-v4-pro", false, ""},
		{"unknown tier errors listing valid tiers", openai, "turbo", "", true, "valid: fast, balanced, cheap"},
		{"known but unmapped errors listing provider tiers", deepseek, "cheap", "", true, "available: balanced, fast"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveModelTier(tc.p, tc.tier)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveModelTier(%q) = %q, want error", tc.tier, got)
				}
				if !strings.Contains(err.Error(), tc.errText) {
					t.Fatalf("error %q must contain %q", err, tc.errText)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveModelTier(%q): %v", tc.tier, err)
			}
			if got != tc.want {
				t.Fatalf("resolveModelTier(%q) = %q, want %q", tc.tier, got, tc.want)
			}
		})
	}
}

// TestResolveModelTierNeverFallsBack asserts the no-silent-fallback invariant
// structurally: for every shipped provider and every known tier, the result is
// either an error or exactly the requested map entry / DefaultModel — never a
// different tier's model.
func TestResolveModelTierNeverFallsBack(t *testing.T) {
	for _, p := range defaultProviders {
		for _, tier := range []string{"fast", "balanced", "cheap"} {
			got, err := resolveModelTier(p, tier)
			if tier == "balanced" {
				if err != nil || got != p.DefaultModel {
					t.Fatalf("provider %q tier balanced = %q (err %v); want DefaultModel %q", p.Name, got, err, p.DefaultModel)
				}
				continue
			}
			if want, ok := p.ModelTiers[tier]; ok {
				if err != nil || got != want {
					t.Fatalf("provider %q tier %q = %q (err %v); want mapped %q", p.Name, tier, got, err, want)
				}
			} else if err == nil {
				t.Fatalf("provider %q tier %q must error (unmapped), got model %q — silent fallback", p.Name, tier, got)
			}
		}
	}
}

func TestDefaultProvidersTierMapsValid(t *testing.T) {
	for _, p := range defaultProviders {
		for tier, model := range p.ModelTiers {
			if !knownModelTiers[tier] || tier == "balanced" {
				t.Fatalf("provider %q has invalid tier key %q", p.Name, tier)
			}
			if strings.TrimSpace(model) == "" {
				t.Fatalf("provider %q tier %q has empty model", p.Name, tier)
			}
		}
	}
	// The five v1 tiered providers must ship a tier map.
	for _, name := range []string{"openai", "openrouter", "deepseek", "anthropic", "gemini"} {
		var p Provider
		for _, q := range defaultProviders {
			if q.Name == name {
				p = q
			}
		}
		if len(p.ModelTiers) == 0 {
			t.Fatalf("provider %q must ship a modelTiers map", name)
		}
	}
}

func TestAvailableTiers(t *testing.T) {
	openai, _ := LookupProvider("openai")
	deepseek, _ := LookupProvider("deepseek")
	groq, _ := LookupProvider("groq")
	tests := []struct {
		name string
		p    Provider
		want []string
	}{
		{"openai sorted with balanced present", openai, []string{"balanced", "cheap", "fast"}},
		{"deepseek omits cheap", deepseek, []string{"balanced", "fast"}},
		{"groq without tier map has only balanced", groq, []string{"balanced"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := availableTiers(tc.p)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("availableTiers = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveLaunchModel(t *testing.T) {
	p, err := LookupProvider("openai")
	if err != nil {
		t.Fatal(err)
	}
	explicit := "anthropic/claude-sonnet-4"
	tests := []struct {
		name    string
		tier    string
		model   string
		want    string
		wantErr bool
	}{
		{"tier only", "fast", "", "openai/gpt-5.4-mini", false},
		{"model only", "", explicit, explicit, false},
		{"neither uses default", "", "", "openai/gpt-5.6-terra", false},
		{"flag conflict errors", "fast", explicit, "", true},
		{"unknown tier errors", "turbo", "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveLaunchModel(p, tc.tier, tc.model)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveLaunchModel(%q, %q) = %q, want error", tc.tier, tc.model, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveLaunchModel(%q, %q): %v", tc.tier, tc.model, err)
			}
			if got != tc.want {
				t.Fatalf("resolveLaunchModel(%q, %q) = %q, want %q", tc.tier, tc.model, got, tc.want)
			}
		})
	}
}
