package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeBenchmarkTask creates a benchmark task directory under root with the
// given task.json content and a stock tests/run.sh, returning the task dir.
func writeBenchmarkTask(t *testing.T, root, name, taskJSON string) string {
	t.Helper()
	dir := filepath.Join(root, "eval", "benchmarks", name)
	if err := os.MkdirAll(filepath.Join(dir, "tests"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "task.json"), []byte(taskJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tests", "run.sh"), []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadBenchmarkTasksValid(t *testing.T) {
	root := t.TempDir()
	writeBenchmarkTask(t, root, "demo", `{"id": "demo", "prompt": "Fix it.", "timeoutSecs": 60}`)
	tasks, errs := loadBenchmarkTasks(root, "")
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(tasks) != 1 || tasks[0].ID != "demo" {
		t.Fatalf("got %+v", tasks)
	}
	if tasks[0].TestScript != "tests/run.sh" {
		t.Fatalf("default testScript not applied: %q", tasks[0].TestScript)
	}
	if tasks[0].TimeoutSecs != 60 {
		t.Fatalf("timeoutSecs = %d, want 60", tasks[0].TimeoutSecs)
	}
}

func TestLoadBenchmarkTasksDefaultTimeout(t *testing.T) {
	root := t.TempDir()
	writeBenchmarkTask(t, root, "demo", `{"id": "demo", "prompt": "Fix it."}`)
	tasks, errs := loadBenchmarkTasks(root, "")
	if len(errs) != 0 || len(tasks) != 1 {
		t.Fatalf("tasks=%v errs=%v", tasks, errs)
	}
	if tasks[0].TimeoutSecs != defaultBenchmarkTimeout {
		t.Fatalf("default timeout = %d, want %d", tasks[0].TimeoutSecs, defaultBenchmarkTimeout)
	}
}

func TestLoadBenchmarkTasksInstructionFile(t *testing.T) {
	root := t.TempDir()
	dir := writeBenchmarkTask(t, root, "demo", `{"id": "demo", "instruction": "instruction.md"}`)
	if err := os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("Fix the bug."), 0o644); err != nil {
		t.Fatal(err)
	}
	tasks, errs := loadBenchmarkTasks(root, "")
	if len(errs) != 0 || len(tasks) != 1 {
		t.Fatalf("tasks=%v errs=%v", tasks, errs)
	}
	if tasks[0].Prompt != "Fix the bug." {
		t.Fatalf("instruction not loaded into prompt: %q", tasks[0].Prompt)
	}
}

func TestLoadBenchmarkTasksMissingInstructionFile(t *testing.T) {
	root := t.TempDir()
	writeBenchmarkTask(t, root, "demo", `{"id": "demo", "instruction": "missing.md"}`)
	_, errs := loadBenchmarkTasks(root, "")
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want 1", errs)
	}
	if !strings.Contains(errs[0].Error(), "missing.md") {
		t.Fatalf("error %q should mention missing.md", errs[0])
	}
}

func TestLoadBenchmarkTasksMissingID(t *testing.T) {
	root := t.TempDir()
	writeBenchmarkTask(t, root, "bad", `{"prompt": "Fix it."}`)
	tasks, errs := loadBenchmarkTasks(root, "")
	if len(tasks) != 0 || len(errs) != 1 {
		t.Fatalf("tasks=%v errs=%v", tasks, errs)
	}
	if !strings.Contains(errs[0].Error(), `"id"`) {
		t.Fatalf("error %q should mention missing id", errs[0])
	}
}

func TestLoadBenchmarkTasksMissingPromptAndInstruction(t *testing.T) {
	root := t.TempDir()
	writeBenchmarkTask(t, root, "bad", `{"id": "bad"}`)
	_, errs := loadBenchmarkTasks(root, "")
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "prompt") {
		t.Fatalf("errs = %v, want missing-prompt error", errs)
	}
}

func TestLoadBenchmarkTasksBothPromptAndInstruction(t *testing.T) {
	root := t.TempDir()
	dir := writeBenchmarkTask(t, root, "bad", `{"id": "bad", "prompt": "x", "instruction": "instruction.md"}`)
	if err := os.WriteFile(filepath.Join(dir, "instruction.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, errs := loadBenchmarkTasks(root, "")
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "not both") {
		t.Fatalf("errs = %v, want both-fields error", errs)
	}
}

func TestLoadBenchmarkTasksUnknownFieldRejected(t *testing.T) {
	root := t.TempDir()
	writeBenchmarkTask(t, root, "bad", `{"id": "bad", "prompt": "x", "bogus": 1}`)
	_, errs := loadBenchmarkTasks(root, "")
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "bogus") {
		t.Fatalf("errs = %v, want unknown-field error naming bogus", errs)
	}
}

