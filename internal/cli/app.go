package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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
  cost          Aggregate real spend from Pi session files (--json/--since/--reset)
  eval          Run the DeepEval pytest suite (--quick for smoke subset) or the Docker benchmark runner
  ci-benchmark  Run the benchmark suite against 2+ providers and gate on a scorecard
  config-check  Run deterministic harness checks (no keys, no network)
  project-understand
                Generate deterministic project-understanding docs (product.md, tech.md, structure.md)
  doctor        Report harness health (node, pi, keys, models)
  setup         Create eval/.venv, install deps, refresh model catalogs
  install       Build the binary into bin/ and symlink pi-run onto your PATH
  clean         Remove eval/.venv and pytest caches
  providers     List configured providers and default models
  mcp-server    Serve a read-only MCP server over stdio (providers, cost, benchmark_dry_run)
  hooks         List or run .pi/hooks.json hooks (pre-eval, post-eval, pre-chat)
  version       Print version
  help          Show this help
  --exit-codes  Print the exit-code table

Exit codes: 0 ok · 1 generic · 2 usage · 3 missing key · 4 node/pi missing · 5 eval venv missing · 6 budget exceeded · 7 docker unavailable · 8 scorecard gate failed

chat/print flags:
  --provider <name>                        Provider (see 'pi-run providers'; env PI_PROVIDER; default openai)
  --model <id>                             Override the per-provider default model
  --model-tier <fast|balanced|cheap>       Within-provider model tier (env PI_MODEL_TIER; flag wins over env; balanced = provider default model; mutually exclusive with --model; not supported for resume)
  --max-budget-usd <n>                     Refuse to launch when cumulative spend >= <n> USD (env PI_MAX_BUDGET_USD)
  --cost-mode <mode>                       Tag this run's ledger spend with <mode>: chat|print|resume|backfill|benchmark|live-eval (default: the command name; lets CI mark live-eval runs)
  --permission-mode <mode>                 Permission tier: default|plan|acceptEdits|bypassPermissions (env PI_PERMISSION_MODE). plan = read-only tools (--tools read,grep,find,ls); acceptEdits = Pi defaults (edits allowed); bypassPermissions = --approve (trust project-local files)
  --read-only                              Alias for --permission-mode plan (read-only session)
  PI_OTLP_ENDPOINT                         Optional OTLP/HTTP collector (e.g. http://localhost:4318); exports one GenAI invoke_agent span per launch to <endpoint>/v1/traces (best-effort, never changes the exit code)

cost flags:
  --json                                   Machine-readable JSON output
  --since <date>                           Only count sessions modified at/after <date> (YYYY-MM-DD or RFC3339)
  --reset                                  Archive the spend ledger and start a fresh budget period

Eval flags:
  --quick                                  Run the deterministic smoke subset
  --benchmark [name]                       Run Docker-isolated benchmark tasks (all when name omitted)
  --benchmark-dry-run                      Validate benchmark task format only (no Docker, no keys)
  --provider <name> / --model <id>         Agent provider/model for --benchmark
  --                                       Pass remaining arguments directly to pytest

ci-benchmark flags:
  --providers <a,b> / --models <m1,m2>     Providers (>=2) and optional per-provider model overrides
  --fail-below <rate>                      Fail (exit 8) if any provider pass rate < rate
  --max-budget-usd <n>                     Fail (exit 6) if total run cost >= n (env PI_MAX_BUDGET_USD)
  --baseline <path> / --baseline-tolerance <n>
                                           Diff pass rates against a prior scorecard/run (default tolerance 0.05)
  --runs <n>                               Repeat each provider suite n times; gate on median pass rate
  --quick-profile                          Cap per-task agent timeout at 60s (cheap smoke run)

project-understand flags:
  --out <dir>   Write docs into <dir> (default <root>/docs/understand)

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
  6  budget exceeded
  7  docker unavailable (benchmarks)
  8  scorecard gate failed (ci-benchmark)
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
	case "cost":
		return runCost(args[1:])
	case "eval":
		return runEval(args[1:])
	case "ci-benchmark":
		return runScorecard(args[1:])
	case "config-check":
		return runConfigCheck()
	case "project-understand":
		return runProjectUnderstand(args[1:])
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
	case "mcp-server":
		return runMCPServer()
	case "hooks":
		return runHooksCmd(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "pi-run: unknown command %q\n\n%s", args[0], usage)
		return 2
	}
}

