package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const Version = "0.1.0"

const usage = `pi-run — Pi harness runtime CLI

Usage:
  pi-run <command> [flags] [args...]

Commands:
  chat          Launch Pi interactively (default provider: openai)
  print         Run Pi in print mode: pi -p --no-session "<prompt>"
  eval          Run the DeepEval pytest suite (--quick for smoke subset)
  config-check  Run deterministic harness checks (no keys, no network)
  doctor        Report harness health (node, pi, keys, models)
  setup         Create eval/.venv, install deps, refresh model catalogs
  install       Build the binary into bin/ and symlink ~/bin/pi-run
  clean         Remove eval/.venv and pytest caches
  version       Print version
  help          Show this help

chat/print flags:
  --provider <openai|openrouter|deepseek>  Provider (env PI_PROVIDER; default openai)
  --model <id>                             Override the per-provider default model

Everything else is passed through to pi unchanged (use -- to escape a message
that starts with a dash, e.g. pi-run print -- "-weird prompt").
`

// Run is the CLI entry point. It returns the process exit code.
func Run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	switch args[0] {
	case "help", "--help", "-h":
		fmt.Print(usage)
		return 0
	case "version":
		fmt.Printf("pi-run %s\n", Version)
		return 0
	case "chat", "print":
		return runLaunch(args[0], args[1:])
	case "eval":
		quick := len(args) > 1 && args[1] == "--quick"
		return runEval(quick)
	case "config-check":
		return runConfigCheck()
	case "doctor":
		return runDoctor()
	case "setup":
		return runSetup()
	case "install":
		return runInstall()
	case "clean":
		return runClean()
	default:
		fmt.Fprintf(os.Stderr, "pi-run: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

// runLaunch handles `chat` and `print`.
func runLaunch(mode string, args []string) int {
	providerFlag, modelFlag, rest := splitLaunchArgs(args)

	// Validate the prompt BEFORE touching keys (usage error wins over key error).
	if mode == "print" && len(rest) == 0 {
		fmt.Fprint(os.Stderr, "pi-run: print requires a prompt\n\n"+usage)
		return 2
	}

	p, err := ResolveProvider(providerFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	key, err := resolveSecret(p.KeyEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "no %s available: export it, or run `bw unlock` then `pi-run doctor`\n", p.KeyEnv)
		return 3
	}

	model := modelFlag
	if model == "" {
		model = p.DefaultModel
	}

	nodeVersion := os.Getenv("PI_NODE_VERSION")
	if nodeVersion == "" {
		nodeVersion = "v22.19.0"
	}
	code, err := execPi(nodeVersion, piArgs(p, model, mode, rest), []string{p.KeyEnv + "=" + key})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return code
}

// splitLaunchArgs separates pi-run's own flags (--provider/--model) from
// pass-through args. Everything else is kept in order; "--" ends flag parsing
// and its tail is also kept (allowing messages that start with a dash).
func splitLaunchArgs(args []string) (provider, model string, rest []string) {
	i := 0
	for i < len(args) {
		a := args[i]
		switch {
		case a == "--":
			rest = append(rest, args[i+1:]...)
			return
		case a == "--provider" && i+1 < len(args):
			provider = args[i+1]
			i += 2
		case strings.HasPrefix(a, "--provider="):
			provider = strings.TrimPrefix(a, "--provider=")
			i++
		case a == "--model" && i+1 < len(args):
			model = args[i+1]
			i += 2
		case strings.HasPrefix(a, "--model="):
			model = strings.TrimPrefix(a, "--model=")
			i++
		default:
			rest = append(rest, a)
			i++
		}
	}
	return
}

// runInstall builds the binary and symlinks it into ~/bin.
func runInstall() int {
	root := repoRoot()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	target := filepath.Join(binDir, "pi-run")
	code, err := runCmd("go", []string{"build", "-o", target, "./cmd/pi-run"}, root)
	if err != nil || code != 0 {
		return code
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	link := filepath.Join(home, "bin", "pi-run")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_ = os.Remove(link)
	if err := os.Symlink(target, link); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("installed %s -> %s\n", link, target)
	return 0
}

// runClean removes eval/.venv and pytest caches.
func runClean() int {
	root := repoRoot()
	for _, p := range []string{
		filepath.Join(root, "eval", ".venv"),
		filepath.Join(root, "eval", "__pycache__"),
		filepath.Join(root, "eval", "tests", "__pycache__"),
		filepath.Join(root, "eval", ".pytest_cache"),
	} {
		if err := os.RemoveAll(p); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	return 0
}
