package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNodeBinDirOK(t *testing.T) {
	home := t.TempDir()
	nodeDir := filepath.Join(home, ".nvm", "versions", "node", "v22.19.0", "bin")
	if err := os.MkdirAll(nodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeDir, "node"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := nodeBinDir(home, "v22.19.0")
	if err != nil {
		t.Fatal(err)
	}
	if got != nodeDir {
		t.Fatalf("got %q, want %q", got, nodeDir)
	}
}

func TestNodeBinDirMissing(t *testing.T) {
	if _, err := nodeBinDir(t.TempDir(), "v99.0.0"); err == nil {
		t.Fatal("expected error when node is missing")
	}
}

func TestPiArgsChatOpenAI(t *testing.T) {
	p, _ := LookupProvider("openai")
	got := piArgs(p, p.DefaultModel, "chat", nil, false, "")
	want := []string{"--provider", "openai", "--model", "openai/gpt-5.6-terra", "--offline"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestPiArgsOllamaAllowsLocalCatalogRefresh(t *testing.T) {
	p, err := LookupProvider("ollama")
	if err != nil {
		t.Fatal(err)
	}
	got := piArgs(p, p.DefaultModel, "chat", nil, false, "")
	want := []string{"--provider", "ollama", "--model", "ollama/llama3.1"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	for _, arg := range got {
		if arg == "--offline" {
			t.Fatalf("ollama must not be offline because its local catalog must refresh: %v", got)
		}
	}
}

func TestPiArgsPrintDeepSeek(t *testing.T) {
	p, _ := LookupProvider("deepseek")
	got := piArgs(p, p.DefaultModel, "print", []string{"hello"}, false, "")
	want := []string{"--provider", "deepseek", "--model", "deepseek/deepseek-v4-flash", "--offline", "-p", "--no-session", "hello"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestPiArgsPrintPinnedSessionPersists(t *testing.T) {
	// EVAL-17 enabler: an explicit --session-id must keep the session (no
	// --no-session) even with persist=false, so the chat-session runner's
	// turn 1 (print) can be continued by resume without a cumulative budget cap.
	p, _ := LookupProvider("openai")
	got := piArgs(p, p.DefaultModel, "print", []string{"--session-id", "eval-coding-056", "hello"}, false, "")
	want := []string{"--provider", "openai", "--model", "openai/gpt-5.6-terra", "--offline", "-p", "--session-id", "eval-coding-056", "hello"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	for _, a := range got {
		if a == "--no-session" {
			t.Fatalf("pinned session must not be ephemeral: %v", got)
		}
	}
}

func TestPiArgsPassThroughFlagsAndMessage(t *testing.T) {
	p, _ := LookupProvider("openai")
	got := piArgs(p, "openai/gpt-5.6-terra", "chat", []string{"--tools", "read", "refactor x"}, false, "")
	want := []string{"--provider", "openai", "--model", "openai/gpt-5.6-terra", "--offline", "--tools", "read", "refactor x"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestPiArgsResumeAppendsContinue(t *testing.T) {
	p, _ := LookupProvider("openai")
	got := piArgs(p, p.DefaultModel, "resume", nil, false, "")
	want := []string{"--provider", "openai", "--model", "openai/gpt-5.6-terra", "--offline", "--continue"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestPiArgsResumePreservesPassThrough(t *testing.T) {
	p, _ := LookupProvider("deepseek")
	got := piArgs(p, "deepseek/deepseek-v4-flash", "resume", []string{"--session", "abc123", "continue refactor"}, false, "")
	// A session pin replaces --continue (pi rejects --continue + a pin): the
	// EVAL-17 runner pins eval-<case> on resume turns.
	want := []string{"--provider", "deepseek", "--model", "deepseek/deepseek-v4-flash", "--offline", "--session", "abc123", "continue refactor"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestChildEnvForcesIPv4FirstDNS(t *testing.T) {
	t.Setenv("NODE_OPTIONS", "--max-old-space-size=4096")
	env := childEnv("/fake/node/bin", []string{"DEEPSEEK_API_KEY=sk-x"})
	joined := strings.Join(env, "\n")
	if !strings.Contains(joined, "NODE_OPTIONS=--dns-result-order=ipv4first") {
		t.Fatal("child env must force IPv4-first DNS")
	}
	if strings.Contains(joined, "--max-old-space-size") {
		t.Fatal("pre-existing NODE_OPTIONS must be overridden, not appended")
	}
	if !strings.Contains(joined, "PATH=/fake/node/bin"+string(os.PathListSeparator)) {
		t.Fatal("PATH must have the node bin dir prepended")
	}
	if !strings.Contains(joined, "DEEPSEEK_API_KEY=sk-x") {
		t.Fatal("extra env must be present")
	}
}

// makeNvmNodeTree creates a fake nvm node dir at home with the given versions.
func makeNvmNodeTree(t *testing.T, home string, versions ...string) {
	t.Helper()
	for _, v := range versions {
		dir := filepath.Join(home, ".nvm", "versions", "node", v, "bin")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "node"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResolveNodeVersionPicksHighest(t *testing.T) {
	t.Setenv("PI_NODE_VERSION", "")
	home := t.TempDir()
	makeNvmNodeTree(t, home, "v18.0.0", "v22.17.0", "v22.19.0")
	got, err := resolveNodeVersion(home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "v22.19.0" {
		t.Fatalf("got %q, want %q", got, "v22.19.0")
	}
}

func TestResolveNodeVersionNoNodeInstalled(t *testing.T) {
	t.Setenv("PI_NODE_VERSION", "")
	home := t.TempDir()
	// nvm dir exists but no node versions installed
	if err := os.MkdirAll(filepath.Join(home, ".nvm", "versions", "node"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := resolveNodeVersion(home)
	if err == nil {
		t.Fatal("expected error when no node installed")
	}
	if !strings.Contains(err.Error(), "nvm install node") {
		t.Fatalf("error should mention 'nvm install node', got: %v", err)
	}
}

func TestResolveNodeVersionNoNvm(t *testing.T) {
	t.Setenv("PI_NODE_VERSION", "")
	home := t.TempDir() // no .nvm at all
	_, err := resolveNodeVersion(home)
	if err == nil {
		t.Fatal("expected error when nvm missing")
	}
	if !strings.Contains(err.Error(), "install") {
		t.Fatalf("error should mention install guidance, got: %v", err)
	}
}

func TestResolveNodeVersionIgnoresNonSemver(t *testing.T) {
	t.Setenv("PI_NODE_VERSION", "")
	home := t.TempDir()
	// only a stray non-semver dir (e.g. v22 with no dots) exists
	if err := os.MkdirAll(filepath.Join(home, ".nvm", "versions", "node", "v22"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := resolveNodeVersion(home)
	if err == nil {
		t.Fatal("expected error when only non-semver dir present")
	}
	if !strings.Contains(err.Error(), "no Node installed") {
		t.Fatalf("error should mention 'no Node installed', got: %v", err)
	}
}

func TestResolveNodeVersionEnvOverride(t *testing.T) {
	t.Setenv("PI_NODE_VERSION", "v20.0.0")
	home := t.TempDir() // no nvm tree
	got, err := resolveNodeVersion(home)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "v20.0.0" {
		t.Fatalf("got %q, want %q", got, "v20.0.0")
	}
}

func TestLaunchEnvAddsBaseURLOnlyWhenConfigured(t *testing.T) {
	without := launchEnv(Provider{KeyEnv: "OPENAI_API_KEY"}, "testvalue")
	for _, item := range without {
		if strings.HasPrefix(item, "OPENAI_BASE_URL=") {
			t.Fatalf("unexpected base URL in environment: %q", item)
		}
	}

	with := launchEnv(Provider{KeyEnv: "LOCAL_API_KEY", BaseURL: "http://localhost:11434/v1"}, "testvalue")
	if !strings.Contains(strings.Join(with, "\n"), "OPENAI_BASE_URL=http://localhost:11434/v1") {
		t.Fatalf("launch environment missing configured base URL: %v", with)
	}
}

// TestLaunchEnvNonInteractive proves every spawned pi process (and therefore
// every child bash tool) gets the non-interactive shell environment, so git
// editors / pagers can never block an agent run forever.
func TestLaunchEnvNonInteractive(t *testing.T) {
	env := strings.Join(launchEnv(Provider{KeyEnv: "OPENAI_API_KEY"}, "testvalue"), "\n")
	for _, want := range []string{
		"GIT_EDITOR=true",
		"GIT_SEQUENCE_EDITOR=true",
		"GIT_TERMINAL_PROMPT=0",
		"PAGER=cat",
	} {
		if !strings.Contains(env, want) {
			t.Fatalf("launch environment missing non-interactive var %q: %v", want, env)
		}
	}
}

// TestNonInteractiveEnvPinned pins the single source of truth itself, so the
// doctor regression guard cannot be silently defeated by emptying the list.
func TestNonInteractiveEnvPinned(t *testing.T) {
	want := []string{"GIT_EDITOR=true", "GIT_SEQUENCE_EDITOR=true", "GIT_TERMINAL_PROMPT=0", "PAGER=cat"}
	if len(nonInteractiveEnv) != len(want) {
		t.Fatalf("nonInteractiveEnv has %d entries, want %d: %v", len(nonInteractiveEnv), len(want), nonInteractiveEnv)
	}
	for i := range want {
		if nonInteractiveEnv[i] != want[i] {
			t.Fatalf("nonInteractiveEnv[%d] = %q, want %q", i, nonInteractiveEnv[i], want[i])
		}
	}
}

func TestLaunchEnvAnthropicCompatibleProvider(t *testing.T) {
	// Anthropic-routed providers (e.g. AWS Bedrock) must hand pi the key under
	// ANTHROPIC_API_KEY and the base URL under ANTHROPIC_BASE_URL so pi's
	// anthropic provider can reach the compatible endpoint.
	bedrock := Provider{KeyEnv: "BEDROCK_API_KEY", PiProvider: "anthropic", BaseURL: "https://bedrock-runtime.us-east-1.amazonaws.com/anthropic/v1"}
	env := strings.Join(launchEnv(bedrock, "testvalue"), "\n")
	for _, want := range []string{
		"BEDROCK_API_KEY=testvalue",
		"ANTHROPIC_API_KEY=testvalue",
		"ANTHROPIC_BASE_URL=https://bedrock-runtime.us-east-1.amazonaws.com/anthropic/v1",
	} {
		if !strings.Contains(env, want) {
			t.Fatalf("launch environment missing %q: %v", want, env)
		}
	}

	// The plain anthropic entry keeps its own key env and must not duplicate
	// ANTHROPIC_API_KEY, but must still carry the non-interactive vars.
	plain := launchEnv(Provider{KeyEnv: "ANTHROPIC_API_KEY", PiProvider: "anthropic"}, "testvalue")
	plainText := strings.Join(plain, "\n")
	if !strings.Contains(plainText, "ANTHROPIC_API_KEY=testvalue") {
		t.Fatalf("plain anthropic launch environment changed: %v", plain)
	}
	if !strings.Contains(plainText, "GIT_EDITOR=true") {
		t.Fatalf("plain anthropic launch environment missing non-interactive vars: %v", plain)
	}
}

func TestChildEnvStripsOtherProviderKeys(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-openai")
	t.Setenv("DEEPSEEK_API_KEY", "sk-deepseek")
	t.Setenv("OPENROUTER_API_KEY", "sk-or")
	t.Setenv("HOME", "/tmp/fake-home")
	env := childEnv("/fake/node/bin", []string{"OPENAI_API_KEY=sk-active"})
	joined := "\n" + strings.Join(env, "\n") + "\n"
	for _, want := range []string{"OPENAI_API_KEY=sk-active", "HOME=/tmp/fake-home", "PATH=/fake/node/bin" + string(os.PathListSeparator)} {
		if !strings.Contains(joined, "\n"+want) {
			t.Fatalf("child env must keep %q; got:\n%s", want, joined)
		}
	}
	for _, absent := range []string{"DEEPSEEK_API_KEY=sk-deepseek", "OPENROUTER_API_KEY=sk-or", "OPENAI_API_KEY=sk-openai"} {
		if strings.Contains(joined, "\n"+absent+"\n") {
			t.Fatalf("child env must strip unrelated provider key %q; got:\n%s", absent, joined)
		}
	}
}

func TestChildEnvStripsDefaultKeysWhenActiveTableKeyless(t *testing.T) {
	orig := Providers
	t.Cleanup(func() { Providers = orig })
	// A project-local table defining ONLY a keyless local gateway omits the
	// canonical provider names; a default-provider credential inherited from
	// the parent env must still be stripped (CodeRabbit SEC-1 finding).
	Providers = []Provider{{Name: "local-gateway", KeyEnv: "LOCAL_GATEWAY_KEY", DefaultModel: "x/y", BaseURL: "http://localhost:9999/v1", Keyless: true}}
	t.Setenv("OPENAI_API_KEY", "sk-openai")
	t.Setenv("LOCAL_GATEWAY_KEY", "sk-gw")
	env := childEnv("/fake/node/bin", []string{"LOCAL_GATEWAY_KEY=sk-active"})
	joined := "\n" + strings.Join(env, "\n") + "\n"
	if strings.Contains(joined, "\nOPENAI_API_KEY=sk-openai\n") {
		t.Fatal("default-provider key must be stripped even when absent from the active table")
	}
	if !strings.Contains(joined, "\nLOCAL_GATEWAY_KEY=sk-active\n") {
		t.Fatal("active keyless provider's credential must still pass through via extraEnv")
	}
}

func TestChildEnvStripsSecretManagerSessionTokens(t *testing.T) {
	t.Setenv("BW_SESSION", "vault-session-token")
	t.Setenv("BW_ACCOUNT_ID", "account-id")
	t.Setenv("OP_SESSION_MY_VAULT", "op-token")
	t.Setenv("HOME", "/tmp/fake-home")
	env := childEnv("/fake/node/bin", []string{"OPENAI_API_KEY=sk-active"})
	joined := "\n" + strings.Join(env, "\n") + "\n"
	for _, absent := range []string{"BW_SESSION=vault-session-token", "BW_ACCOUNT_ID=account-id", "OP_SESSION_MY_VAULT=op-token"} {
		if strings.Contains(joined, "\n"+absent+"\n") {
			t.Fatalf("child env must strip secret-manager token %q (bw/op unusable in sessions); got:\n%s", absent, joined)
		}
	}
	if !strings.Contains(joined, "\nOPENAI_API_KEY=sk-active\n") {
		t.Fatal("active provider credential must still be present")
	}
	if !strings.Contains(joined, "\nHOME=/tmp/fake-home\n") {
		t.Fatal("non-secret env must survive")
	}
}

func TestChildEnvStripsCaseInsensitiveCredentialShapedNames(t *testing.T) {
	// Windows env names are case-insensitive (os.Environ preserves raw casing):
	// lowercase provider/secret-manager names must not bypass the filters.
	t.Setenv("bw_session", "lower-session")
	t.Setenv("openai_api_key", "lower-provider")
	env := childEnv("/fake/node/bin", []string{"OPENAI_API_KEY=sk-active"})
	joined := "\n" + strings.Join(env, "\n") + "\n"
	for _, absent := range []string{"bw_session=lower-session", "openai_api_key=lower-provider"} {
		if strings.Contains(joined, "\n"+absent+"\n") {
			t.Fatalf("lowercase credential-shaped names must be stripped; got:\n%s", joined)
		}
	}
	if !strings.Contains(joined, "\nOPENAI_API_KEY=sk-active\n") {
		t.Fatal("active provider credential must still be present")
	}
}
