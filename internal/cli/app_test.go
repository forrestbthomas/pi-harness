package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSplitLaunchArgsProvider(t *testing.T) {
	p, _, rest := splitLaunchArgs([]string{"--provider", "deepseek", "hello"})
	if p != "deepseek" || len(rest) != 1 || rest[0] != "hello" {
		t.Fatalf("got provider=%q rest=%v", p, rest)
	}
}

func TestSplitLaunchArgsModelEquals(t *testing.T) {
	p, m, _ := splitLaunchArgs([]string{"--provider=openrouter", "--model=deepseek/deepseek-chat"})
	if p != "openrouter" || m != "deepseek/deepseek-chat" {
		t.Fatalf("got provider=%q model=%q", p, m)
	}
}

func TestSplitLaunchArgsKeepsEverythingElse(t *testing.T) {
	_, _, rest := splitLaunchArgs([]string{"--tools", "read", "--thinking", "high", "hi there"})
	want := []string{"--tools", "read", "--thinking", "high", "hi there"}
	if len(rest) != len(want) {
		t.Fatalf("got %v, want %v", rest, want)
	}
	for i := range want {
		if rest[i] != want[i] {
			t.Fatalf("got %v, want %v", rest, want)
		}
	}
}

func TestSplitLaunchArgsDoubleDashEscapesTail(t *testing.T) {
	_, _, rest := splitLaunchArgs([]string{"--", "--provider", "x"})
	if len(rest) != 2 || rest[0] != "--provider" {
		t.Fatalf("got %v", rest)
	}
}

func TestRunVersion(t *testing.T) {
	if code := Run([]string{"version"}); code != 0 {
		t.Fatalf("version exit = %d", code)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if code := Run([]string{"frobnicate"}); code != 2 {
		t.Fatalf("unknown command exit = %d, want 2", code)
	}
}

func TestRunPrintEmptyPrompt(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	// Prompt check happens before key resolution: exit 2, not 3.
	if code := Run([]string{"print"}); code != 2 {
		t.Fatalf("print with no prompt exit = %d, want 2", code)
	}
}

func TestRunEvalWithoutVenv(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir()) // no eval/.venv inside
	if code := Run([]string{"eval"}); code != 5 {
		t.Fatalf("eval exit = %d, want 5 (venv missing)", code)
	}
}

func TestUsageMentionsProviders(t *testing.T) {
	if !strings.Contains(usage, "deepseek") || !strings.Contains(usage, "openrouter") {
		t.Fatal("usage must document all providers")
	}
}

func TestModulePath(t *testing.T) {
	// The public module path must be github.com/forrestthomas1/pi-harness.
	// Tests run with CWD = the package dir and os.Executable() = the temp test
	// binary, so derive the repo root from this source file's own path.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// file = <root>/internal/cli/app_test.go
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "module github.com/forrestthomas1/pi-harness") {
		t.Fatalf("go.mod must declare module github.com/forrestthomas1/pi-harness, got:\n%s", b)
	}
}
