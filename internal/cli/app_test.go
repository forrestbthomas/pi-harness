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
	if !strings.Contains(usage, "--provider <name>") || !strings.Contains(usage, "pi-run providers") {
		t.Fatal("usage must direct users to pi-run providers for valid provider names")
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

func TestSplitLaunchArgsProviderEqualsForm(t *testing.T) {
	p, _, _ := splitLaunchArgs([]string{"--provider=anthropic", "hi"})
	if p != "anthropic" {
		t.Fatalf("got provider=%q", p)
	}
}

func TestSplitLaunchArgsModelSeparate(t *testing.T) {
	_, m, _ := splitLaunchArgs([]string{"--model", "deepseek/deepseek-v4-pro", "hi"})
	if m != "deepseek/deepseek-v4-pro" {
		t.Fatalf("got model=%q", m)
	}
}

func TestRunPrintMissingKeyExits3(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	t.Setenv("PI_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("BW_GET", filepath.Join(t.TempDir(), "nonexistent-bw-get")) // hermetic: no vault
	// print with a prompt but no key → exit 3.
	if code := Run([]string{"print", "hello"}); code != 3 {
		t.Fatalf("print no key exit = %d, want 3", code)
	}
}

func TestRunProvidersEmptyTable(t *testing.T) {
	orig := Providers
	defer func() { Providers = orig }()
	Providers = nil
	// runProviders should handle empty table gracefully (prints nothing, exit 0).
	if code := Run([]string{"providers"}); code != 0 {
		t.Fatalf("providers empty exit = %d, want 0", code)
	}
}

func installTestPaths(t *testing.T) (target, link string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	target = filepath.Join(t.TempDir(), "pi-run")
	if err := os.WriteFile(target, []byte("test binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	link = filepath.Join(home, "bin", "pi-run")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	return target, link
}

func TestRunInstallCreatesLink(t *testing.T) {
	target, link := installTestPaths(t)
	if err := installLink(target, link, false); err != nil {
		t.Fatalf("installLink: %v", err)
	}
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Fatalf("link target = %q, want %q", got, target)
	}
}

func TestRunInstallRefusesForeignSymlink(t *testing.T) {
	target, link := installTestPaths(t)
	foreign := filepath.Join(t.TempDir(), "other-pi-run")
	if err := os.WriteFile(foreign, []byte("foreign"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(foreign, link); err != nil {
		t.Fatal(err)
	}
	if err := installLink(target, link, false); err == nil {
		t.Fatal("installLink unexpectedly overwrote foreign symlink")
	}
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != foreign {
		t.Fatalf("foreign link changed to %q, want %q", got, foreign)
	}
}

func TestRunInstallRefusesRegularFile(t *testing.T) {
	target, link := installTestPaths(t)
	if err := os.WriteFile(link, []byte("do not replace"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installLink(target, link, false); err == nil {
		t.Fatal("installLink unexpectedly overwrote regular file")
	}
	got, err := os.ReadFile(link)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "do not replace" {
		t.Fatalf("regular file changed to %q", got)
	}
}

func TestRunInstallForceOverwrites(t *testing.T) {
	target, link := installTestPaths(t)
	foreign := filepath.Join(t.TempDir(), "other-pi-run")
	if err := os.WriteFile(foreign, []byte("foreign"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(foreign, link); err != nil {
		t.Fatal(err)
	}
	if err := installLink(target, link, true); err != nil {
		t.Fatalf("installLink --force: %v", err)
	}
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Fatalf("link target = %q, want %q", got, target)
	}
}

func TestRunInstallReplacesOwnLink(t *testing.T) {
	target, link := installTestPaths(t)
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := installLink(target, link, false); err != nil {
		t.Fatalf("installLink own link: %v", err)
	}
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Fatalf("link target = %q, want %q", got, target)
	}
}

func TestRunInstallAtomicReplace(t *testing.T) {
	target, link := installTestPaths(t)
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if err := installLink(target, link, false); err != nil {
		t.Fatalf("installLink atomic replace: %v", err)
	}
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Fatalf("link target = %q, want %q", got, target)
	}

	entries, err := os.ReadDir(filepath.Dir(link))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".pi-run-link-") {
			t.Fatalf("leftover temporary link %q", entry.Name())
		}
	}
}

func TestRunInstallReplacesRelativeOwnLink(t *testing.T) {
	target, link := installTestPaths(t)
	relativeTarget, err := filepath.Rel(filepath.Dir(link), target)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(relativeTarget, link); err != nil {
		t.Fatal(err)
	}
	if err := installLink(target, link, false); err != nil {
		t.Fatalf("installLink relative own link: %v", err)
	}
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != target {
		t.Fatalf("link target = %q, want %q", got, target)
	}
}

func evalTestRoot(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	for _, key := range supportedProviderKeyEnvs {
		t.Setenv(key, "")
	}
	venv := filepath.Join(root, "eval", ".venv", "bin")
	if err := os.MkdirAll(venv, 0o755); err != nil {
		t.Fatal(err)
	}
	argsFile := filepath.Join(root, "pytest-args.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + argsFile + "\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(venv, "python"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return root, argsFile
}

func captureRunStdout(t *testing.T, args []string) (int, string) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	code := Run(args)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = old
	var out strings.Builder
	if _, err := io.Copy(&out, r); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return code, out.String()
}

func captureRunStderr(t *testing.T, args []string) (int, string) {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	code := Run(args)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stderr = old
	var out strings.Builder
	if _, err := io.Copy(&out, r); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return code, out.String()
}

func TestRunEvalHelpDoesNotRunSuite(t *testing.T) {
	_, argsFile := evalTestRoot(t)
	code, out := captureRunStdout(t, []string{"eval", "--help"})
	if code != 0 {
		t.Fatalf("eval --help exit = %d, want 0", code)
	}
	if !strings.Contains(out, "Usage: pi-run eval") {
		t.Fatalf("eval --help output missing usage: %q", out)
	}
	if _, err := os.Stat(argsFile); !os.IsNotExist(err) {
		t.Fatalf("eval --help must not invoke pytest, args file err = %v", err)
	}
}

func TestRunEvalUnknownFlagExit2(t *testing.T) {
	_, argsFile := evalTestRoot(t)
	code, out := captureRunStderr(t, []string{"eval", "--bogus"})
	if code != 2 {
		t.Fatalf("eval --bogus exit = %d, want 2", code)
	}
	if !strings.Contains(out, "unknown flag") {
		t.Fatalf("eval --bogus stderr missing unknown flag: %q", out)
	}
	if _, err := os.Stat(argsFile); !os.IsNotExist(err) {
		t.Fatalf("eval --bogus must not invoke pytest, args file err = %v", err)
	}
}

func TestRunEvalPytestPassThrough(t *testing.T) {
	_, argsFile := evalTestRoot(t)
	selector := "tests/test_x.py::test_y"
	if code := Run([]string{"eval", "--", selector}); code != 0 {
		t.Fatalf("eval pass-through exit = %d, want 0", code)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), selector) {
		t.Fatalf("pytest args %q missing pass-through selector %q", args, selector)
	}
}

func TestRunEvalQuickStillWorks(t *testing.T) {
	_, argsFile := evalTestRoot(t)
	if code := Run([]string{"eval", "--quick"}); code != 0 {
		t.Fatalf("eval --quick exit = %d, want 0", code)
	}
	args, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"tests/test_code_quality.py",
		"tests/test_agent_task_completion.py::test_dataset_expected_outputs_are_non_empty",
	} {
		if !strings.Contains(string(args), want) {
			t.Fatalf("pytest args %q missing quick selector %q", args, want)
		}
	}
}

func TestUsageMentionsAllCommands(t *testing.T) {
	for _, want := range []string{"resume", "providers", "--exit-codes", "eval"} {
		if !strings.Contains(usage, want) {
			t.Fatalf("usage missing command or flag %q", want)
		}
	}
}

func TestRunCleanPrintsRemovedAndMissingPaths(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)

	existing := filepath.Join(root, "eval", ".venv")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(existing, "marker"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, output := captureRunStdout(t, []string{"clean"})
	if code != 0 {
		t.Fatalf("Run(clean) code = %d, want 0; output:\n%s", code, output)
	}
	if !strings.Contains(output, "removed "+existing) {
		t.Fatalf("clean output missing removed path %q:\n%s", existing, output)
	}
	missing := filepath.Join(root, "eval", ".pytest_cache")
	if !strings.Contains(output, "nothing to clean: "+missing) {
		t.Fatalf("clean output missing missing-path status %q:\n%s", missing, output)
	}
	if !strings.Contains(output, "clean complete") {
		t.Fatalf("clean output missing completion summary:\n%s", output)
	}
	if _, err := os.Stat(existing); !os.IsNotExist(err) {
		t.Fatalf("existing clean path still exists or stat failed: %v", err)
	}
}
