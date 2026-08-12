package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Hook events fired by pi-run around its own invocations (the harness-level
// counterpart to agent-internal hooks such as Claude Code pre/post-tool hooks
// or Copilot's errorOccurred). Each event runs zero or more commands from
// .pi/hooks.json at a well-defined point:
//
//	pre-eval   before the DeepEval pytest suite runs
//	post-eval  after the pytest suite finishes (always, even on failure)
//	pre-chat   before pi is launched (chat/print/resume)
const (
	hookEventPreEval  = "pre-eval"
	hookEventPostEval = "post-eval"
	hookEventPreChat  = "pre-chat"
)

// hookEvents lists the supported hook events in definition order.
var hookEvents = []string{hookEventPreEval, hookEventPostEval, hookEventPreChat}

// isHookEvent reports whether event is a known hook event.
func isHookEvent(event string) bool {
	for _, e := range hookEvents {
		if e == event {
			return true
		}
	}
	return false
}

// hookConfig is the parsed .pi/hooks.json document:
//
//	{
//	  "hooks": {
//	    "pre-eval":  [{"cmd": "./scripts/ci/notify.sh start", "timeoutSecs": 60}],
//	    "post-eval": [{"cmd": "./scripts/ci/notify.sh done", "continueOnError": true}],
//	    "pre-chat":  []
//	  }
//	}
type hookConfig struct {
	Hooks map[string][]hookSpec `json:"hooks"`
}

// hookSpec is a single hook command. Cmd is required and runs via `sh -c`
// from the harness repo root. timeoutSecs defaults to 30; a hook that exceeds
// it is killed and counts as a failure (exit code 124). continueOnError
// defaults to false: a failing hook aborts the pi-run invocation with the
// command's exit code unless it is true.
type hookSpec struct {
	Cmd             string `json:"cmd"`
	TimeoutSecs     int    `json:"timeoutSecs"`
	ContinueOnError bool   `json:"continueOnError"`
}

// hooksFilePath returns the hooks config path for the given harness root.
func hooksFilePath(root string) string {
	return filepath.Join(root, ".pi", "hooks.json")
}

// loadHookConfig reads and validates .pi/hooks.json under root. A missing
// file yields an empty config (hooks are entirely optional); a malformed
// file, an unknown event, or a hook without a cmd is an error so CI notices
// a broken hooks file instead of silently skipping commands.
func loadHookConfig(root string) (*hookConfig, error) {
	cfg := &hookConfig{Hooks: map[string][]hookSpec{}}
	data, err := os.ReadFile(hooksFilePath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read hooks config: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse hooks config: %w", err)
	}
	if cfg.Hooks == nil {
		cfg.Hooks = map[string][]hookSpec{}
	}
	for event, hooks := range cfg.Hooks {
		if !isHookEvent(event) {
			return nil, fmt.Errorf("hooks config: unknown event %q (supported: %v)", event, hookEvents)
		}
		for _, h := range hooks {
			if h.Cmd == "" {
				return nil, fmt.Errorf("hooks config: %s: hook with empty cmd", event)
			}
		}
	}
	return cfg, nil
}

// runHooks executes every configured hook for event, in order, streaming each
// command's output. A hook that exits non-zero aborts with a *hookExitError
// (whose code the caller propagates) unless the hook sets continueOnError.
// Returns nil when the event has no hooks or every hook succeeded.
func runHooks(event string) error {
	root := repoRoot()
	cfg, err := loadHookConfig(root)
	if err != nil {
		return err
	}
	hooks := cfg.Hooks[event]
	if len(hooks) == 0 {
		return nil
	}
	fmt.Printf("== pi-run hook %s (%d command(s)) ==\n", event, len(hooks))
	for i, h := range hooks {
		fmt.Printf("> [%d/%d] %s\n", i+1, len(hooks), h.Cmd)
		code, err := execHookCmd(root, h.Cmd, hookTimeout(h), os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
		if code == 0 {
			continue
		}
		if h.ContinueOnError {
			fmt.Printf("pi-run: hook %s[%d] exited %d; continuing (continueOnError)\n", event, i+1, code)
			continue
		}
		return &hookExitError{event: event, index: i + 1, code: code}
	}
	return nil
}

// hookTimeout resolves a hook's timeoutSecs (default 30s).
func hookTimeout(h hookSpec) time.Duration {
	secs := h.TimeoutSecs
	if secs <= 0 {
		secs = 30
	}
	return time.Duration(secs) * time.Second
}

// hookExitError reports a hook command that exited non-zero and whose
// continueOnError is false. The caller propagates code as its own exit code.
type hookExitError struct {
	event string
	index int
	code  int
}

func (e *hookExitError) Error() string {
	return fmt.Sprintf("pi-run: hook %s[%d] failed with exit code %d", e.event, e.index, e.code)
}

// hookExitCode maps a runHooks error to the exit code the caller should
// return: the failing command's exit code capped to the valid 0-255 range,
// or 1 for any other error.
func hookExitCode(err error) int {
	he, ok := errors.AsType[*hookExitError](err)
	if !ok {
		return 1
	}
	code := he.code
	if code < 0 {
		return 1
	}
	if code > 255 {
		return 255
	}
	return code
}

// execHookCmd runs cmd via `sh -c` with dir as the working directory,
// streaming to stdout/stderr, and kills the process when timeout expires
// (reported as exit code 124, mirroring the `timeout` CLI convention). The
// command is killed with SIGKILL on expiry so a hung hook cannot block
// pi-run forever. Returns the command's exit code, or an error only when the
// command could not be started.
func execHookCmd(dir, cmd string, timeout time.Duration, stdout, stderr io.Writer) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c := exec.CommandContext(ctx, "sh", "-c", cmd)
	c.Dir = dir
	c.Stdin = os.Stdin
	c.Stdout = stdout
	c.Stderr = stderr
	if err := c.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return 124, nil
		}
		if ee, ok := errors.AsType[*exec.ExitError](err); ok {
			return ee.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

const hooksUsage = `Usage: pi-run hooks <list|run <event>>

Run hook commands configured in .pi/hooks.json around pi-run invocations.

Commands:
  list              List the hooks configured in .pi/hooks.json
  run <event>       Run the hooks for <event> now (pre-eval, post-eval, pre-chat)
`

// runHooksCmd implements `pi-run hooks list|run <event>`.
func runHooksCmd(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, hooksUsage)
		return 2
	}
	switch args[0] {
	case "list":
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "pi-run: hooks: list takes no arguments\n\n%s", hooksUsage)
			return 2
		}
		return hooksListCmd(repoRoot())
	case "run":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "pi-run: hooks: run requires an event (one of %v)\n\n%s", hookEvents, hooksUsage)
			return 2
		}
		if len(args) > 2 {
			fmt.Fprintf(os.Stderr, "pi-run: hooks: run takes exactly one event\n\n%s", hooksUsage)
			return 2
		}
		event := args[1]
		if !isHookEvent(event) {
			fmt.Fprintf(os.Stderr, "pi-run: hooks: unknown event %q (one of %v)\n\n%s", event, hookEvents, hooksUsage)
			return 2
		}
		if err := runHooks(event); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return hookExitCode(err)
		}
		return 0
	default:
		fmt.Fprintf(os.Stderr, "pi-run: hooks: unknown subcommand %q\n\n%s", args[0], hooksUsage)
		return 2
	}
}

// hooksListCmd prints the hooks configured under root, per event.
func hooksListCmd(root string) int {
	cfg, err := loadHookConfig(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("hooks: %s\n", hooksFilePath(root))
	if len(cfg.Hooks) == 0 {
		fmt.Println("no hooks configured")
		return 0
	}
	for _, event := range hookEvents {
		hooks := cfg.Hooks[event]
		if len(hooks) == 0 {
			fmt.Printf("  %s: (none)\n", event)
			continue
		}
		for i, h := range hooks {
			extra := "timeout default 30s"
			if h.TimeoutSecs > 0 {
				extra = fmt.Sprintf("timeout %ds", h.TimeoutSecs)
			}
			if h.ContinueOnError {
				extra += ", continueOnError"
			}
			fmt.Printf("  %s[%d]: %s (%s)\n", event, i+1, h.Cmd, extra)
		}
	}
	return 0
}
