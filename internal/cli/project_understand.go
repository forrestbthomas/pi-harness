package cli

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// projectUnderstandDefaultOut is the default output directory for
// project-understand docs, relative to the scanned repo root.
const projectUnderstandDefaultOut = "docs/understand"

// projectUnderstandUsage is the CLI help for `pi-run project-understand`.
const projectUnderstandUsage = `Usage: pi-run project-understand [--out <dir>]

Generate deterministic project-understanding docs (product.md, tech.md,
structure.md) from the current repo checkout. No network, no LLM calls.

Flags:
  --out <dir>   Write docs into <dir> (default <root>/docs/understand)
  -h, --help    Show this help
`

// projectUnderstandSkipDirs are directory names pruned from the scan and the
// structure tree (dependency, build, and harness noise). "eval/.venv" is
// covered by the ".venv" entry.
var projectUnderstandSkipDirs = map[string]bool{
	".git":          true,
	"node_modules":  true,
	".venv":         true,
	".pi":           true,
	".worktrees":    true,
	"bin":           true,
	"dist":          true,
	"build":         true,
	"__pycache__":   true,
	".pytest_cache": true,
}

// languageByExt maps file extensions to census display names. Extensions are
// matched case-insensitively.
var languageByExt = map[string]string{
	".go":    "Go",
	".py":    "Python",
	".js":    "JavaScript",
	".ts":    "TypeScript",
	".jsx":   "JSX",
	".tsx":   "TSX",
	".rs":    "Rust",
	".java":  "Java",
	".rb":    "Ruby",
	".sh":    "Shell",
	".md":    "Markdown",
	".json":  "JSON",
	".yaml":  "YAML",
	".yml":   "YAML",
	".html":  "HTML",
	".css":   "CSS",
	".sql":   "SQL",
	".c":     "C",
	".h":     "C header",
	".cpp":   "C++",
	".hpp":   "C++ header",
	".cs":    "C#",
	".kt":    "Kotlin",
	".swift": "Swift",
	".php":   "PHP",
}

// docFormatLangs are excluded from the product.md "primary stack" summary:
// documentation and config formats are not a runtime stack.
var docFormatLangs = map[string]bool{
	"Markdown": true,
	"JSON":     true,
	"YAML":     true,
}

// languageStats aggregates the non-blank line census for one language.
type languageStats struct {
	Name  string
	Lines int
	Files int
}

// stackMeta is a detected framework/tooling marker and its evidence path.
type stackMeta struct {
	Name     string
	Evidence string
}

// projectScan holds everything collected from a single repo walk.
type projectScan struct {
	languages       map[string]*languageStats
	testFiles       map[string]int // language name -> test file count
	workflows       []string       // relative paths of .github/workflows/*.yml
	hasGoMod        bool
	hasPackageJSON  bool
	hasPyProject    bool
	hasRequirements bool
	hasCargo        bool
	hasDockerfile   bool
}

// runProjectUnderstand implements `pi-run project-understand`. Exit codes:
// 0 success, 1 scan/write error, 2 usage error.
func runProjectUnderstand(args []string) int {
	outDir := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--help" || a == "-h":
			fmt.Print(projectUnderstandUsage)
			return 0
		case a == "--out" && i+1 < len(args):
			outDir = args[i+1]
			i++
		case strings.HasPrefix(a, "--out="):
			outDir = strings.TrimPrefix(a, "--out=")
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(os.Stderr, "pi-run: project-understand: unknown flag %q\n", a)
			return 2
		default:
			fmt.Fprintf(os.Stderr, "pi-run: project-understand: unexpected argument %q\n", a)
			return 2
		}
	}

	root := repoRoot()
	if outDir == "" {
		outDir = filepath.Join(root, projectUnderstandDefaultOut)
	}

	// Scan first so previously generated docs (in the default out dir) never
	// pollute the census.
	scan, err := scanProject(root, outDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: project-understand: %v\n", err)
		return 1
	}
	tree, err := projectTree(root, 2, outDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: project-understand: %v\n", err)
		return 1
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: project-understand: %v\n", err)
		return 1
	}
	docs := map[string]string{
		"product.md":   productDoc(root, scan),
		"tech.md":      techDoc(scan),
		"structure.md": structureDoc(root, tree),
	}
	for name, content := range docs {
		if err := os.WriteFile(filepath.Join(outDir, name), []byte(content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "pi-run: project-understand: write %s: %v\n", name, err)
			return 1
		}
	}
	fmt.Printf("wrote product.md, tech.md, structure.md to %s\n", outDir)
	return 0
}

