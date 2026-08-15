package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// defaultBenchmarkImage is the pinned base image used when a task has no
// environment/Dockerfile. python:3.12-slim ships bash plus the Python
// interpreter, which covers the seed suite (Python and shell tasks).
const defaultBenchmarkImage = "python:3.12-slim"

// defaultBenchmarkTimeout applies when task.json omits timeoutSecs.
const defaultBenchmarkTimeout = 300

// benchmarkNameRe constrains task ids and --benchmark names: they are used in
// filesystem paths and Docker image/container names, so only safe characters
// are allowed and the first character must be alphanumeric.
var benchmarkNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// Shared task taxonomy (live-agent-eval-v2 spec §4.6): the same categories and
// difficulty levels used by the live JSONL surface and the unified manifest
// (eval/datasets/tasks.json), so live vs benchmark scores are comparable.
// Benchmark tasks are always deterministically graded (spec §4.5: Docker
// grading has no judge), so grader must be "deterministic".
var benchmarkCategories = []string{
	"code-gen", "bug-fix", "shell/ops", "concept", "negative-edge", "harness-routing", "agentic",
}

var benchmarkDifficulties = []string{"easy", "medium", "hard"}

const benchmarkGraderDeterministic = "deterministic"

// benchmarkTask is the parsed eval/benchmarks/<name>/task.json shape. Dir is
// the absolute task directory (not part of the file format).
type benchmarkTask struct {
	ID          string `json:"id"`
	Prompt      string `json:"prompt,omitempty"`
	Instruction string `json:"instruction,omitempty"` // relative path to a markdown task description
	SetupCmd    string `json:"setupCmd,omitempty"`    // run inside the container before tests/run.sh
	Repo        string `json:"repo,omitempty"`        // git URL cloned as the workspace (overrides src/)
	TimeoutSecs int    `json:"timeoutSecs,omitempty"` // per-task container timeout (default 300)
	TestScript  string `json:"testScript,omitempty"`  // relative path; default tests/run.sh
	Dockerfile  string `json:"dockerfile,omitempty"`  // relative path; default pinned base image
	Solution    string `json:"solution,omitempty"`    // optional oracle for future diff grading

	// Schema extension (spec §4.6): shared-taxonomy fields. Validated when
	// present so older task.json files keep parsing; the Python format lint
	// (test_benchmark_format.py) requires them on shipped tasks.
	Category   string `json:"category,omitempty"`
	Difficulty string `json:"difficulty,omitempty"`
	Grader     string `json:"grader,omitempty"`

	Dir string `json:"-"` // absolute task directory
}

// benchmarkTaskResult is the per-task outcome of a benchmark run.
type benchmarkTaskResult struct {
	ID       string  `json:"id"`
	Status   string  `json:"status"` // pass | fail | error
	Passed   bool    `json:"passed"`
	Duration float64 `json:"durationSecs"`
	Error    string  `json:"error,omitempty"`
}

// benchmarkSummary aggregates per-task outcomes.
type benchmarkSummary struct {
	Total  int     `json:"total"`
	Passed int     `json:"passed"`
	Failed int     `json:"failed"`
	Errors int     `json:"errors"`
	Score  float64 `json:"score"` // passed/total
}

// benchmarkRunResult is the on-disk JSON report shape.
type benchmarkRunResult struct {
	RunID     string                `json:"runId"`
	Provider  string                `json:"provider"`
	Model     string                `json:"model"`
	DryRun    bool                  `json:"dryRun"`
	Timestamp string                `json:"timestamp"`
	Tasks     []benchmarkTaskResult `json:"tasks"`
	Summary   benchmarkSummary      `json:"summary"`
}

// parseBenchmarkTask decodes a task.json, rejecting unknown fields so typos in
// the format surface during validation instead of being silently ignored.
func parseBenchmarkTask(data []byte) (benchmarkTask, error) {
	var t benchmarkTask
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return t, err
	}
	return t, nil
}

