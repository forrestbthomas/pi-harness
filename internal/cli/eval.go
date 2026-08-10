package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// supportedProviderKeyEnvs mirrors eval/conftest.py's SUPPORTED_PROVIDER_KEYS.
// Env-only (no Bitwarden) so the eval command never blocks on a vault.
var supportedProviderKeyEnvs = []string{
	"OPENROUTER_API_KEY",
	"OPENAI_API_KEY",
	"ANTHROPIC_API_KEY",
	"GEMINI_API_KEY",
	"GROQ_API_KEY",
	"DEEPSEEK_API_KEY",
	"LOCAL_API_KEY",
}

const evalUsage = `Usage: pi-run eval [--quick] [-- <pytest args...>]

Run the DeepEval pytest suite.
  --quick  Run the deterministic smoke subset.
  --       Pass remaining arguments directly to pytest.
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

// parseEvalArgs validates eval's owned flags and returns whether to run the
// smoke subset plus any pytest arguments after a -- delimiter.
func parseEvalArgs(args []string) (quick bool, pytestArgs []string, exitCode int) {
	for i, arg := range args {
		switch arg {
		case "--help", "-h":
			fmt.Print(evalUsage)
			return false, nil, 0
		case "--quick":
			quick = true
		case "--":
			return quick, args[i+1:], -1
		default:
			fmt.Fprintf(os.Stderr, "pi-run: eval: unknown flag or argument %q\n\n%s", arg, evalUsage)
			return false, nil, 2
		}
	}
	return quick, nil, -1
}

// runEval runs the DeepEval pytest suite. --quick runs the smoke subset; a
// -- delimiter passes the remaining arguments through to pytest.
func runEval(cliArgs []string) int {
	quick, pytestArgs, parseCode := parseEvalArgs(cliArgs)
	if parseCode >= 0 {
		return parseCode
	}

	root := repoRoot()
	evalDir := filepath.Join(root, "eval")
	venvPython := filepath.Join(evalDir, ".venv", "bin", "python")
	if _, err := os.Stat(venvPython); err != nil {
		fmt.Fprintln(os.Stderr, "pi-run: eval: eval/.venv missing — run `pi-run setup` first")
		return 5
	}
	args := []string{"-m", "pytest"}
	if len(pytestArgs) > 0 {
		args = append(args, pytestArgs...)
	} else if quick {
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
		return 1
	}
	return code
}
