package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeHooksConfig writes a hooks config file under root/.pi/hooks.json.
func writeHooksConfig(t *testing.T, root, content string) {
	t.Helper()
	dir := filepath.Join(root, ".pi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hooks.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const sampleHooksJSON = `{
  "hooks": {
    "pre-eval":  [{"cmd": "./scripts/notify.sh start", "timeoutSecs": 60}],
    "post-eval": [{"cmd": "./scripts/notify.sh done", "continueOnError": true}],
    "pre-chat":  [{"cmd": "git status --porcelain"}]
  }
}`

func TestLoadHookConfigValid(t *testing.T) {
	root := t.TempDir()
	writeHooksConfig(t, root, sampleHooksJSON)
	cfg, err := loadHookConfig(root)
	if err != nil {
		t.Fatalf("loadHookConfig: %v", err)
	}
	if len(cfg.Hooks) != 3 {
		t.Fatalf("got %d events, want 3", len(cfg.Hooks))
	}
	pre := cfg.Hooks["pre-eval"]
	if len(pre) != 1 || pre[0].Cmd != "./scripts/notify.sh start" || pre[0].TimeoutSecs != 60 || pre[0].ContinueOnError {
		t.Fatalf("unexpected pre-eval hooks: %+v", pre)
	}
	post := cfg.Hooks["post-eval"]
	if len(post) != 1 || !post[0].ContinueOnError || post[0].TimeoutSecs != 0 {
		t.Fatalf("unexpected post-eval hooks: %+v", post)
	}
}

func TestLoadHookConfigMissingFile(t *testing.T) {
	cfg, err := loadHookConfig(t.TempDir())
	if err != nil {
		t.Fatalf("missing hooks file must not error, got %v", err)
	}
	if len(cfg.Hooks) != 0 {
		t.Fatalf("missing hooks file should yield an empty config, got %+v", cfg.Hooks)
	}
}

func TestLoadHookConfigMalformed(t *testing.T) {
	root := t.TempDir()
	writeHooksConfig(t, root, `{"hooks": {`)
	if _, err := loadHookConfig(root); err == nil {
		t.Fatal("malformed JSON must error")
	}
}

func TestLoadHookConfigUnknownEvent(t *testing.T) {
	root := t.TempDir()
	writeHooksConfig(t, root, `{"hooks": {"pre-tool": [{"cmd": "true"}]}}`)
	_, err := loadHookConfig(root)
	if err == nil || !strings.Contains(err.Error(), "pre-tool") {
		t.Fatalf("unknown event must error mentioning the event, got %v", err)
	}
}

func TestLoadHookConfigEmptyCmd(t *testing.T) {
	root := t.TempDir()
	writeHooksConfig(t, root, `{"hooks": {"pre-eval": [{"timeoutSecs": 5}]}}`)
	if _, err := loadHookConfig(root); err == nil {
		t.Fatal("hook without a cmd must error")
	}
}

func TestExecHookCmdEcho(t *testing.T) {
	var buf bytes.Buffer
	code, err := execHookCmd(t.TempDir(), "echo hello-hook", 5*time.Second, &buf, &buf)
	if err != nil {
		t.Fatalf("execHookCmd: %v", err)
	}
	if code != 0 {
		t.Fatalf("echo exit = %d, want 0", code)
	}
	if !strings.Contains(buf.String(), "hello-hook") {
		t.Fatalf("output %q must contain command output", buf.String())
	}
}

func TestExecHookCmdFailureCode(t *testing.T) {
	code, err := execHookCmd(t.TempDir(), "exit 7", 5*time.Second, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("execHookCmd: %v", err)
	}
	if code != 7 {
		t.Fatalf("exit 7 returned %d, want 7", code)
	}
}

func TestExecHookCmdTimeoutKillsHungHook(t *testing.T) {
	// A shell builtin busy-loop: sh itself hangs (no child process to orphan),
	// so the deadline kill is deterministic and hermetic.
	start := time.Now()
	code, err := execHookCmd(t.TempDir(), "while :; do :; done", time.Second, &bytes.Buffer{}, &bytes.Buffer{})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("execHookCmd: %v", err)
	}
	if code != 124 {
		t.Fatalf("hung hook returned %d, want 124 (timeout)", code)
	}
	if elapsed > 10*time.Second {
		t.Fatalf("hung hook took %s, want ~1s timeout", elapsed)
	}
}

func TestRunHooksContinuesOnError(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	// First hook fails but continueOnError; second writes a marker file in the
	// hooks working directory (the harness root) proving execution continued.
	writeHooksConfig(t, root, `{"hooks": {"pre-eval": [
		{"cmd": "exit 3", "continueOnError": true},
		{"cmd": "printf ran > marker.txt"}
	]}}`)
	if err := runHooks("pre-eval"); err != nil {
		t.Fatalf("runHooks with continueOnError must not fail, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "marker.txt")); err != nil {
		t.Fatalf("second hook did not run after continueOnError: %v", err)
	}
}

func TestRunHooksAbortsWithoutContinueOnError(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	writeHooksConfig(t, root, `{"hooks": {"pre-eval": [{"cmd": "exit 7"}]}}`)
	err := runHooks("pre-eval")
	if err == nil {
		t.Fatal("failing hook must abort runHooks")
	}
	if code := hookExitCode(err); code != 7 {
		t.Fatalf("hookExitCode = %d, want 7 (command exit code)", code)
	}
}

func TestRunHooksNoConfigNoop(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	if err := runHooks("pre-eval"); err != nil {
		t.Fatalf("no config must be a no-op, got %v", err)
	}
}

func TestRunHooksUnknownEventNoop(t *testing.T) {
	// An event with no configured hooks is a no-op even when the file exists.
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	writeHooksConfig(t, root, `{"hooks": {"pre-eval": [{"cmd": "true"}]}}`)
	if err := runHooks("post-eval"); err != nil {
		t.Fatalf("event without hooks must be a no-op, got %v", err)
	}
}

func TestHookExitCodeCapped(t *testing.T) {
	if code := hookExitCode(&hookExitError{event: "pre-eval", index: 1, code: 300}); code != 255 {
		t.Fatalf("code 300 should cap to 255, got %d", code)
	}
	if code := hookExitCode(&hookExitError{event: "pre-eval", index: 1, code: -1}); code != 1 {
		t.Fatalf("negative code should fall back to 1, got %d", code)
	}
	if code := hookExitCode(errors.New("boom")); code != 1 {
		t.Fatalf("non-hook error should map to 1, got %d", code)
	}
}

func TestRunHooksCmdUsage(t *testing.T) {
	if code, _ := captureRunStderr(t, []string{"hooks"}); code != 2 {
		t.Fatalf("bare hooks exit = %d, want 2 (usage)", code)
	}
	if code, _ := captureRunStderr(t, []string{"hooks", "run"}); code != 2 {
		t.Fatalf("hooks run without event exit = %d, want 2", code)
	}
	if code, out := captureRunStderr(t, []string{"hooks", "run", "pre-tool"}); code != 2 || !strings.Contains(out, "unknown event") {
		t.Fatalf("hooks run <bad-event> exit = %d (stderr %q), want 2 with unknown-event error", code, out)
	}
	if code, _ := captureRunStderr(t, []string{"hooks", "frobnicate"}); code != 2 {
		t.Fatalf("unknown hooks subcommand exit = %d, want 2", code)
	}
}

func TestRunHooksCmdListNoConfig(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	code, out := captureRunStdout(t, []string{"hooks", "list"})
	if code != 0 {
		t.Fatalf("hooks list (no config) exit = %d, want 0", code)
	}
	if !strings.Contains(out, "no hooks configured") {
		t.Fatalf("hooks list output %q must say no hooks configured", out)
	}
}

func TestRunHooksCmdListWithConfig(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	writeHooksConfig(t, root, sampleHooksJSON)
	code, out := captureRunStdout(t, []string{"hooks", "list"})
	if code != 0 {
		t.Fatalf("hooks list exit = %d, want 0", code)
	}
	for _, want := range []string{
		"hooks.json", "pre-eval[1]: ./scripts/notify.sh start (timeout 60s)",
		"post-eval[1]: ./scripts/notify.sh done (timeout default 30s, continueOnError)",
		"pre-chat[1]: git status --porcelain",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("hooks list output missing %q; got:\n%s", want, out)
		}
	}
}

func TestRunHooksCmdRunEcho(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	writeHooksConfig(t, root, `{"hooks": {"pre-eval": [{"cmd": "echo from-hook"}]}}`)
	code, out := captureRunStdout(t, []string{"hooks", "run", "pre-eval"})
	if code != 0 {
		t.Fatalf("hooks run pre-eval exit = %d, want 0", code)
	}
	if !strings.Contains(out, "from-hook") {
		t.Fatalf("hooks run output %q must stream hook stdout", out)
	}
}

func TestRunHooksCmdRunFailurePropagatesCode(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	writeHooksConfig(t, root, `{"hooks": {"pre-eval": [{"cmd": "exit 9"}]}}`)
	code, _ := captureRunStderr(t, []string{"hooks", "run", "pre-eval"})
	if code != 9 {
		t.Fatalf("hooks run with failing hook exit = %d, want 9", code)
	}
}

func TestRunHooksCmdRunContinueOnErrorExitsZero(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	writeHooksConfig(t, root, `{"hooks": {"post-eval": [{"cmd": "exit 9", "continueOnError": true}]}}`)
	code, _ := captureRunStderr(t, []string{"hooks", "run", "post-eval"})
	if code != 0 {
		t.Fatalf("hooks run with continueOnError exit = %d, want 0", code)
	}
}

func TestRunHooksCmdListMalformedConfig(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	writeHooksConfig(t, root, `not json`)
	code, out := captureRunStderr(t, []string{"hooks", "list"})
	if code != 1 {
		t.Fatalf("hooks list with malformed config exit = %d, want 1", code)
	}
	if !strings.Contains(out, "parse hooks config") {
		t.Fatalf("hooks list stderr %q must report parse failure", out)
	}
}

func TestUsageMentionsHooks(t *testing.T) {
	if !strings.Contains(usage, "hooks") {
		t.Fatal("usage must document the hooks command")
	}
}