// validateBenchmarkTask checks a parsed task and fills in defaults. It mutates
// t (defaults + prompt text loaded from instruction.md).
func validateBenchmarkTask(t *benchmarkTask) error {
	if t.ID == "" {
		return errors.New(`missing required field "id"`)
	}
	if !benchmarkNameRe.MatchString(t.ID) {
		return fmt.Errorf("invalid id %q: must match %s", t.ID, benchmarkNameRe)
	}
	if t.Prompt == "" && t.Instruction == "" {
		return errors.New(`missing required field "prompt" (or "instruction" pointing at a markdown file)`)
	}
	if t.Prompt != "" && t.Instruction != "" {
		return errors.New(`set either "prompt" or "instruction", not both`)
	}
	if t.Instruction != "" {
		if !filepath.IsLocal(t.Instruction) {
			return fmt.Errorf("instruction %q must be a relative path inside the task directory", t.Instruction)
		}
		b, err := os.ReadFile(filepath.Join(t.Dir, t.Instruction))
		if err != nil {
			return fmt.Errorf("read instruction %q: %w", t.Instruction, err)
		}
		t.Prompt = string(b)
	}
	if t.TestScript == "" {
		t.TestScript = "tests/run.sh"
	}
	if !filepath.IsLocal(t.TestScript) {
		return fmt.Errorf("testScript %q must be a relative path inside the task directory", t.TestScript)
	}
	// Repo tasks verify their test script after clone; local tasks must ship it.
	if t.Repo == "" {
		if _, err := os.Stat(filepath.Join(t.Dir, t.TestScript)); err != nil {
			return fmt.Errorf("testScript %q not found in task directory: %v", t.TestScript, err)
		}
	}
	if t.TimeoutSecs < 0 {
		return fmt.Errorf("timeoutSecs must be >= 0, got %d", t.TimeoutSecs)
	}
	if t.TimeoutSecs == 0 {
		t.TimeoutSecs = defaultBenchmarkTimeout
	}
	if t.Dockerfile != "" {
		if !filepath.IsLocal(t.Dockerfile) {
			return fmt.Errorf("dockerfile %q must be a relative path inside the task directory", t.Dockerfile)
		}
		if _, err := os.Stat(filepath.Join(t.Dir, t.Dockerfile)); err != nil {
			return fmt.Errorf("dockerfile %q not found in task directory: %v", t.Dockerfile, err)
		}
	}
	if t.Solution != "" {
		if !filepath.IsLocal(t.Solution) {
			return fmt.Errorf("solution %q must be a relative path inside the task directory", t.Solution)
		}
		if _, err := os.Stat(filepath.Join(t.Dir, t.Solution)); err != nil {
			return fmt.Errorf("solution %q not found in task directory: %v", t.Solution, err)
		}
	}
	if t.Category != "" && !slices.Contains(benchmarkCategories, t.Category) {
		return fmt.Errorf("category %q must be one of %v", t.Category, benchmarkCategories)
	}
	if t.Difficulty != "" && !slices.Contains(benchmarkDifficulties, t.Difficulty) {
		return fmt.Errorf("difficulty %q must be one of %v", t.Difficulty, benchmarkDifficulties)
	}
	if t.Grader != "" && t.Grader != benchmarkGraderDeterministic {
		return fmt.Errorf("grader %q must be %q (benchmark tasks are deterministically graded, spec §4.5)", t.Grader, benchmarkGraderDeterministic)
	}
	return nil
}

// loadOneBenchmarkTask parses and validates a single task directory, appending
// any failure to errs (callers report all problems instead of failing fast).
func loadOneBenchmarkTask(benchRoot, name string, tasks []benchmarkTask, errs []error) ([]benchmarkTask, []error) {
	dir := filepath.Join(benchRoot, name)
	taskPath := filepath.Join(dir, "task.json")
	data, err := os.ReadFile(taskPath)
	if err != nil {
		return tasks, append(errs, fmt.Errorf("%s: %v", taskPath, err))
	}
	t, err := parseBenchmarkTask(data)
	if err != nil {
		return tasks, append(errs, fmt.Errorf("%s: %v", taskPath, err))
	}
	t.Dir = dir
	if err := validateBenchmarkTask(&t); err != nil {
		return tasks, append(errs, fmt.Errorf("%s: %v", taskPath, err))
	}
	return append(tasks, t), errs
}

