package cli

import "testing"

func TestLookupProviderKnown(t *testing.T) {
	for _, name := range []string{"openai", "openrouter", "deepseek"} {
		if _, err := LookupProvider(name); err != nil {
			t.Fatalf("LookupProvider(%q): %v", name, err)
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
