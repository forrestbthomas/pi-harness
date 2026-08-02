package cli

import (
	"os"
	"path/filepath"
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
	got := piArgs(p, p.DefaultModel, "chat", nil)
	want := []string{"--provider", "openai", "--model", "openai/gpt-5.6-terra"}
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
	want := []string{"--provider", "deepseek", "--model", "deepseek/deepseek-v4-flash", "-p", "--no-session", "hello"}
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
	want := []string{"--provider", "openai", "--model", "openai/gpt-5.6-terra", "--tools", "read", "refactor x"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