// loadBenchmarkTasks discovers and validates benchmark tasks under
// eval/benchmarks. name selects a single task ("" = all). Per-task failures are
// collected in errs so validation can report every broken task at once.
func loadBenchmarkTasks(root, name string) ([]benchmarkTask, []error) {
	benchRoot := filepath.Join(root, "eval", "benchmarks")
	var tasks []benchmarkTask
	var errs []error
	if name != "" {
		if !benchmarkNameRe.MatchString(name) {
			return tasks, append(errs, fmt.Errorf("invalid benchmark name %q: must match %s", name, benchmarkNameRe))
		}
		return loadOneBenchmarkTask(benchRoot, name, tasks, errs)
	}
	entries, err := os.ReadDir(benchRoot)
	if err != nil {
		return tasks, append(errs, fmt.Errorf("eval/benchmarks: %v", err))
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		tasks, errs = loadOneBenchmarkTask(benchRoot, e.Name(), tasks, errs)
	}
	seen := make(map[string]string, len(tasks))
	for _, t := range tasks {
		if prev, dup := seen[t.ID]; dup {
			errs = append(errs, fmt.Errorf("duplicate benchmark id %q (in %q and %q)", t.ID, prev, t.Dir))
			continue
		}
		seen[t.ID] = t.Dir
	}
	if len(tasks) == 0 && len(errs) == 0 {
		return tasks, append(errs, fmt.Errorf("no benchmark tasks found under %s (each needs task.json + tests/run.sh)", benchRoot))
	}
	return tasks, errs
}

// runBenchmarks is the eval --benchmark entry point. Dry-run is hermetic (no
// Docker, no keys); a live run requires Docker, a provider key, and node/pi.
func runBenchmarks(opts evalOptions) int {
	root := repoRoot()
	tasks, errs := loadBenchmarkTasks(root, opts.benchmark)
	if opts.benchmarkDryRun {
		return reportBenchmarkDryRun(tasks, errs)
	}
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "pi-run: eval: benchmark: %v\n", err)
		}
		return 1
	}
	return runBenchmarkLive(tasks, opts, root)
}

// reportBenchmarkDryRun prints per-task validation status and exits 0 when
// every task is valid, 1 otherwise. Never touches Docker or provider keys.
func reportBenchmarkDryRun(tasks []benchmarkTask, errs []error) int {
	fmt.Println("== pi-run eval --benchmark (dry run) ==")
	for _, t := range tasks {
		fmt.Printf("  valid %s (%s)\n", t.ID, t.TestScript)
	}
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "  invalid: %v\n", err)
		}
		fmt.Fprintf(os.Stderr, "== benchmark dry run: %d task(s), %d invalid ==\n", len(tasks)+len(errs), len(errs))
		return 1
	}
	fmt.Printf("== benchmark dry run: %d/%d tasks valid ==\n", len(tasks), len(tasks))
	return 0
}

// benchmarkExitError wraps a benchmark failure with its documented exit code
// (1 generic, 2 provider resolution, 3 missing key, 4 node/pi missing, 7
// docker unavailable). Both runBenchmarkLive and ci-benchmark map it back to
// the code so eval --benchmark and the scorecard share one pre-flight.
type benchmarkExitError struct {
	code int
	err  error
}

func (e *benchmarkExitError) Error() string { return e.err.Error() }

// benchmarkErrorCode maps a runProviderBenchmark error to its exit code, 1
// for anything not wrapped in a benchmarkExitError.
func benchmarkErrorCode(err error) int {
	if be, ok := errors.AsType[*benchmarkExitError](err); ok {
		return be.code
	}
	return 1
}

// runBenchmarkLive runs each task end-to-end: prepare the workspace, run the
// agent against it, verify inside Docker, and record per-task results. It is a
// thin wrapper over runProviderBenchmark that keeps the eval --benchmark
// output and exit codes unchanged.
func runBenchmarkLive(tasks []benchmarkTask, opts evalOptions, root string) int {
	run, err := runProviderBenchmark(tasks, opts, root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: eval: benchmark: %v\n", err)
		return benchmarkErrorCode(err)
	}

	fmt.Printf("== pi-run eval --benchmark (%s/%s) ==\n", run.Provider, run.Model)
	for _, res := range run.Tasks {
		fmt.Printf("  %s %s (%.1fs)\n", strings.ToUpper(res.Status), res.ID, res.Duration)
		if res.Error != "" {
			fmt.Printf("       %s\n", res.Error)
		}
	}
	summary := run.Summary
	fmt.Printf("== score: %d/%d passed (%.0f%%) ==\n", summary.Passed, summary.Total, summary.Score*100)
	path, err := writeBenchmarkResults(root, run.RunID, run.Provider, run.Model, run.Tasks, summary)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: eval: benchmark: %v\n", err)
		return 1
	}
	fmt.Printf("results written to %s\n", path)
	if summary.Errors > 0 {
		fmt.Fprintln(os.Stderr, "pi-run: eval: benchmark: run incomplete: some tasks errored (see above)")
		return 1
	}
	return 0
}

