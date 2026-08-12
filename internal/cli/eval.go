package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// supportedProviderKeyEnvs mirrors eval/conftest.py's SUPPORTED_PROVIDER_KEYS
// and must cover every keyEnv in the provider catalog (providers.json /
// defaultProviders). Env-only (no Bitwarden) so the eval command never blocks
// on a vault.
var supportedProviderKeyEnvs = []string{
	"OPENROUTER_API_KEY",
	"OPENAI_API_KEY",
	"ANTHROPIC_API_KEY",
	"GEMINI_API_KEY",
	"GROQ_API_KEY",
	"DEEPSEEK_API_KEY",
	"LOCAL_API_KEY",
	"AZURE_OPENAI_API_KEY",
	"OLLAMA_API_KEY",
	"MISTRAL_API_KEY",
	"COHERE_API_KEY",
	"TOGETHER_API_KEY",
	"PERPLEXITY_API_KEY",
	"FIREWORKS_API_KEY",
	"MOONSHOT_API_KEY",
	"XAI_API_KEY",
	"BEDROCK_API_KEY",
}

const evalUsage = `Usage: pi-run eval [--quick] [--benchmark [name]] [--benchmark-dry-run]
             [--provider <name>] [--model <id>] [-- <pytest args...>]

Run the DeepEval pytest suite.
  --quick           Run the deterministic smoke subset.
  --                Pass remaining arguments directly to pytest.

Run Docker-isolated benchmark tasks (eval/benchmarks/<name>/task.json).
  --benchmark [name]  Run the benchmark suite (all tasks when name is omitted).
  --benchmark-dry-run Validate all task.json files only; no Docker, no keys.
  --provider <name>   Agent provider (see 'pi-run providers').
  --model <id>        Override the per-provider default model.
`

// anyProviderKeyEnv reports whether any supported provider key is set in the
// environment (presence only; never the value).
func anyProviderKeyEnv() bool {
	for _, k := range supportedProviderKeyEnvs {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}

// evalOptions is the parsed result of eval's owned flags.
type evalOptions struct {
	quick           bool
	benchmark       string // selected benchmark name ("" = all)
	benchmarkMode   bool
	benchmarkDryRun bool
	provider        string
	model           string
	pytestArgs      []string
}

// parseEvalArgs validates eval's owned flags and returns the parsed options.
// A returned exitCode >= 0 means parsing failed (or --help) and the caller
// should return it; -1 means "run the command".
func parseEvalArgs(args []string) (opts evalOptions, exitCode int) {
	i := 0
	for i < len(args) {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			fmt.Print(evalUsage)
			return opts, 0
		case arg == "--quick":
			opts.quick = true
		case arg == "--benchmark":
			opts.benchmarkMode = true
			// Optional positional name; a following flag means "all tasks".
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				opts.benchmark = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--benchmark="):
			opts.benchmarkMode = true
			opts.benchmark = strings.TrimPrefix(arg, "--benchmark=")
		case arg == "--benchmark-dry-run":
			opts.benchmarkMode = true
			opts.benchmarkDryRun = true
		case arg == "--provider" && i+1 < len(args):
			opts.provider = args[i+1]
			i++
		case strings.HasPrefix(arg, "--provider="):
			opts.provider = strings.TrimPrefix(arg, "--provider=")
		case arg == "--model" && i+1 < len(args):
			opts.model = args[i+1]
			i++
		case strings.HasPrefix(arg, "--model="):
			opts.model = strings.TrimPrefix(arg, "--model=")
		case arg == "--provider" || arg == "--model":
			fmt.Fprintf(os.Stderr, "pi-run: eval: %s requires an argument\n\n%s", arg, evalUsage)
			return opts, 2
		case arg == "--":
			opts.pytestArgs = args[i+1:]
			i = len(args)
		default:
			fmt.Fprintf(os.Stderr, "pi-run: eval: unknown flag or argument %q\n\n%s", arg, evalUsage)
			return opts, 2
		}
		i++
	}
	// Cross-flag validation: benchmark flags are only meaningful together.
	if !opts.benchmarkMode && (opts.provider != "" || opts.model != "") {
		fmt.Fprintf(os.Stderr, "pi-run: eval: --provider/--model require --benchmark\n\n%s", evalUsage)
		return opts, 2
	}
	if opts.benchmarkMode && opts.quick {
		fmt.Fprintf(os.Stderr, "pi-run: eval: --quick cannot be combined with --benchmark\n\n%s", evalUsage)
		return opts, 2
	}
	if opts.benchmarkMode && len(opts.pytestArgs) > 0 {
		fmt.Fprintf(os.Stderr, "pi-run: eval: --benchmark does not accept pytest pass-through (--)\n\n%s", evalUsage)
		return opts, 2
	}
	return opts, -1
}

// runEval runs the DeepEval pytest suite, or the Docker-isolated benchmark
// runner when --benchmark/--benchmark-dry-run is present. --quick runs the
// smoke subset; a -- delimiter passes the remaining arguments through to pytest.
func runEval(cliArgs []string) int {
	opts, parseCode := parseEvalArgs(cliArgs)
	if parseCode >= 0 {
		return parseCode
	}
	if opts.benchmarkMode {
		return runBenchmarks(opts)
	}

	root := repoRoot()

	// Hooks: pre-eval fires before pytest, post-eval always after — even when
	// pytest fails — so CI cleanup/notification steps still run.
	if err := runHooks(hookEventPreEval); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return hookExitCode(err)
	}

	evalDir := filepath.Join(root, "eval")
	venvPython := filepath.Join(evalDir, ".venv", "bin", "python")
	if _, err := os.Stat(venvPython); err != nil {
		fmt.Fprintln(os.Stderr, "pi-run: eval: eval/.venv missing — run `pi-run setup` first")
		return 5
	}
	args := []string{"-m", "pytest"}
	if len(opts.pytestArgs) > 0 {
		args = append(args, opts.pytestArgs...)
	} else if opts.quick {
		args = append(args, "tests/test_code_quality.py",
			"tests/test_agent_task_completion.py::test_dataset_expected_outputs_are_non_empty")
	} else if !anyProviderKeyEnv() {
		fmt.Println("No provider key found — run `pi-run doctor` to check keys. Skipping live tests.")
		args = append(args, "tests/test_harness_config.py", "tests/test_code_quality.py",
			"tests/test_agent_task_completion.py::test_dataset_expected_outputs_are_non_empty")
	}
	args = append(args, "-v")
	code, err := runCmd(venvPython, args, evalDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
	}
	// post-eval always fires after the suite; a failed post-eval hook (without
	// continueOnError) overrides pytest's exit code so CI can't miss it.
	if err := runHooks(hookEventPostEval); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return hookExitCode(err)
	}
	return code
}