// runLaunch handles `chat`, `print`, and `resume`.
func runLaunch(mode string, args []string) int {
	costModeFlag, launchArgs := splitCostModeFlag(args)
	providerFlag, modelFlag, budgetFlag, permissionModeFlag, tierFlag, rest := splitLaunchArgs(launchArgs)

	// Validate the prompt and permission mode BEFORE touching keys (usage
	// error wins over key error).
	if mode == "print" && len(rest) == 0 {
		fmt.Fprintf(os.Stderr, "pi-run: print: requires a prompt\n\n%s", usage)
		return 2
	}

	// resume continues a session with the model it was launched with: the
	// --model-tier flag is a usage error, and PI_MODEL_TIER is never read for
	// resume (the env must not silently swap a resumed session's model).
	if mode == "resume" && tierFlag != "" {
		fmt.Fprintf(os.Stderr, "pi-run: resume: --model-tier cannot be used with resume (a resumed session continues with its original model)\n")
		return 2
	}
	// --cost-mode tags a launch's ledger spend for CI attribution and is
	// documented for chat/print only: resume continues an existing session and
	// is not a taggable launch.
	if mode == "resume" && costModeFlag != "" {
		fmt.Fprintf(os.Stderr, "pi-run: resume: --cost-mode cannot be used with resume\n")
		return 2
	}
	costMode, err := resolveCostMode(mode, costModeFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: %s: %v\n", mode, err)
		return 2
	}

	permissionMode, err := resolvePermissionMode(permissionModeFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: %s: %v\n", mode, err)
		return 2
	}

	capUSD, err := resolveBudgetCap(budgetFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: %s: %v\n", mode, err)
		return 2
	}

	// (c) --model-tier and --model are mutually exclusive as explicit flags.
	// Rejected (rather than "--model wins") because silently ignoring an
	// explicit flag is the same surprise class as fallback. Checked before
	// provider resolution so usage errors always win over key/node errors.
	if tierFlag != "" && modelFlag != "" {
		fmt.Fprintf(os.Stderr, "pi-run: %s: --model-tier and --model are mutually exclusive; pick one\n", mode)
		return 2
	}
	// Effective tier: the flag wins; PI_MODEL_TIER is a default consulted only
	// when the flag is absent, never against an explicit --model (c': an
	// explicit --model overrides the env tier) and never for resume.
	tier := tierFlag
	if tier == "" && mode != "resume" && modelFlag == "" {
		tier = os.Getenv("PI_MODEL_TIER")
	}

	p, err := ResolveProvider(providerFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: %s: %v\n", mode, err)
		return 2
	}

	model, err := resolveLaunchModel(p, tier, modelFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: %s: %v\n", mode, err)
		return 2
	}

	// Pre-flight budget check: refuse BEFORE resolving keys or launching pi.
	root := repoRoot()
	preSpend, err := currentSpend(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: %s: budget check failed: %v\n", mode, err)
		return 1
	}
	if capUSD > 0 && preSpend >= capUSD {
		fmt.Fprintf(os.Stderr,
			"pi-run: %s: budget exceeded: $%.6f already spent (cap $%.6f) — raise --max-budget-usd, or start a fresh period with `pi-run cost --reset`\n",
			mode, preSpend, capUSD)
		return exitBudgetExceeded
	}

	key, err := resolveSecret(p.KeyEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: %s: no %s available: export it, or check your secret manager (`pi-run doctor`)\n", mode, p.KeyEnv)
		return 3
	}

	nodeVersion, err := resolveNodeVersion(os.Getenv("HOME"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: %s: %v\n", mode, err)
		return 4
	}

	// Hooks: pre-chat fires before launching pi. A failing hook (without
	// continueOnError) aborts the launch with the hook's exit code.
	if err := runHooks(hookEventPreChat); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return hookExitCode(err)
	}

	runStart := time.Now()
	code, err := execPi(nodeVersion, piArgs(p, model, mode, rest, capUSD > 0, permissionMode), launchEnv(p, key))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// Record this run's spend in the ledger (best-effort; never fatal). The
	// mode is the --cost-mode value when given, else the command name.
	postSpend, rerr := recordRunSpend(root, runStart, costMode, p.Name, model, preSpend)
	if rerr != nil {
		fmt.Fprintf(os.Stderr, "pi-run: %s: warning: could not record spend: %v\n", mode, rerr)
	} else if capUSD > 0 && postSpend > capUSD {
		fmt.Fprintf(os.Stderr,
			"pi-run: %s: warning: cumulative spend $%.6f now exceeds --max-budget-usd $%.6f\n",
			mode, postSpend, capUSD)
	}

	// Best-effort OTel GenAI telemetry (env-gated; never affects the exit code).
	maybeExportOTLPSpan(p, model, mode, runStart, time.Now(), code)
	return code
}