// runProviderBenchmark runs the same per-task loop as eval --benchmark —
// workspace prep, execPiDirTimeout agent run, Docker build/run, grade — for
// one provider and returns the graded result instead of printing/writing it.
// It performs the same pre-flight checks as the eval wrapper (docker 7,
// provider 2, key 3, node 4); failures are wrapped in benchmarkExitError so
// callers map them to the documented exit codes. ci-benchmark reuses this
// function per provider, so per-provider cost attribution stays clean.
func runProviderBenchmark(tasks []benchmarkTask, opts evalOptions, root string) (benchmarkRunResult, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return benchmarkRunResult{}, &benchmarkExitError{7, errors.New("Docker not found on PATH — install Docker to run benchmarks, or use --benchmark-dry-run for format-only validation")}
	}
	p, err := ResolveProvider(opts.provider)
	if err != nil {
		return benchmarkRunResult{}, &benchmarkExitError{2, err}
	}
	key := ""
	if providerRequiresCredential(p) {
		key, err = resolveSecret(p.KeyEnv)
		if err != nil {
			return benchmarkRunResult{}, &benchmarkExitError{3, fmt.Errorf("no %s available: export it, or check your secret manager (`pi-run doctor`)", p.KeyEnv)}
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return benchmarkRunResult{}, &benchmarkExitError{1, err}
	}
	nodeVersion, err := resolveNodeVersion(home)
	if err != nil {
		return benchmarkRunResult{}, &benchmarkExitError{4, err}
	}
	model := benchmarkLaunchModel(p, opts.model)

	runID := benchmarkRunID(p.Name, model)
	wsBase, err := os.MkdirTemp("", "pi-bench-"+runID)
	if err != nil {
		return benchmarkRunResult{}, err
	}
	defer os.RemoveAll(wsBase)

	results := make([]benchmarkTaskResult, 0, len(tasks))
	for _, task := range tasks {
		results = append(results, runBenchmarkTask(task, p, model, key, nodeVersion, wsBase, runID))
	}
	summary := aggregateBenchmarkResults(results)
	return benchmarkRunResult{
		RunID:     runID,
		Provider:  p.Name,
		Model:     model,
		DryRun:    false,
		Timestamp: time.Now().Format(time.RFC3339),
		Tasks:     results,
		Summary:   summary,
	}, nil
}

// runBenchmarkTask executes one task and returns its graded result.
func runBenchmarkTask(task benchmarkTask, p Provider, model, key, nodeVersion, wsBase, runID string) benchmarkTaskResult {
	res := benchmarkTaskResult{ID: task.ID}
	start := time.Now()
	defer func() { res.Duration = time.Since(start).Seconds() }()

	// 1. Prepare the workspace the agent edits (src/ or repo clone + tests/).
	ws, err := prepareWorkspace(task, wsBase)
	if err != nil {
		res.Status = "error"
		res.Error = err.Error()
		return res
	}

	// 2. Run the agent against the workspace, bounded by the task timeout so a
	//    hung pi child cannot block the whole benchmark run.
	agentTimeout := time.Duration(task.TimeoutSecs) * time.Second
	code, err := execPiDirTimeout(nodeVersion, piArgs(p, model, "print", []string{task.Prompt}, false, ""), launchEnv(p, key), ws, agentTimeout)
	if err != nil || code != 0 {
		res.Status = "error"
		res.Error = fmt.Sprintf("agent run failed (exit %d: %v)", code, err)
		return res
	}

	// 3. Build the task image (custom Dockerfile) or use the pinned default.
	image := defaultBenchmarkImage
	if task.Dockerfile != "" {
		image = benchmarkImageName(runID, task.ID)
		args := []string{"build", "-t", image, "-f", filepath.Join(task.Dir, task.Dockerfile), task.Dir}
		code, err := dockerRun("docker", args, 10*time.Minute, "")
		if err != nil || code != 0 {
			res.Status = "error"
			res.Error = fmt.Sprintf("docker build failed (exit %d: %v)", code, err)
			return res
		}
	}

	// 4. Grade: run setupCmd (if any) then tests/run.sh in the container.
	cmdStr := "bash " + task.TestScript
	if task.SetupCmd != "" {
		cmdStr = task.SetupCmd + " && " + cmdStr
	}
	container := benchmarkContainerName(runID, task.ID)
	args := []string{"run", "--rm", "--name", container,
		"-v", ws + ":/workspace", "-w", "/workspace",
		image, "bash", "-c", cmdStr}
	code, err = dockerRun("docker", args, time.Duration(task.TimeoutSecs)*time.Second, container)
	if err != nil {
		res.Status = "error"
		res.Error = fmt.Sprintf("docker run failed: %v", err)
		return res
	}
	if code == 0 {
		res.Status = "pass"
		res.Passed = true
	} else {
		res.Status = "fail"
		res.Error = fmt.Sprintf("%s exited %d", task.TestScript, code)
	}
	return res
}

