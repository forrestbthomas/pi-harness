package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// runEval runs the DeepEval pytest suite. --quick runs the smoke subset.
func runEval(quick bool) int {
	root := repoRoot()
	evalDir := filepath.Join(root, "eval")
	venvPython := filepath.Join(evalDir, ".venv", "bin", "python")
	if _, err := os.Stat(venvPython); err != nil {
		fmt.Fprintln(os.Stderr, "eval/.venv is missing — run `pi-run setup` first")
		return 5
	}
	args := []string{"-m", "pytest"}
	if quick {
		args = append(args, "tests/test_code_quality.py",
			"tests/test_agent_task_completion.py::test_dataset_expected_outputs_are_non_empty")
	} else {
		args = append(args, "tests/")
	}
	args = append(args, "-v")
	code, err := runCmd(venvPython, args, evalDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return code
}