func TestLoadBenchmarkTasksMissingTestScript(t *testing.T) {
	root := t.TempDir()
	writeBenchmarkTask(t, root, "demo", `{"id": "demo", "prompt": "x", "testScript": "tests/check.sh"}`)
	_, errs := loadBenchmarkTasks(root, "")
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "check.sh") {
		t.Fatalf("errs = %v, want missing testScript error", errs)
	}
}

func TestLoadBenchmarkTasksNegativeTimeout(t *testing.T) {
	root := t.TempDir()
	writeBenchmarkTask(t, root, "bad", `{"id": "bad", "prompt": "x", "timeoutSecs": -5}`)
	_, errs := loadBenchmarkTasks(root, "")
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "timeoutSecs") {
		t.Fatalf("errs = %v, want timeout error", errs)
	}
}

func TestLoadBenchmarkTasksByName(t *testing.T) {
	root := t.TempDir()
	writeBenchmarkTask(t, root, "aaa", `{"id": "aaa", "prompt": "x"}`)
	writeBenchmarkTask(t, root, "bbb", `{"id": "bbb", "prompt": "y"}`)
	tasks, errs := loadBenchmarkTasks(root, "bbb")
	if len(errs) != 0 || len(tasks) != 1 || tasks[0].ID != "bbb" {
		t.Fatalf("tasks=%v errs=%v", tasks, errs)
	}
}

func TestLoadBenchmarkTasksRejectsUnsafeName(t *testing.T) {
	root := t.TempDir()
	writeBenchmarkTask(t, root, "aaa", `{"id": "aaa", "prompt": "x"}`)
	for _, name := range []string{"../evil", "a/b", ".hidden"} {
		_, errs := loadBenchmarkTasks(root, name)
		if len(errs) == 0 {
			t.Fatalf("name %q accepted, want validation error", name)
		}
	}
}

func TestLoadBenchmarkTasksMissingDir(t *testing.T) {
	root := t.TempDir()
	_, errs := loadBenchmarkTasks(root, "")
	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "eval/benchmarks") {
		t.Fatalf("errs = %v, want missing-dir error", errs)
	}
}

func TestLoadBenchmarkTasksDuplicateIDs(t *testing.T) {
	root := t.TempDir()
	// Two directories with the same task id must be rejected: they would
	// collide in Docker image/container names.
	writeBenchmarkTask(t, root, "aaa", `{"id": "same", "prompt": "x"}`)
	writeBenchmarkTask(t, root, "bbb", `{"id": "same", "prompt": "y"}`)
	tasks, errs := loadBenchmarkTasks(root, "")
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want 1 duplicate-id error", errs)
	}
	if !strings.Contains(errs[0].Error(), "duplicate benchmark id \"same\"") {
		t.Fatalf("error %q should mention duplicate id", errs[0])
	}
	if len(tasks) != 2 {
		t.Fatalf("tasks = %d, want both parsed (error reported separately)", len(tasks))
	}
}