// splitLaunchArgs separates pi-run's own flags (--provider/--model/
// --model-tier/--max-budget-usd/--permission-mode/--read-only) from
// pass-through args.
// Everything else is kept in order; "--" ends flag parsing and its tail is
// also kept (allowing messages that start with a dash).
func splitLaunchArgs(args []string) (provider, model, budget, permissionMode, tier string, rest []string) {
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
		case a == "--model-tier" && i+1 < len(args):
			tier = args[i+1]
			i += 2
		case strings.HasPrefix(a, "--model-tier="):
			tier = strings.TrimPrefix(a, "--model-tier=")
			i++
		case a == "--max-budget-usd" && i+1 < len(args):
			budget = args[i+1]
			i += 2
		case strings.HasPrefix(a, "--max-budget-usd="):
			budget = strings.TrimPrefix(a, "--max-budget-usd=")
			i++
		case a == "--permission-mode" && i+1 < len(args):
			permissionMode = args[i+1]
			i += 2
		case strings.HasPrefix(a, "--permission-mode="):
			permissionMode = strings.TrimPrefix(a, "--permission-mode=")
			i++
		case a == "--read-only":
			permissionMode = "plan" // alias for --permission-mode plan
			i++
		default:
			rest = append(rest, a)
			i++
		}
	}
	return
}

// splitCostModeFlag strips pi-run's --cost-mode flag from the launch args
// before splitLaunchArgs runs, so the value reaches runLaunch while the flag
// never leaks into pi's pass-through args. "--" ends flag parsing and its
// tail is preserved verbatim, matching splitLaunchArgs. An absent flag yields
// an empty mode (the caller defaults to the command name).
func splitCostModeFlag(args []string) (mode string, rest []string) {
	i := 0
	for i < len(args) {
		a := args[i]
		switch {
		case a == "--":
			rest = append(rest, args[i:]...)
			return
		case a == "--cost-mode" && i+1 < len(args):
			mode = args[i+1]
			i += 2
		case strings.HasPrefix(a, "--cost-mode="):
			mode = strings.TrimPrefix(a, "--cost-mode=")
			i++
		default:
			rest = append(rest, a)
			i++
		}
	}
	return
}

// permissionModes is the validated set of harness-level permission tiers. It
// mirrors the common Claude Code permission-mode set. The value is NOT passed
// to pi unchanged: piArgs maps each tier to Pi's real flags (plan ->
// --tools read,grep,find,ls; bypassPermissions -> --approve; default/
// acceptEdits -> no flag), because Pi has no --permission-mode flag.
var permissionModes = map[string]bool{
	"default":           true,
	"plan":              true,
	"acceptEdits":       true,
	"bypassPermissions": true,
}

