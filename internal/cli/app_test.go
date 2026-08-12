package cli

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestSplitLaunchArgsProvider(t *testing.T) {
	p, _, _, _, _, rest := splitLaunchArgs([]string{"--provider", "deepseek", "hello"})
	if p != "deepseek" || len(rest) != 1 || rest[0] != "hello" {
		t.Fatalf("got provider=%q rest=%v", p, rest)
	}
}

func TestSplitLaunchArgsModelEquals(t *testing.T) {
	p, m, _, _, _, _ := splitLaunchArgs([]string{"--provider=openrouter", "--model=deepseek/deepseek-chat"})
	if p != "openrouter" || m != "deepseek/deepseek-chat" {
		t.Fatalf("got provider=%q model=%q", p, m)
	}
}

func TestSplitLaunchArgsKeepsEverythingElse(t *testing.T) {
	_, _, _, _, _, rest := splitLaunchArgs([]string{"--tools", "read", "--thinking", "high", "hi there"})
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
	_, _, _, _, _, rest := splitLaunchArgs([]string{"--", "--provider", "x"})
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
	t.Setenv("PI_PERMISSION_MODE", "")
	t.Setenv("PI_MAX_BUDGET_USD", "")
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
	p, _, _, _, _, _ := splitLaunchArgs([]string{"--provider=anthropic", "hi"})
	if p != "anthropic" {
		t.Fatalf("got provider=%q", p)
	}
}

func TestSplitLaunchArgsModelSeparate(t *testing.T) {
	_, m, _, _, _, _ := splitLaunchArgs([]string{"--model", "deepseek/deepseek-v4-pro", "hi"})
	if m != "deepseek/deepseek-v4-pro" {
		t.Fatalf("got model=%q", m)
	}
}

func TestRunPrintMissingKeyExits3(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	t.Setenv("PI_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("BW_GET", filepath.Join(t.TempDir(), "nonexistent-bw-get")) // hermetic: no vault
	t.Setenv("PI_PERMISSION_MODE", "")
	t.Setenv("PI_MAX_BUDGET_USD", "")
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

// TestRunInstallForceFlagParsing exercises the command-level install flag
// parser (Run with a temp repo root and a stub go binary) to prove --force is
// parsed and reaches installLink, and that unknown flags are rejected before
// any build happens. The installLink behavior itself is covered by the unit
// tests above; this test covers the Run-level parse path that those miss.
func TestRunInstallForceFlagParsing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	// Stub `go` on PATH so runInstall's build step succeeds without a toolchain.
	// The build invocation is `go build -ldflags <ldflags> -o <target> ./cmd/pi-run`,
	// so $1=build, $2=-ldflags, $3=<ldflags>, $4=-o, $5=<target>.
	stubDir := t.TempDir()
	goStub := filepath.Join(stubDir, "go")
	script := "#!/bin/sh\nprintf 'stub' > \"$5\"\nexit 0\n"
	if err := os.WriteFile(goStub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	home := t.TempDir()
	t.Setenv("HOME", home)
	link := filepath.Join(home, "bin", "pi-run")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}

	// install --bogus: unknown flag must be rejected (exit 2) before building.
	code, out := captureRunStderr(t, []string{"install", "--bogus"})
	if code != 2 {
		t.Fatalf("install --bogus exit = %d, want 2; stderr: %s", code, out)
	}
	if !strings.Contains(out, "unknown flag") {
		t.Fatalf("install --bogus stderr missing unknown-flag error: %q", out)
	}
	if _, err := os.Stat(filepath.Join(root, "bin", "pi-run")); !os.IsNotExist(err) {
		t.Fatalf("install --bogus must not build; bin/pi-run exists: %v", err)
	}

	// install --force: parse succeeds, builds, and creates the symlink.
	code, out = captureRunStdout(t, []string{"install", "--force"})
	if code != 0 {
		t.Fatalf("install --force exit = %d, want 0; stderr: %s", code, out)
	}
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("install --force did not create symlink: %v", err)
	}
	want := filepath.Join(root, "bin", "pi-run")
	if got != want {
		t.Fatalf("link target = %q, want %q", got, want)
	}
}