// dockerRun invokes the docker CLI, streaming output, bounded by timeout. On
// timeout it best-effort kills the named container (for `docker run`) so --rm
// can clean up, then reports the timeout as an error.
func dockerRun(dockerPath string, args []string, timeout time.Duration, killName string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, dockerPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		if killName != "" {
			_ = exec.Command(dockerPath, "kill", killName).Run()
		}
		return 1, fmt.Errorf("timed out after %s", timeout)
	}
	if err != nil {
		if ee, ok := errors.AsType[*exec.ExitError](err); ok {
			return ee.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

// prepareWorkspace creates the working directory for one task: either a shallow
// clone of task.Repo, or a copy of the task's src/ subdir (preserved as src/)
// plus its tests/. The seed suite and the task prompts reference src/... paths
// (e.g. PYTHONPATH=src, edit src/calc.py), so the src/ subdir must be kept
// intact in the workspace. The same files are later mounted into the container.
func prepareWorkspace(task benchmarkTask, base string) (string, error) {
	ws := filepath.Join(base, task.ID)
	if err := os.MkdirAll(ws, 0o755); err != nil {
		return "", err
	}
	if task.Repo != "" {
		code, err := runCmd("git", []string{"clone", "--depth", "1", task.Repo, ws}, base)
		if err != nil || code != 0 {
			return "", fmt.Errorf("clone %s failed (exit %d: %v)", task.Repo, code, err)
		}
		return ws, nil
	}
	if srcDir := filepath.Join(task.Dir, "src"); dirExists(srcDir) {
		if err := copyTree(srcDir, filepath.Join(ws, "src")); err != nil {
			return "", fmt.Errorf("copy src/: %w", err)
		}
	}
	if testsDir := filepath.Join(task.Dir, "tests"); dirExists(testsDir) {
		if err := copyTree(testsDir, filepath.Join(ws, "tests")); err != nil {
			return "", fmt.Errorf("copy tests/: %w", err)
		}
	}
	return ws, nil
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// copyTree recursively copies the contents of src into dst.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

// aggregateBenchmarkResults computes the pass/fail/error summary.
func aggregateBenchmarkResults(results []benchmarkTaskResult) benchmarkSummary {
	s := benchmarkSummary{Total: len(results)}
	for _, r := range results {
		switch r.Status {
		case "pass":
			s.Passed++
		case "fail":
			s.Failed++
		default:
			s.Errors++
		}
	}
	if s.Total > 0 {
		s.Score = float64(s.Passed) / float64(s.Total)
	}
	return s
}

// writeBenchmarkResults writes the JSON report under eval/benchmark-results/
// (gitignored) and returns its path.
func writeBenchmarkResults(root, runID, provider, model string, results []benchmarkTaskResult, summary benchmarkSummary) (string, error) {
	dir := filepath.Join(root, "eval", "benchmark-results")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	path := filepath.Join(dir, runID+".json")
	run := benchmarkRunResult{
		RunID:     runID,
		Provider:  provider,
		Model:     model,
		DryRun:    false,
		Timestamp: time.Now().Format(time.RFC3339),
		Tasks:     results,
		Summary:   summary,
	}
	b, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// benchmarkRunID builds the result filename stem: timestamp + provider + model
// (non-alphanumeric characters collapsed to dashes).
func benchmarkRunID(provider, model string) string {
	return time.Now().Format("20060102T150405") + "-" + sanitizeForID(provider) + "-" + sanitizeForID(model)
}

func benchmarkImageName(runID, taskID string) string {
	return "pi-bench-" + sanitizeForID(runID) + "-" + sanitizeForID(taskID)
}

func benchmarkContainerName(runID, taskID string) string {
	return "pi-bench-" + sanitizeForID(runID) + "-" + sanitizeForID(taskID)
}

// sanitizeForID collapses characters to a Docker-safe [a-zA-Z0-9._-] set.
func sanitizeForID(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '.', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}