// costModes is the validated set of ledger spend modes accepted by
// --cost-mode. It mirrors the ledgerEntry mode values: the chat/print/resume
// launch commands, "backfill" (pre-ledger spend), "benchmark" (ci-benchmark
// runs), and "live-eval" (CI-tagged live agent-eval runs).
var costModes = map[string]bool{
	"chat":      true,
	"print":     true,
	"resume":    true,
	"backfill":  true,
	"benchmark": true,
	"live-eval": true,
}

// resolveCostMode returns the effective ledger spend mode for a launch: the
// --cost-mode flag wins; when absent the launch command's own name is used,
// which is today's behavior unchanged. The flag is inert except for the mode
// string passed to recordRunSpend. Unknown values are usage errors.
func resolveCostMode(command, flagVal string) (string, error) {
	v := flagVal
	if v == "" {
		return command, nil
	}
	if !costModes[v] {
		return "", fmt.Errorf("unknown cost mode %q (valid: chat, print, resume, backfill, benchmark, live-eval)", v)
	}
	return v, nil
}

// resolvePermissionMode returns the effective permission tier: the
// --permission-mode flag wins over the PI_PERMISSION_MODE env var. An empty
// result means no explicit tier (pi's own defaults apply). Unknown values are
// usage errors.
func resolvePermissionMode(flagVal string) (string, error) {
	v := flagVal
	if v == "" {
		v = os.Getenv("PI_PERMISSION_MODE")
	}
	if v == "" {
		return "", nil
	}
	if !permissionModes[v] {
		return "", fmt.Errorf("unknown permission mode %q (valid: default, plan, acceptEdits, bypassPermissions)", v)
	}
	return v, nil
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

	// Create the replacement in the same directory, then rename it over the
	// existing link. os.Rename is atomic when both paths share a filesystem.
	tmpFile, err := os.CreateTemp(filepath.Dir(link), ".pi-run-link-")
	if err != nil {
		return fmt.Errorf("create temporary link for %s: %w", link, err)
	}
	tmp := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close temporary link for %s: %w", link, err)
	}
	if err := os.Remove(tmp); err != nil {
		return fmt.Errorf("prepare temporary link for %s: %w", link, err)
	}
	if err := os.Symlink(target, tmp); err != nil {
		return fmt.Errorf("create temporary symlink for %s: %w", link, err)
	}
	defer os.Remove(tmp) // best-effort cleanup if rename fails
	if err := os.Rename(tmp, link); err != nil {
		return fmt.Errorf("atomically replace %s: %w", link, err)
	}
	return nil
}

// buildVersion returns the version to stamp into the binary: "dev" for local
// builds. Release workflows pass the tag via -ldflags directly (see
// scripts/build-release.sh); this default keeps `pi-run install` simple.
func buildVersion() string {
	return "dev"
}

// runClean removes eval/.venv and pytest caches, reporting what it removed.
func runClean() int {
	root := repoRoot()
	removed := false
	for _, p := range []string{
		filepath.Join(root, "eval", ".venv"),
		filepath.Join(root, "eval", "__pycache__"),
		filepath.Join(root, "eval", "tests", "__pycache__"),
		filepath.Join(root, "eval", ".pytest_cache"),
	} {
		if _, err := os.Lstat(p); err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("nothing to clean: %s\n", p)
				continue
			}
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := os.RemoveAll(p); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		removed = true
		fmt.Printf("removed %s\n", p)
	}
	if removed {
		fmt.Println("clean complete")
	} else {
		fmt.Println("nothing to clean")
	}
	return 0
}

// runProviders lists configured providers, their default models, the model
// tiers each provider offers, and the key env var.
func runProviders() int {
	for _, p := range Providers {
		line := fmt.Sprintf("%s\t%s\t%s\t%s", p.Name, p.DefaultModel, joinStrings(availableTiers(p), ","), p.KeyEnv)
		if p.BaseURL != "" {
			line += "\t" + p.BaseURL
		}
		fmt.Println(line)
	}
	return 0
}