// scanProject walks root and aggregates the language census, test-file counts,
// CI workflows, and framework markers. The output dir (when inside root) is
// pruned so previously generated docs never pollute the census.
func scanProject(root, outDir string) (*projectScan, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve repo root: %w", err)
	}
	absOut, _ := filepath.Abs(outDir)
	relOut, relErr := filepath.Rel(absRoot, absOut)
	pruneOut := relErr == nil && relOut != "." && relOut != ".." &&
		!strings.HasPrefix(relOut, ".."+string(filepath.Separator))

	s := &projectScan{
		languages: map[string]*languageStats{},
		testFiles: map[string]int{},
	}
	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if d.IsDir() {
			if path != absRoot {
				if strings.HasPrefix(d.Name(), ".") && d.Name() != ".github" {
					return filepath.SkipDir
				}
				if projectUnderstandSkipDirs[d.Name()] {
					return filepath.SkipDir
				}
				if pruneOut && filepath.Clean(path) == filepath.Clean(absOut) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil // hidden files are noise
		}
		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		if !strings.Contains(relSlash, "/") {
			switch d.Name() {
			case "go.mod":
				s.hasGoMod = true
			case "package.json":
				s.hasPackageJSON = true
			case "pyproject.toml":
				s.hasPyProject = true
			case "requirements.txt":
				s.hasRequirements = true
			case "Cargo.toml":
				s.hasCargo = true
			case "Dockerfile":
				s.hasDockerfile = true
			}
		}
		if lang, ok := languageByExt[strings.ToLower(filepath.Ext(d.Name()))]; ok {
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			st := s.languages[lang]
			if st == nil {
				st = &languageStats{Name: lang}
				s.languages[lang] = st
			}
			st.Lines += countNonBlankLines(b)
			st.Files++
			if isProjectTestFile(d.Name()) {
				s.testFiles[lang]++
			}
		}
		if strings.HasPrefix(relSlash, ".github/workflows/") &&
			(strings.HasSuffix(relSlash, ".yml") || strings.HasSuffix(relSlash, ".yaml")) {
			s.workflows = append(s.workflows, relSlash)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(s.workflows)
	return s, nil
}

// countNonBlankLines returns the number of lines in b with non-whitespace
// content.
func countNonBlankLines(b []byte) int {
	n := 0
	for line := range strings.Lines(string(b)) {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}

// isProjectTestFile reports whether a filename matches a common test-file
// convention across the supported languages.
func isProjectTestFile(name string) bool {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, "_test.go"),
		strings.HasSuffix(lower, "_test.py"),
		strings.HasSuffix(lower, "_spec.py"),
		strings.HasSuffix(lower, "_test.rs"),
		strings.HasSuffix(lower, "_test.rb"),
		strings.HasSuffix(lower, "_spec.rb"),
		strings.HasSuffix(lower, "_test.php"),
		strings.HasPrefix(lower, "test_"):
		return true
	case strings.Contains(lower, ".test."), strings.Contains(lower, ".spec."):
		return true
	case strings.HasSuffix(name, "Test.java"),
		strings.HasSuffix(name, "Tests.java"),
		strings.HasSuffix(name, "Test.kt"),
		strings.HasSuffix(name, "Test.cs"):
		return true
	}
	return false
}

