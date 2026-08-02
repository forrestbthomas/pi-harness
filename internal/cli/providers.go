package cli

import (
	"fmt"
	"os"
)

// Provider is a single routing entry.
type Provider struct {
	Name         string // CLI name: openai | openrouter | deepseek
	KeyEnv       string // env var / Bitwarden item holding the API key
	PiProvider   string // value passed to `pi --provider`
	DefaultModel string // default `pi --model` value
}

// Providers is the routing table. openai is the default.
var Providers = []Provider{
	{Name: "openai", KeyEnv: "OPENAI_API_KEY", PiProvider: "openai", DefaultModel: "openai/gpt-5.6-terra"},
	{Name: "openrouter", KeyEnv: "OPENROUTER_API_KEY", PiProvider: "openrouter", DefaultModel: "openai/gpt-5.6-terra"},
	{Name: "deepseek", KeyEnv: "DEEPSEEK_API_KEY", PiProvider: "deepseek", DefaultModel: "deepseek/deepseek-v4-flash"},
}

// LookupProvider returns the provider named name, or an error.
func LookupProvider(name string) (Provider, error) {
	for _, p := range Providers {
		if p.Name == name {
			return p, nil
		}
	}
	return Provider{}, fmt.Errorf("unknown provider %q (want openai, openrouter, or deepseek)", name)
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
