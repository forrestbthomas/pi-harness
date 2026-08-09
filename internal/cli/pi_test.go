package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNodeBinDirOK(t *testing.T) {
	home := t.TempDir()
	nodeDir := filepath.Join(home, ".nvm", "versions", "node", "v"+defaultNodeVersion, "bin")
	if err := os.MkdirAll(nodeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeDir, "node"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := nodeBinDir(home, "v"+defaultNodeVersion)
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
	got := piArgs(p, p.DefaultModel, "chat", nil)
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

func TestPiArgsPrintDeepSeek(t *testing.T) {
	p, _ := LookupProvider("deepseek")
	got := piArgs(p, p.DefaultModel, "print", []string{"hello"})
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

func TestPiArgsPassThroughFlagsAndMessage(t *testing.T) {
	p, _ := LookupProvider("openai")
	got := piArgs(p, "openai/gpt-5.6-terra", "chat", []string{"--tools", "read", "refactor x"})
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
	got := piArgs(p, p.DefaultModel, "resume", nil)
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
	got := piArgs(p, "deepseek/deepseek-v4-flash", "resume", []string{"--session", "abc123", "continue refactor"})
	want := []string{"--provider", "deepseek", "--model", "deepseek/deepseek-v4-flash", "--offline", "--continue", "--session", "abc123", "continue refactor"}
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
