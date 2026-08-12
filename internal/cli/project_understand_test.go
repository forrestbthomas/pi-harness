package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// projectUnderstandFixture builds a fake repo in t.TempDir() with known
// contents: a README with a distinctive first paragraph, three .go files with
// a known non-blank LOC total, a nested Python file, a node_modules/ junk
// directory (must be skipped), a hidden .env file (must be skipped), and a
// GitHub Actions CI workflow. HARNESS_ROOT is pointed at the fixture so
// repoRoot() resolves deterministically.
func projectUnderstandFixture(t *testing.T) (root string) {
	t.Helper()
	root = t.TempDir()
	mustWrite := func(rel, content string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("go.mod", "module example.com/demo\n\ngo 1.26\n")
	mustWrite("README.md", "# Demo Project\n\nThe pi-harness demo project understands things with style.\n\nMore details below.\n")
	mustWrite("main.go", "package main\n\nfunc main() {\n\t// entry point\n\tprintln(\"hello\")\n}\n")
	mustWrite("util.go", "package main\n\n// util counts things.\nfunc util() int {\n\treturn 42\n}\n")
	mustWrite("util_test.go", "package main\n\nfunc TestUtil(t *testing.T) {\n\t_ = util()\n}\n")
	mustWrite("cmd/tool/tool.py", "#!/usr/bin/env python3\n\ndef main() -> None:\n    print(\"tool\")\n\n\nif __name__ == \"__main__\":\n    main()\n")
	mustWrite("node_modules/junk/index.js", "// junk\n"+strings.Repeat("const x = 1;\n", 200))
	mustWrite(".github/workflows/ci.yml", "name: ci\non: [push]\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - run: go test ./...\n")
	mustWrite(".env", "LOCAL_CONFIG=1\n")
	t.Setenv("HARNESS_ROOT", root)
	return root
}

// TestRunProjectUnderstandWritesDocs is the main hermetic test: it runs
// runProjectUnderstand directly against a fixture repo and asserts the
// contents of all three docs, including deterministic LOC counts, CI
// detection, skip rules, and output hygiene (no absolute paths, no placeholder
// secrets).
func TestRunProjectUnderstandWritesDocs(t *testing.T) {
	root := projectUnderstandFixture(t)
	outDir := t.TempDir()

	code := runProjectUnderstand([]string{"--out", outDir})
	if code != 0 {
		t.Fatalf("project-understand exit = %d, want 0", code)
	}

	product := string(readFile(filepath.Join(outDir, "product.md")))
	if !strings.Contains(product, "The pi-harness demo project understands things with style.") {
		t.Fatalf("product.md missing README paragraph:\n%s", product)
	}
	if !strings.Contains(product, "**Primary stack:** Go, Python") {
		t.Fatalf("product.md missing primary stack:\n%s", product)
	}
	if !strings.Contains(product, "**Tooling/frameworks:** Go modules, GitHub Actions (CI)") {
		t.Fatalf("product.md missing framework markers:\n%s", product)
	}

	tech := string(readFile(filepath.Join(outDir, "tech.md")))
	if !strings.Contains(tech, "Go: 14 lines (3 file(s))") {
		t.Fatalf("tech.md missing Go LOC:\n%s", tech)
	}
	if !strings.Contains(tech, "Python: 5 lines") {
		t.Fatalf("tech.md missing Python LOC:\n%s", tech)
	}
	if !strings.Contains(tech, "GitHub Actions") || !strings.Contains(tech, "## CI") {
		t.Fatalf("tech.md missing CI mention:\n%s", tech)
	}
	if !strings.Contains(tech, "1 test file(s) detected (Go: 1)") {
		t.Fatalf("tech.md missing test-file count:\n%s", tech)
	}
	if strings.Contains(tech, "JavaScript") {
		t.Fatalf("tech.md must skip node_modules junk:\n%s", tech)
	}

	structure := string(readFile(filepath.Join(outDir, "structure.md")))
	for _, want := range []string{".github/", "cmd/", "tool/"} {
		if !strings.Contains(structure, want) {
			t.Fatalf("structure.md missing %q:\n%s", want, structure)
		}
	}
	if strings.Contains(structure, "node_modules") {
		t.Fatalf("structure.md must omit node_modules:\n%s", structure)
	}
	if !strings.Contains(structure, "README.md") || !strings.Contains(structure, "go.mod") {
		t.Fatalf("structure.md missing key files:\n%s", structure)
	}
	if !strings.Contains(structure, "Go module: example.com/demo") {
		t.Fatalf("structure.md missing module name:\n%s", structure)
	}

	// Output hygiene: no absolute temp paths, no placeholder secrets.
	all := product + tech + structure
	if strings.Contains(all, root) {
		t.Fatalf("output must not contain absolute root path %q:\n%s", root, all)
	}
	if strings.Contains(all, "test"+"-key") {
		t.Fatalf("output must not contain placeholder secret:\n%s", all)
	}
}

// TestRunProjectUnderstandMissingREADME verifies graceful fallback when the
// repo has no README: the command still succeeds and product.md names the repo.
func TestRunProjectUnderstandMissingREADME(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HARNESS_ROOT", root)
	outDir := t.TempDir()

	if code := runProjectUnderstand([]string{"--out", outDir}); code != 0 {
		t.Fatalf("project-understand exit = %d, want 0", code)
	}
	product := string(readFile(filepath.Join(outDir, "product.md")))
	if !strings.Contains(product, filepath.Base(root)) {
		t.Fatalf("product.md fallback must name the repo:\n%s", product)
	}
	if !strings.Contains(product, "no README description") {
		t.Fatalf("product.md fallback must note the missing README:\n%s", product)
	}
}

// TestRunProjectUnderstandDefaultOutDir verifies the default output location
// (<root>/docs/understand) is created and populated.
func TestRunProjectUnderstandDefaultOutDir(t *testing.T) {
	root := projectUnderstandFixture(t)
	if code := runProjectUnderstand(nil); code != 0 {
		t.Fatalf("project-understand exit = %d, want 0", code)
	}
	for _, name := range []string{"product.md", "tech.md", "structure.md"} {
		if _, err := os.Stat(filepath.Join(root, "docs", "understand", name)); err != nil {
			t.Fatalf("default out %s missing: %v", name, err)
		}
	}
}

// TestRunProjectUnderstandUnknownFlagExit2 verifies unknown flags return exit
// code 2 with a clear message (through the registered command path).
func TestRunProjectUnderstandUnknownFlagExit2(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	code, out := captureRunStderr(t, []string{"project-understand", "--bogus"})
	if code != 2 {
		t.Fatalf("unknown flag exit = %d, want 2; stderr: %q", code, out)
	}
	if !strings.Contains(out, "unknown flag") {
		t.Fatalf("stderr missing unknown-flag message: %q", out)
	}
}

// TestRunProjectUnderstandMissingRoot verifies scan/walk errors exit 1.
func TestRunProjectUnderstandMissingRoot(t *testing.T) {
	t.Setenv("HARNESS_ROOT", filepath.Join(t.TempDir(), "missing"))
	if code := runProjectUnderstand([]string{"--out", t.TempDir()}); code != 1 {
		t.Fatalf("missing root exit = %d, want 1", code)
	}
}

// TestRunProjectUnderstandRegistered verifies the command is wired into Run()
// and succeeds end to end.
func TestRunProjectUnderstandRegistered(t *testing.T) {
	projectUnderstandFixture(t)
	outDir := t.TempDir()
	code, _ := captureRunStdout(t, []string{"project-understand", "--out", outDir})
	if code != 0 {
		t.Fatalf("registered command exit = %d, want 0", code)
	}
	if _, err := os.Stat(filepath.Join(outDir, "product.md")); err != nil {
		t.Fatalf("product.md not written: %v", err)
	}
}

// TestRunProjectUnderstandHelp verifies --help prints usage and exits 0.
func TestRunProjectUnderstandHelp(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	code, out := captureRunStdout(t, []string{"project-understand", "--help"})
	if code != 0 {
		t.Fatalf("--help exit = %d, want 0", code)
	}
	if !strings.Contains(out, "Usage: pi-run project-understand") {
		t.Fatalf("--help output missing usage: %q", out)
	}
}