// TestSeedBenchmarksGradeWithSolutions is a hermetic smoke test (no Docker)
// that proves the seed suite can actually pass the real runner's grading: for
// each seed task it prepares the workspace exactly as prepareWorkspace does,
// applies the task's solution over src/ when one ships (tasks without a
// solution intentionally contain buggy code), then runs tests/run.sh with the
// local tooling. With a solution grading must exit 0; without one it must
// FAIL on the task's real bug — not on a layout error (src/ flattening, missing
// files). This catches workspace-layout regressions that dry-run format
// validation cannot see.
func TestSeedBenchmarksGradeWithSolutions(t *testing.T) {
	// Derive the repo root from this test file's own path (same pattern as
	// TestModulePath) so the test is hermetic — no hardcoded user paths.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	if _, err := os.Stat(filepath.Join(root, "eval", "benchmarks")); err != nil {
		t.Skipf("seed benchmarks not found under %s: %v", root, err)
	}
	tasks, errs := loadBenchmarkTasks(root, "")
	if len(errs) != 0 || len(tasks) == 0 {
		t.Fatalf("seed tasks failed to load: tasks=%d errs=%v", len(tasks), errs)
	}

	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available; skipping seed-grade smoke test")
	}

	for _, task := range tasks {
		t.Run(task.ID, func(t *testing.T) {
			ws, err := prepareWorkspace(task, t.TempDir())
			if err != nil {
				t.Fatalf("prepareWorkspace: %v", err)
			}
			hasSolution := task.Solution != "" && dirExists(filepath.Join(task.Dir, task.Solution))
			if hasSolution {
				// Apply the solution over src/ so grading should pass.
				if err := copyTree(filepath.Join(task.Dir, task.Solution), filepath.Join(ws, "src")); err != nil {
					t.Fatalf("apply solution: %v", err)
				}
			}
			// Run the task's verification script with local tooling (no Docker).
			cmd := exec.Command("bash", filepath.Join(ws, task.TestScript))
			cmd.Dir = ws
			out, err := cmd.CombinedOutput()
			if hasSolution {
				if err != nil {
					t.Fatalf("grading %s with solution failed (exit %v):\n%s", task.ID, err, out)
				}
				return
			}
			// No solution: the task ships buggy src/, so grading must fail — but
			// on the real bug, not on a workspace-layout error. A layout failure
			// (missing src/, flattened paths) would surface as one of these.
			if err == nil {
				t.Fatalf("task %s passed without a solution; expected the intentional bug to fail grading", task.ID)
			}
			text := string(out)
			for _, marker := range []string{"No such file or directory", "ModuleNotFoundError", "command not found", "src/: No such file"} {
				if strings.Contains(text, marker) {
					t.Fatalf("task %s failed with a layout error (%q) instead of the task bug:\n%s", task.ID, marker, text)
				}
			}
		})
	}
}

func TestParseEvalArgsBenchmarkNamed(t *testing.T) {
	opts, code := parseEvalArgs([]string{"--benchmark", "demo", "--provider", "deepseek", "--model", "deepseek/deepseek-v4-flash"})
	if code != -1 {
		t.Fatalf("code = %d, want -1", code)
	}
	if !opts.benchmarkMode || opts.benchmark != "demo" || opts.provider != "deepseek" || opts.model != "deepseek/deepseek-v4-flash" {
		t.Fatalf("opts = %+v", opts)
	}
}

func TestParseEvalArgsBenchmarkAll(t *testing.T) {
	opts, code := parseEvalArgs([]string{"--benchmark", "--provider", "openai"})
	if code != -1 {
		t.Fatalf("code = %d, want -1", code)
	}
	if !opts.benchmarkMode || opts.benchmark != "" || opts.provider != "openai" {
		t.Fatalf("opts = %+v", opts)
	}
}

func TestParseEvalArgsBenchmarkEqualsForm(t *testing.T) {
	opts, code := parseEvalArgs([]string{"--benchmark=demo", "--model=openai/gpt-5.6-terra"})
	if code != -1 {
		t.Fatalf("code = %d, want -1", code)
	}
	if opts.benchmark != "demo" || opts.model != "openai/gpt-5.6-terra" {
		t.Fatalf("opts = %+v", opts)
	}
}

func TestParseEvalArgsDryRunImpliesBenchmark(t *testing.T) {
	opts, code := parseEvalArgs([]string{"--benchmark-dry-run"})
	if code != -1 {
		t.Fatalf("code = %d, want -1", code)
	}
	if !opts.benchmarkMode || !opts.benchmarkDryRun {
		t.Fatalf("opts = %+v", opts)
	}
}

