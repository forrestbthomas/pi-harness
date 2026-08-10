package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Version is the build-time version, injected via -ldflags
// "-X github.com/forrestthomas1/pi-harness/internal/cli.Version=<tag>".
// Defaults to "dev" for local builds.
var Version = "dev"

const usage = `pi-run — Pi harness runtime CLI

Usage:
  pi-run <command> [flags] [args...]

Commands:
  chat          Launch Pi interactively (default provider: openai)
  print         Run Pi in print mode: pi -p --no-session "<prompt>"
  resume        Continue the most recent session (pi --continue)
  eval          Run the DeepEval pytest suite (--quick for smoke subset)
  config-check  Run deterministic harness checks (no keys, no network)
  doctor        Report harness health (node, pi, keys, models)
  setup         Create eval/.venv, install deps, refresh model catalogs
  install       Build the binary into bin/ and symlink pi-run onto your PATH
  clean         Remove eval/.venv and pytest caches
  providers     List configured providers and default models
  version       Print version
  help          Show this help
  --exit-codes  Print the exit-code table

Exit codes: 0 ok · 1 generic · 2 usage · 3 missing key · 4 node/pi missing · 5 eval venv missing

chat/print flags:
  --provider <name>                        Provider (see 'pi-run providers'; env PI_PROVIDER; default openai)
  --model <id>                             Override the per-provider default model

Eval flags:
  --quick                                  Run the deterministic smoke subset
  --                                       Pass remaining arguments directly to pytest

Everything else is passed through to pi unchanged (use -- to escape a message
that starts with a dash, e.g. pi-run print -- "-weird prompt").
`

const exitCodesText = `Exit codes:
  0  ok
  1  generic error
  2  usage error
  3  missing API key
  4  node/pi not found
  5  eval venv missing
`

// Run is the CLI entry point. It returns the process exit code.
func Run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	if len(args) == 1 && args[0] == "--exit-codes" {
		fmt.Print(exitCodesText)
		return 0
	}
	switch args[0] {
	case "help", "--help", "-h":
		fmt.Print(usage)
		return 0
	case "version":
		fmt.Printf("pi-run %s\n", Version)
		return 0
	case "chat", "print", "resume":
		return runLaunch(args[0], args[1:])
	case "eval":
		return runEval(args[1:])
	case "config-check":
		return runConfigCheck()
	case "doctor":
		return runDoctor()
	case "setup":
		return runSetup()
	case "install":
		force := false
		for _, arg := range args[1:] {
			switch arg {
			case "--force":
				force = true
			case "--help", "-h":
				fmt.Println("Usage: pi-run install [--force]\n\nBuild pi-run and symlink it onto your PATH.\n--force overwrites an existing ~/bin/pi-run entry.")
				return 0
			default:
				fmt.Fprintf(os.Stderr, "pi-run: install: unknown flag %q\n", arg)
				return 2
			}
		}
		return runInstall(force)
	case "clean":
		return runClean()
	case "providers":
		return runProviders()
	default:
		fmt.Fprintf(os.Stderr, "pi-run: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

// runLaunch handles `chat`, `print`, and `resume`.
func runLaunch(mode string, args []string) int {
	providerFlag, modelFlag, rest := splitLaunchArgs(args)

	// Validate the prompt BEFORE touching keys (usage error wins over key error).
	if mode == "print" && len(rest) == 0 {
		fmt.Fprintf(os.Stderr, "pi-run: print: requires a prompt\n\n%s", usage)
		return 2
	}

	p, err := ResolveProvider(providerFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: %s: %v\n", mode, err)
		return 2
	}

	key, err := resolveSecret(p.KeyEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: %s: no %s available: export it, or check your secret manager (`pi-run doctor`)\n", mode, p.KeyEnv)
		return 3
	}

	model := modelFlag
	if model == "" {
		model = p.DefaultModel
	}

	nodeVersion, err := resolveNodeVersion(os.Getenv("HOME"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: %s: %v\n", mode, err)
		return 4
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

// runInstall builds the binary and symlinks it onto the user's PATH.
func runInstall(force bool) int {
	root := repoRoot()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	target := filepath.Join(binDir, "pi-run")
	ldflags := "-X github.com/forrestthomas1/pi-harness/internal/cli.Version=" + buildVersion()
	code, err := runCmd("go", []string{"build", "-ldflags", ldflags, "-o", target, "./cmd/pi-run"}, root)
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
	if force {
		if _, err := os.Lstat(link); err == nil {
			fmt.Fprintf(os.Stderr, "pi-run: install: warning: --force overwriting existing %s\n", link)
		}
	}
	if err := installLink(target, link, force); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("installed %s -> %s\n", link, target)
	return 0
}

// installLink creates link -> target without overwriting unrelated user files
// unless force is true. An existing link to the same target is safe to replace.
func installLink(target, link string, force bool) error {
	info, err := os.Lstat(link)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("inspect existing %s: %w", link, err)
		}
		return os.Symlink(target, link)
	}

	if !force {
		if info.Mode()&os.ModeSymlink != 0 {
			existing, err := os.Readlink(link)
			if err != nil {
				return fmt.Errorf("inspect existing symlink %s: %w", link, err)
			}
			existingTarget := existing
			if !filepath.IsAbs(existingTarget) {
				existingTarget = filepath.Join(filepath.Dir(link), existingTarget)
			}
			if filepath.Clean(existingTarget) != filepath.Clean(target) {
				return fmt.Errorf("refusing to overwrite existing symlink %s -> %s; move or remove it, or re-run with --force", link, existing)
			}
		} else {
			return fmt.Errorf("refusing to overwrite existing non-symlink %s; move or remove it, or re-run with --force", link)
		}
	}

	if err := os.Remove(link); err != nil {
		return fmt.Errorf("remove existing %s: %w", link, err)
	}
	return os.Symlink(target, link)
}

// buildVersion returns the version to stamp into the binary: "dev" for local
// builds. Release workflows pass the tag via -ldflags directly (see
// scripts/build-release.sh); this default keeps `pi-run install` simple.
func buildVersion() string {
	return "dev"
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

// runProviders lists configured providers and their default models.
func runProviders() int {
	for _, p := range Providers {
		line := fmt.Sprintf("%s\t%s\t%s", p.Name, p.DefaultModel, p.KeyEnv)
		if p.BaseURL != "" {
			line += "\t" + p.BaseURL
		}
		fmt.Println(line)
	}
	return 0
}