// sortedLanguages returns the census ordered by non-blank lines descending,
// then by name.
func (s *projectScan) sortedLanguages() []*languageStats {
	out := make([]*languageStats, 0, len(s.languages))
	for _, st := range s.languages {
		out = append(out, st)
	}
	slices.SortFunc(out, func(a, b *languageStats) int {
		if c := cmp.Compare(b.Lines, a.Lines); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})
	return out
}

// frameworks returns the detected framework/tooling markers in a fixed,
// deterministic order.
func (s *projectScan) frameworks() []stackMeta {
	var out []stackMeta
	if s.hasGoMod {
		out = append(out, stackMeta{Name: "Go modules", Evidence: "go.mod"})
	}
	if s.hasPackageJSON {
		out = append(out, stackMeta{Name: "npm/Node.js", Evidence: "package.json"})
	}
	if s.hasPyProject {
		out = append(out, stackMeta{Name: "Python packaging", Evidence: "pyproject.toml"})
	}
	if s.hasRequirements {
		out = append(out, stackMeta{Name: "Python (pip)", Evidence: "requirements.txt"})
	}
	if s.hasCargo {
		out = append(out, stackMeta{Name: "Cargo/Rust", Evidence: "Cargo.toml"})
	}
	if s.hasDockerfile {
		out = append(out, stackMeta{Name: "Docker", Evidence: "Dockerfile"})
	}
	if len(s.workflows) > 0 {
		out = append(out, stackMeta{Name: "GitHub Actions (CI)", Evidence: ".github/workflows"})
	}
	return out
}

// primaryStack returns the top three non-doc languages by LOC for the
// product.md summary.
func primaryStack(s *projectScan) []string {
	var stack []*languageStats
	for name, st := range s.languages {
		if !docFormatLangs[name] {
			stack = append(stack, st)
		}
	}
	slices.SortFunc(stack, func(a, b *languageStats) int {
		if c := cmp.Compare(b.Lines, a.Lines); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})
	names := make([]string, 0, min(3, len(stack)))
	for i := 0; i < len(stack) && i < 3; i++ {
		names = append(names, stack[i].Name)
	}
	return names
}