// TestRunInstallHelpFlag verifies install --help prints usage and exits 0
// without attempting a build.
func TestRunInstallHelpFlag(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	code, out := captureRunStdout(t, []string{"install", "--help"})
	if code != 0 {
		t.Fatalf("install --help exit = %d, want 0", code)
	}
	if !strings.Contains(out, "Usage: pi-run install") {
		t.Fatalf("install --help output missing usage: %q", out)
	}
	if _, err := os.Stat(filepath.Join(root, "bin", "pi-run")); !os.IsNotExist(err) {
		t.Fatalf("install --help must not build; bin/pi-run exists: %v", err)
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

// usageCommandsBlock returns the text of the usage const's Commands: block
// (between the "Commands:" header and the following "Exit codes:" line in
// app.go).
func usageCommandsBlock(src string) string {
	start := strings.Index(src, "Commands:")
	if start < 0 {
		return ""
	}
	rest := src[start+len("Commands:"):]
	if end := strings.Index(rest, "Exit codes:"); end >= 0 {
		rest = rest[:end]
	}
	return rest
}

// usageCommandRe matches one documented command line in the Commands: block:
// two leading spaces, the command word (\w or -), then whitespace. The
// project-understand continuation line (16 leading spaces) never matches.
var usageCommandRe = regexp.MustCompile(`(?m)^  ([\w-]+)\s`)

// parseUsageCommands derives the documented command set from the usage text.
func parseUsageCommands(src string) map[string]bool {
	cmds := map[string]bool{}
	for _, m := range usageCommandRe.FindAllStringSubmatch(usageCommandsBlock(src), -1) {
		cmds[m[1]] = true
	}
	return cmds
}

// runDispatchFrom extracts the top-level switch's case-label groups and the
// handled command set (case labels plus the pre-switch --exit-codes special
// case) from Run in app.go. The nested install flag switch is excluded by
// brace-depth tracking, so --force never leaks into the command surface.
func runDispatchFrom(src string) (groups [][]string, handled map[string]bool) {
	handled = map[string]bool{}
	fnStart := strings.Index(src, "func Run(args []string) int {")
	if fnStart < 0 {
		return nil, handled
	}
	fn := src[fnStart:]
	swStart := strings.Index(fn, "switch args[0] {")
	if swStart < 0 {
		return nil, handled
	}
	body := fn[swStart+len("switch args[0] {"):]
	depth := 1
	for i := 0; i < len(body); {
		switch body[i] {
		case '{':
			depth++
			i++
		case '}':
			depth--
			if depth == 0 {
				if strings.Contains(fn, `args[0] == "--exit-codes"`) {
					handled["--exit-codes"] = true
				}
				return groups, handled
			}
			i++
		default:
			if depth == 1 && strings.HasPrefix(body[i:], "case ") {
				if colon := strings.Index(body[i:], ":"); colon >= 0 {
					var group []string
					for _, part := range strings.Split(body[i+len("case "):i+colon], ",") {
						part = strings.Trim(strings.TrimSpace(part), `"`)
						group = append(group, part)
						handled[part] = true
					}
					groups = append(groups, group)
					i += colon + 1
					continue
				}
			}
			i++
		}
	}
	return groups, handled
}

// TestUsageDocumentsNewCommands pins the usage Commands: block (app.go:21-40)
// to Run's dispatch switch (app.go:104-149): every documented command must be
// a handled dispatch key, and every dispatch case must appear in usage — so
// adding a command without documenting it (or vice versa) fails. Supplements
// the one-directional hardcoded-list check in TestUsageMentionsAllCommands.
func TestUsageDocumentsNewCommands(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	src, err := os.ReadFile(filepath.Join(filepath.Dir(file), "app.go"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)

	documented := parseUsageCommands(s)
	if len(documented) == 0 {
		t.Fatal("parsed no documented commands from the usage Commands: block")
	}
	groups, handled := runDispatchFrom(s)
	if len(groups) == 0 {
		t.Fatal("parsed no dispatch cases from Run")
	}

	// Every documented command must be a handled dispatch key.
	for cmd := range documented {
		if !handled[cmd] {
			t.Errorf("usage documents %q but Run has no dispatch case for it", cmd)
		}
	}
	// Every dispatch case must appear in usage (at least one label of each
	// case group is the documented command name; -h/--help are the documented
	// help command's aliases).
	for _, group := range groups {
		covered := false
		for _, label := range group {
			if documented[label] {
				covered = true
				break
			}
		}
		if !covered {
			t.Errorf("dispatch case %v is not documented in the usage Commands: block", group)
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

func TestSplitLaunchArgsModelTier(t *testing.T) {
	_, _, _, _, tier, rest := splitLaunchArgs([]string{"--model-tier", "fast", "hi"})
	if tier != "fast" || len(rest) != 1 || rest[0] != "hi" {
		t.Fatalf("got tier=%q rest=%v", tier, rest)
	}
	_, _, _, _, tier, _ = splitLaunchArgs([]string{"--model-tier=cheap", "hi"})
	if tier != "cheap" {
		t.Fatalf("got tier=%q, want cheap", tier)
	}
	// The tier flag must not leak into pass-through args.
	_, _, _, _, tier, rest = splitLaunchArgs([]string{"--model-tier", "fast", "--tools", "read", "x"})
	if tier != "fast" || len(rest) != 3 || rest[0] != "--tools" {
		t.Fatalf("got tier=%q rest=%v", tier, rest)
	}
}

func TestLaunchModelTierConflictExit2(t *testing.T) {
	hermeticLaunchEnv(t)
	code, out := captureRunStderr(t, []string{"chat", "--model-tier", "fast", "--model", "x"})
	if code != 2 {
		t.Fatalf("chat --model-tier fast --model x exit = %d, want 2; stderr: %s", code, out)
	}
	if !strings.Contains(out, "mutually exclusive") {
		t.Fatalf("stderr must mention mutual exclusion, got %q", out)
	}
}

func TestLaunchUnknownTierExit2(t *testing.T) {
	hermeticLaunchEnv(t)
	code, out := captureRunStderr(t, []string{"chat", "--model-tier", "turbo"})
	if code != 2 {
		t.Fatalf("chat --model-tier turbo exit = %d, want 2; stderr: %s", code, out)
	}
	if !strings.Contains(out, "unknown model tier") || !strings.Contains(out, "valid: fast, balanced, cheap") {
		t.Fatalf("stderr must list valid tiers, got %q", out)
	}
}

func TestLaunchUnavailableTierExit2(t *testing.T) {
	hermeticLaunchEnv(t)
	code, out := captureRunStderr(t, []string{"print", "--provider", "deepseek", "--model-tier", "cheap", "hello"})
	if code != 2 {
		t.Fatalf("print --provider deepseek --model-tier cheap exit = %d, want 2; stderr: %s", code, out)
	}
	if !strings.Contains(out, "has no model for tier") || !strings.Contains(out, "available: balanced, fast") {
		t.Fatalf("stderr must list the provider's available tiers, got %q", out)
	}
}

func TestLaunchModelTierEnvWithModelFlagModelWins(t *testing.T) {
	hermeticLaunchEnv(t)
	// (c'): an explicit --model overrides an env-set tier. PI_MODEL_TIER=turbo
	// would be a usage error if the env were applied; with --model winning, the
	// launch proceeds to key resolution (exit 3 in hermetic env) — no error.
	t.Setenv("PI_MODEL_TIER", "turbo")
	if code := Run([]string{"chat", "--model", "openai/gpt-5.6-terra"}); code != 3 {
		t.Fatalf("chat --model with PI_MODEL_TIER=turbo exit = %d, want 3 (missing key; --model must win, env tier ignored)", code)
	}
}

func TestLaunchModelTierNoProviderDefaultsToOpenAI(t *testing.T) {
	hermeticLaunchEnv(t)
	// (d): no --provider given; the tier resolves against the default openai
	// table (which has a fast entry), so the launch reaches key resolution.
	if code := Run([]string{"chat", "--model-tier", "fast"}); code != 3 {
		t.Fatalf("chat --model-tier fast (no provider) exit = %d, want 3 (missing key; tier resolved against default provider)", code)
	}
}

func TestLaunchModelTierResumeRejected(t *testing.T) {
	hermeticLaunchEnv(t)
	code, out := captureRunStderr(t, []string{"resume", "--model-tier", "fast"})
	if code != 2 {
		t.Fatalf("resume --model-tier fast exit = %d, want 2; stderr: %s", code, out)
	}
	if !strings.Contains(out, "resume") || !strings.Contains(out, "--model-tier") {
		t.Fatalf("stderr must name resume and --model-tier, got %q", out)
	}
}

func TestResumeIgnoresModelTierEnv(t *testing.T) {
	hermeticLaunchEnv(t)
	// resume must never read PI_MODEL_TIER: turbo would be a usage error if the
	// env were applied, so exit 3 (missing key) proves it was ignored.
	t.Setenv("PI_MODEL_TIER", "turbo")
	if code := Run([]string{"resume"}); code != 3 {
		t.Fatalf("resume with PI_MODEL_TIER=turbo exit = %d, want 3 (missing key; resume must ignore the env)", code)
	}
}

func TestRunProvidersShowsTiers(t *testing.T) {
	orig := Providers
	defer func() { Providers = orig }()
	Providers = []Provider{
		{Name: "openai", KeyEnv: "OPENAI_API_KEY", PiProvider: "openai", DefaultModel: "openai/gpt-5.6-terra",
			ModelTiers: map[string]string{"fast": "openai/gpt-5.6-mini", "cheap": "openai/gpt-5.1-mini"}},
		{Name: "groq", KeyEnv: "GROQ_API_KEY", PiProvider: "groq", DefaultModel: "groq/llama-3.3-70b-versatile"},
	}
	code, out := captureRunStdout(t, []string{"providers"})
	if code != 0 {
		t.Fatalf("providers exit = %d, want 0", code)
	}
	if !strings.Contains(out, "openai/gpt-5.6-terra\tbalanced,cheap,fast\tOPENAI_API_KEY") {
		t.Fatalf("providers output missing openai tiers column: %q", out)
	}
	if !strings.Contains(out, "groq/llama-3.3-70b-versatile\tbalanced\tGROQ_API_KEY") {
		t.Fatalf("providers output missing groq balanced-only tiers column: %q", out)
	}
}

func TestUsageMentionsCostMode(t *testing.T) {
	if !strings.Contains(usage, "--cost-mode") {
		t.Fatal("usage must document the --cost-mode flag")
	}
	if !strings.Contains(usage, "live-eval") {
		t.Fatal("usage must document the live-eval cost mode")
	}
}

func TestRunCostModeUnknownExit2(t *testing.T) {
	hermeticLaunchEnv(t)
	code, out := captureRunStderr(t, []string{"print", "--cost-mode", "bogus", "hello"})
	if code != 2 {
		t.Fatalf("print --cost-mode bogus exit = %d, want 2; stderr: %s", code, out)
	}
	if !strings.Contains(out, "unknown cost mode") || !strings.Contains(out, "live-eval") {
		t.Fatalf("stderr must name the error and the valid modes, got %q", out)
	}
	// chat form too (no prompt required; validation must fire before any key
	// resolution).
	code, out = captureRunStderr(t, []string{"chat", "--cost-mode", "bogus"})
	if code != 2 || !strings.Contains(out, "unknown cost mode") {
		t.Fatalf("chat --cost-mode bogus exit = %d, want 2; stderr: %s", code, out)
	}
}

func TestRunCostModeAccepted(t *testing.T) {
	hermeticLaunchEnv(t)
	// A valid --cost-mode is inert apart from the ledger mode: the launch
	// proceeds to key resolution (exit 3 in hermetic env).
	if code := Run([]string{"print", "--cost-mode", "live-eval", "hello"}); code != 3 {
		t.Fatalf("print --cost-mode live-eval exit = %d, want 3 (missing key; flag accepted)", code)
	}
	// Absent flag → default mode (the command name), behavior unchanged.
	if code := Run([]string{"chat"}); code != 3 {
		t.Fatalf("chat without --cost-mode exit = %d, want 3 (missing key; default mode unchanged)", code)
	}
}

func TestRunCostModeResumeRejected(t *testing.T) {
	hermeticLaunchEnv(t)
	// --cost-mode is documented for chat/print only; resume rejects it as a
	// usage error (mirroring --model-tier on resume).
	code, out := captureRunStderr(t, []string{"resume", "--cost-mode", "live-eval"})
	if code != 2 || !strings.Contains(out, "--cost-mode") {
		t.Fatalf("resume --cost-mode exit = %d, want 2; stderr: %s", code, out)
	}
}
