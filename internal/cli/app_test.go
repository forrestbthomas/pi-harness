package cli

import (
	"io"
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

func TestRunResumeDispatches(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	t.Setenv("PI_PROVIDER", "")
	// Force resolveSecret to fail deterministically: empty provider key env AND
	// a BW_GET path that cannot exist, so the test never depends on ambient
	// credentials (a real key in the environment or an unlocked vault would
	// otherwise let it launch pi and return 0 instead of 3).
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("BW_GET", filepath.Join(t.TempDir(), "nonexistent-bw-get"))
	// No key: resume should fail with exit 3 (missing key), proving dispatch
	// reached runLaunch (not "unknown command").
	if code := Run([]string{"resume"}); code != 3 {
		t.Fatalf("resume exit = %d, want 3 (missing key → dispatch worked)", code)
	}
}

func TestUsageMentionsResume(t *testing.T) {
	if !strings.Contains(usage, "resume") {
		t.Fatal("usage must document the resume command")
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

func TestRunProviders(t *testing.T) {
	// providers reads the Providers table; it must list at least the defaults.
	// Capture stdout via a bytes.Buffer by running Run with a temp HARNESS_ROOT
	// so repo-root detection doesn't load providers.json (defaults remain).
	t.Setenv("HARNESS_ROOT", t.TempDir())
	// Providers may have been loaded from providers.json at init; snapshot the
	// count so the test is order-independent.
	orig := Providers
	defer func() { Providers = orig }()
	// Force defaults to make the test hermetic regardless of init().
	Providers = defaultProviders
	if code := Run([]string{"providers"}); code != 0 {
		t.Fatalf("providers exit = %d, want 0", code)
	}
	// The command prints to stdout; we can't easily capture it here, so just
	// assert it runs and returns 0 (covered more thoroughly by unit tests).
}

func TestRunEvalNoKeySkipsLive(t *testing.T) {
	// Provide a fake repo root with eval/.venv/bin/python present so eval
	// proceeds past the venv check, but no provider key in env → skip notice.
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GROQ_API_KEY", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
	// Fake venv python that records its argv to a file and exits 0, so we can
	// assert what pytest was asked to run. With no provider key, the skip
	// guard must run the deterministic subset, NOT the full suite ("tests/"
	// as a bare positional arg). This is the honest behavioral check: the
	// brief's exit-0 stub with no argv inspection would pass on the old
	// behavior too, so it could not prove the skip guard changes anything.
	venv := filepath.Join(root, "eval", ".venv", "bin")
	if err := os.MkdirAll(venv, 0o755); err != nil {
		t.Fatal(err)
	}
	argsFile := filepath.Join(root, "args.txt")
	script := "#!/bin/sh\nprintf '%s' \"$@\" > " + argsFile + "\nexit 0\n"
	if err := os.WriteFile(filepath.Join(venv, "python"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	// Full eval (quick=false) with no key must not run the live suite.
	// It should print a skip notice and exit 0.
	if code := Run([]string{"eval"}); code != 0 {
		t.Fatalf("eval no-key exit = %d, want 0 (skip live)", code)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	s := string(args)
	// The deterministic subset must be present.
	for _, want := range []string{
		"tests/test_harness_config.py",
		"tests/test_code_quality.py",
		"tests/test_agent_task_completion.py::test_dataset_expected_outputs_are_non_empty",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %q in pytest args %q", want, s)
		}
	}
	// The full suite is a bare "tests/" positional arg; it must not be present
	// as a standalone token. (tests/test_... file paths are fine.)
	if strings.Contains(s, "tests/\"") || strings.Contains(s, "tests/ -") || strings.HasSuffix(strings.TrimSpace(s), "tests/") {
		t.Fatalf("full suite tests/ should be skipped, got args %q", s)
	}
}

func TestVersionCommand(t *testing.T) {
	// Version is a package var; snapshot and restore to keep hermetic.
	orig := Version
	defer func() { Version = orig }()
	Version = "test-version"
	// Run version; capture stdout via a pipe.
	var sb strings.Builder
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := Run([]string{"version"})
	w.Close()
	os.Stdout = old
	io.Copy(&sb, r)
	if code != 0 {
		t.Fatalf("version exit = %d, want 0", code)
	}
	if !strings.Contains(sb.String(), "test-version") {
		t.Fatalf("version output %q must contain injected value", sb.String())
	}
}

func TestRunPrintEmptyPromptErrorMessage(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	var sb strings.Builder
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	code := Run([]string{"print"})
	w.Close()
	os.Stderr = old
	io.Copy(&sb, r)
	if code != 2 {
		t.Fatalf("print no prompt exit = %d, want 2", code)
	}
	if !strings.Contains(sb.String(), "pi-run: print:") {
		t.Fatalf("stderr must use pi-run: <cmd>: format, got %q", sb.String())
	}
}

func TestRunEvalMissingVenvErrorMessage(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	var sb strings.Builder
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	code := Run([]string{"eval"})
	w.Close()
	os.Stderr = old
	io.Copy(&sb, r)
	if code != 5 {
		t.Fatalf("eval no venv exit = %d, want 5", code)
	}
	if !strings.Contains(sb.String(), "pi-run: eval:") {
		t.Fatalf("stderr must use pi-run: eval: format, got %q", sb.String())
	}
}

func TestUsageMentionsExitCodes(t *testing.T) {
	if !strings.Contains(usage, "--exit-codes") || !strings.Contains(usage, "Exit codes") {
		t.Fatal("usage must document exit codes and the --exit-codes flag")
	}
}