// readmeSummary returns the first meaningful paragraph of the root README:
// the first run of non-blank lines that contains at least one non-heading
// line, with leading heading lines stripped and the remainder joined with
// spaces. Returns "" when there is no README or no such paragraph.
func readmeSummary(root string) string {
	matches, err := filepath.Glob(filepath.Join(root, "README*"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	slices.Sort(matches)
	b := readFile(matches[0])
	if len(b) == 0 {
		return ""
	}
	var para []string
	for line := range strings.Lines(string(b)) {
		t := strings.TrimSpace(line)
		if t == "" {
			if summary := joinParagraph(para); summary != "" {
				return summary
			}
			para = nil
			continue
		}
		para = append(para, t)
	}
	return joinParagraph(para)
}

// joinParagraph strips leading markdown heading/rule lines from a paragraph
// and joins the remaining lines with single spaces.
func joinParagraph(lines []string) string {
	i := 0
	for i < len(lines) && isMarkdownHeading(lines[i]) {
		i++
	}
	if i >= len(lines) {
		return ""
	}
	return strings.Join(lines[i:], " ")
}

// isMarkdownHeading reports whether a trimmed line is pure markdown heading or
// horizontal-rule noise.
func isMarkdownHeading(t string) bool {
	return strings.HasPrefix(t, "#") ||
		strings.HasPrefix(t, "---") ||
		strings.HasPrefix(t, "===")
}

// productDoc renders product.md: what the project is.
func productDoc(root string, s *projectScan) string {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(filepath.Base(root))
	b.WriteString(" — What This Project Is\n\n")
	summary := readmeSummary(root)
	if summary == "" {
		summary = fmt.Sprintf("A repository named %q with no README description; see tech.md and structure.md for details.", filepath.Base(root))
	}
	b.WriteString(summary)
	b.WriteString("\n\n**Primary stack:** ")
	stack := primaryStack(s)
	if len(stack) == 0 {
		b.WriteString("none detected")
	} else {
		b.WriteString(strings.Join(stack, ", "))
	}
	b.WriteString("\n")
	if fw := s.frameworks(); len(fw) > 0 {
		b.WriteString("**Tooling/frameworks:** ")
		for i, f := range fw {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(f.Name)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// techDoc renders tech.md: how the project is built.
func techDoc(s *projectScan) string {
	var b strings.Builder
	b.WriteString("# Tech Stack — How It's Built\n\n")
	b.WriteString("## Languages (non-blank LOC, descending)\n\n")
	if len(s.languages) == 0 {
		b.WriteString("No source files detected.\n")
	} else {
		for _, st := range s.sortedLanguages() {
			fmt.Fprintf(&b, "- %s: %d lines (%d file(s))\n", st.Name, st.Lines, st.Files)
		}
	}
	b.WriteString("\n## Frameworks & tooling\n\n")
	fw := s.frameworks()
	if len(fw) == 0 {
		b.WriteString("None detected.\n")
	} else {
		for _, f := range fw {
			b.WriteString("- ")
			b.WriteString(f.Name)
			b.WriteString(" (")
			b.WriteString(f.Evidence)
			b.WriteString(")\n")
		}
	}
	b.WriteString("\n## Tests\n\n")
	if len(s.testFiles) == 0 {
		b.WriteString("No test files detected.\n")
	} else {
		langs := make([]string, 0, len(s.testFiles))
		total := 0
		for name, n := range s.testFiles {
			langs = append(langs, name)
			total += n
		}
		slices.Sort(langs)
		fmt.Fprintf(&b, "%d test file(s) detected", total)
		for _, name := range langs {
			fmt.Fprintf(&b, " (%s: %d)", name, s.testFiles[name])
		}
		b.WriteString("\n")
	}
	b.WriteString("\n## CI\n\n")
	if len(s.workflows) == 0 {
		b.WriteString("No CI workflow detected.\n")
	} else {
		b.WriteString("GitHub Actions workflows:\n")
		for _, w := range s.workflows {
			b.WriteString("- ")
			b.WriteString(w)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// projectDirNode is one directory entry in the structure tree.
type projectDirNode struct {
	name       string
	dirs       []*projectDirNode
	hasContent bool
}

// projectTree builds a pruned directory tree up to depth levels deep: hidden
// entries (except .github), skip-list directories, and the output dir are
// pruned, and directories with no visible content are dropped.
func projectTree(dir string, depth int, outDir string) (*projectDirNode, error) {
	absOut, _ := filepath.Abs(outDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	node := &projectDirNode{name: filepath.Base(dir)}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") && name != ".github" {
			continue
		}
		if filepath.Clean(filepath.Join(dir, name)) == filepath.Clean(absOut) {
			continue
		}
		if e.IsDir() {
			if projectUnderstandSkipDirs[name] {
				continue
			}
			node.hasContent = true
			if depth > 0 {
				child, err := projectTree(filepath.Join(dir, name), depth-1, outDir)
				if err != nil || !child.hasContent {
					continue // unreadable or empty subdir: prune
				}
				node.dirs = append(node.dirs, child)
			}
		} else {
			node.hasContent = true
		}
	}
	slices.SortFunc(node.dirs, func(a, b *projectDirNode) int { return cmp.Compare(a.name, b.name) })
	return node, nil
}

// render writes the tree as an indented box-drawing listing.
func (n *projectDirNode) render(b *strings.Builder, indent string) {
	for i, d := range n.dirs {
		if i == len(n.dirs)-1 {
			b.WriteString(indent)
			b.WriteString("└── ")
			b.WriteString(d.name)
			b.WriteString("/\n")
			d.render(b, indent+"    ")
		} else {
			b.WriteString(indent)
			b.WriteString("├── ")
			b.WriteString(d.name)
			b.WriteString("/\n")
			d.render(b, indent+"│   ")
		}
	}
}

// structureDoc renders structure.md: where things live.
func structureDoc(root string, tree *projectDirNode) string {
	var b strings.Builder
	b.WriteString("# Project Structure — Where Things Live\n\n")
	b.WriteString("## Top-level directories\n\n")
	if tree == nil || len(tree.dirs) == 0 {
		b.WriteString("_no directories_\n")
	} else {
		b.WriteString("```\n")
		b.WriteString(filepath.Base(root))
		b.WriteString("/\n")
		tree.render(&b, "")
		b.WriteString("```\n")
	}
	b.WriteString("\n## Key files\n\n")
	keys := keyFilesAt(root)
	if len(keys) == 0 {
		b.WriteString("_none_\n")
	} else {
		for _, k := range keys {
			b.WriteString("- ")
			b.WriteString(k)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n## Packages / modules\n\n")
	mods := projectModules(root)
	if len(mods) == 0 {
		b.WriteString("_none detected_\n")
	} else {
		for _, m := range mods {
			b.WriteString("- ")
			b.WriteString(m)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// keyFilesAt lists the key root-level files present in the repo.
func keyFilesAt(root string) []string {
	var keys []string
	for _, pattern := range []string{"README*", "LICENSE*", "Makefile", "go.mod"} {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			continue
		}
		for _, m := range matches {
			keys = append(keys, filepath.Base(m))
		}
	}
	slices.Sort(keys)
	return keys
}

// projectModules extracts package/module names from root-level manifests.
func projectModules(root string) []string {
	var mods []string
	if b := readFile(filepath.Join(root, "go.mod")); len(b) > 0 {
		for line := range strings.Lines(string(b)) {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "module ") {
				fields := strings.Fields(t)
				if len(fields) >= 2 {
					mods = append(mods, "Go module: "+fields[1])
				}
				break
			}
		}
	}
	if b := readFile(filepath.Join(root, "package.json")); len(b) > 0 {
		if name := quotedField(b, "name"); name != "" {
			mods = append(mods, "npm package: "+name)
		}
	}
	if b := readFile(filepath.Join(root, "pyproject.toml")); len(b) > 0 {
		if name := tomlQuotedField(b, "name"); name != "" {
			mods = append(mods, "Python project: "+name)
		}
	}
	if b := readFile(filepath.Join(root, "Cargo.toml")); len(b) > 0 {
		if name := tomlQuotedField(b, "name"); name != "" {
			mods = append(mods, "Cargo crate: "+name)
		}
	}
	return mods
}

// quotedField extracts the value of a double-quoted JSON-style field, e.g.
// `"name": "value"`, tolerating comments and trailing commas that strict
// json.Unmarshal rejects.
func quotedField(b []byte, field string) string {
	s := string(b)
	idx := strings.Index(s, `"`+field+`"`)
	if idx < 0 {
		return ""
	}
	rest := s[idx+len(field)+2:]
	if colon := strings.Index(rest, ":"); colon >= 0 {
		rest = strings.TrimSpace(rest[colon+1:])
		if strings.HasPrefix(rest, `"`) {
			if end := strings.Index(rest[1:], `"`); end >= 0 {
				return rest[1 : 1+end]
			}
		}
	}
	return ""
}

// tomlQuotedField extracts a TOML `name = "value"` assignment.
func tomlQuotedField(b []byte, field string) string {
	for line := range strings.Lines(string(b)) {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, field) {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(t, field))
		if !strings.HasPrefix(rest, "=") {
			continue
		}
		rest = strings.TrimSpace(strings.TrimPrefix(rest, "="))
		rest = strings.Trim(rest, `"`)
		if rest != "" && !strings.Contains(rest, "[") {
			return rest
		}
	}
	return ""
}