func TestParseEvalArgsProviderNeedsArgument(t *testing.T) {
	_, code := parseEvalArgs([]string{"--benchmark", "--provider"})
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	_, code = parseEvalArgs([]string{"--benchmark", "--model"})
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

func TestParseEvalArgsProviderRequiresBenchmark(t *testing.T) {
	_, code := parseEvalArgs([]string{"--provider", "openai"})
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	_, code = parseEvalArgs([]string{"--model", "openai/gpt-5.6-terra"})
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

func TestParseEvalArgsQuickConflictsWithBenchmark(t *testing.T) {
	_, code := parseEvalArgs([]string{"--quick", "--benchmark"})
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

func TestParseEvalArgsPytestPassThroughConflictsWithBenchmark(t *testing.T) {
	_, code := parseEvalArgs([]string{"--benchmark", "--", "tests/test_x.py"})
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

func TestParseEvalArgsUnknownFlag(t *testing.T) {
	_, code := parseEvalArgs([]string{"--benchmark", "--bogus"})
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

func TestRunEvalBenchmarkDryRun(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	// No eval/.venv and no provider keys: dry-run must be fully hermetic.
	for _, key := range supportedProviderKeyEnvs {
		t.Setenv(key, "")
	}
	writeBenchmarkTask(t, root, "demo", `{"id": "demo", "prompt": "Fix it."}`)
	code, out := captureRunStdout(t, []string{"eval", "--benchmark", "--benchmark-dry-run"})
	if code != 0 {
		t.Fatalf("dry run exit = %d, want 0; output:\n%s", code, out)
	}
	if !strings.Contains(out, "valid demo") {
		t.Fatalf("dry run output missing valid line:\n%s", out)
	}
	if !strings.Contains(out, "1/1 tasks valid") {
		t.Fatalf("dry run output missing summary:\n%s", out)
	}
}

func TestRunEvalBenchmarkDryRunInvalidTask(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	writeBenchmarkTask(t, root, "bad", `{"prompt": "no id here"}`)
	code, out := captureRunStderr(t, []string{"eval", "--benchmark-dry-run"})
	if code != 1 {
		t.Fatalf("dry run invalid exit = %d, want 1; stderr:\n%s", code, out)
	}
	if !strings.Contains(out, "bad") {
		t.Fatalf("stderr missing task name:\n%s", out)
	}
}

func TestRunEvalBenchmarkDryRunNamedTask(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	writeBenchmarkTask(t, root, "aaa", `{"id": "aaa", "prompt": "x"}`)
	writeBenchmarkTask(t, root, "bad", `{"prompt": "no id"}`)
	// Selecting aaa validates only aaa, so the broken bad task is ignored.
	code, out := captureRunStdout(t, []string{"eval", "--benchmark", "aaa", "--benchmark-dry-run"})
	if code != 0 {
		t.Fatalf("dry run named exit = %d, want 0; output:\n%s", code, out)
	}
	if !strings.Contains(out, "valid aaa") {
		t.Fatalf("dry run output missing aaa:\n%s", out)
	}
}

func TestRunEvalBenchmarkNoDocker(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	writeBenchmarkTask(t, root, "demo", `{"id": "demo", "prompt": "x"}`)
	// Hide docker from PATH: the live path must fail fast with exit 7 before
	// any key/node resolution.
	t.Setenv("PATH", t.TempDir())
	code, out := captureRunStderr(t, []string{"eval", "--benchmark"})
	if code != 7 {
		t.Fatalf("no-docker exit = %d, want 7; stderr:\n%s", code, out)
	}
	if !strings.Contains(out, "Docker") {
		t.Fatalf("stderr missing docker message:\n%s", out)
	}
}

func TestRunEvalBenchmarkUnknownFlag(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	writeBenchmarkTask(t, root, "demo", `{"id": "demo", "prompt": "x"}`)
	code, out := captureRunStderr(t, []string{"eval", "--benchmark", "--bogus"})
	if code != 2 {
		t.Fatalf("unknown flag exit = %d, want 2; stderr:\n%s", code, out)
	}
	if !strings.Contains(out, "unknown flag") {
		t.Fatalf("stderr missing unknown-flag error:\n%s", out)
	}
}

func TestRunEvalProviderRequiresBenchmark(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	code, out := captureRunStderr(t, []string{"eval", "--provider", "openai"})
	if code != 2 {
		t.Fatalf("provider-without-benchmark exit = %d, want 2; stderr:\n%s", code, out)
	}
	if !strings.Contains(out, "--benchmark") {
		t.Fatalf("stderr missing --benchmark hint:\n%s", out)
	}
}

func TestAggregateBenchmarkResults(t *testing.T) {
	results := []benchmarkTaskResult{
		{ID: "a", Status: "pass", Passed: true},
		{ID: "b", Status: "fail"},
		{ID: "c", Status: "error"},
		{ID: "d", Status: "pass", Passed: true},
	}
	s := aggregateBenchmarkResults(results)
	if s.Total != 4 || s.Passed != 2 || s.Failed != 1 || s.Errors != 1 {
		t.Fatalf("summary = %+v", s)
	}
	if s.Score != 0.5 {
		t.Fatalf("score = %v, want 0.5", s.Score)
	}
}

func TestAggregateBenchmarkResultsEmpty(t *testing.T) {
	s := aggregateBenchmarkResults(nil)
	if s.Total != 0 || s.Passed != 0 || s.Score != 0 {
		t.Fatalf("summary = %+v", s)
	}
}

func TestBenchmarkRunResultJSONRoundTrip(t *testing.T) {
	run := benchmarkRunResult{
		RunID:     "20260811T101112-openai-openai-gpt-5.6-terra",
		Provider:  "openai",
		Model:     "openai/gpt-5.6-terra",
		Timestamp: "2026-08-11T10:11:12Z",
		Tasks: []benchmarkTaskResult{
			{ID: "demo", Status: "pass", Passed: true, Duration: 12.5},
			{ID: "other", Status: "fail", Duration: 3.1, Error: "tests/run.sh exited 1"},
		},
		Summary: benchmarkSummary{Total: 2, Passed: 1, Failed: 1, Score: 0.5},
	}
	b, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	var got benchmarkRunResult
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.RunID != run.RunID || got.Summary.Passed != 1 || len(got.Tasks) != 2 {
		t.Fatalf("round trip mismatch: %+v", got)
	}
	if got.Tasks[1].Status != "fail" || got.Tasks[1].Error != "tests/run.sh exited 1" {
		t.Fatalf("task round trip mismatch: %+v", got.Tasks[1])
	}
}

func TestWriteBenchmarkResults(t *testing.T) {
	root := t.TempDir()
	results := []benchmarkTaskResult{{ID: "demo", Status: "pass", Passed: true}}
	summary := aggregateBenchmarkResults(results)
	path, err := writeBenchmarkResults(root, "run-1", "openai", "openai/gpt-5.6-terra", results, summary)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, "benchmark-results") {
		t.Fatalf("path = %q, want under benchmark-results", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got benchmarkRunResult
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.RunID != "run-1" || got.Summary.Passed != 1 || got.Summary.Total != 1 {
		t.Fatalf("written result mismatch: %+v", got)
	}
}

func TestSanitizeForID(t *testing.T) {
	if got := sanitizeForID("openai/gpt-5.6-terra"); got != "openai-gpt-5.6-terra" {
		t.Fatalf("got %q", got)
	}
	if got := sanitizeForID("demo task!"); got != "demo-task-" {
		t.Fatalf("got %q", got)
	}
}

func TestPrepareWorkspaceCopiesSrcAndTests(t *testing.T) {
	root := t.TempDir()
	dir := writeBenchmarkTask(t, root, "demo", `{"id": "demo", "prompt": "x"}`)
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "calc.py"), []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	task := benchmarkTask{ID: "demo", Dir: dir, TestScript: "tests/run.sh"}
	base := t.TempDir()
	ws, err := prepareWorkspace(task, base)
	if err != nil {
		t.Fatal(err)
	}
	// The workspace must preserve the src/ subdir (the seed suite and prompts
	// reference src/... paths) plus tests/ at the workspace root.
	for _, want := range []string{"src/calc.py", "tests/run.sh"} {
		if _, err := os.Stat(filepath.Join(ws, want)); err != nil {
			t.Fatalf("workspace missing %s: %v", want, err)
		}
	}
	// The workspace must not have flattened src contents at its root.
	if _, err := os.Stat(filepath.Join(ws, "calc.py")); !os.IsNotExist(err) {
		t.Fatalf("workspace should not contain a flattened src file at root: %v", err)
	}
}

// TestPrepareWorkspacePreservesSrcForRepoLayout verifies the src/ subdir is
// preserved even when only src/ and tests/ exist (no repo clone).
func TestPrepareWorkspacePreservesSrcLayout(t *testing.T) {
	root := t.TempDir()
	dir := writeBenchmarkTask(t, root, "demo", `{"id": "demo", "prompt": "x"}`)
	if err := os.MkdirAll(filepath.Join(dir, "src", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "sub", "a.py"), []byte("a=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	task := benchmarkTask{ID: "demo", Dir: dir, TestScript: "tests/run.sh"}
	ws, err := prepareWorkspace(task, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ws, "src", "sub", "a.py")); err != nil {
		t.Fatalf("nested src path not preserved: %v", err)
	}
}
