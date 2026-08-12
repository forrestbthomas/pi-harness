package cli

import "testing"

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
		if got != want {
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
