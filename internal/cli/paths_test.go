package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestNoHardcodedUserPaths ensures no shipped file references the original
// developer's username or the old module path. Walk only shipped dirs (skip
// historical docs, vendored/ignored dirs) so the harness is portable.
func TestNoHardcodedUserPaths(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// file = <root>/internal/cli/paths_test.go
	root := filepath.Dir(filepath.Dir(filepath.Dir(file)))

	// Directories/files to scan (relative to root). Historical docs under
	// docs/superpowers/ legitimately reference the old path in plans/specs and
	// are not shipped runtime artifacts, so they are excluded here. The Python
	// harness-config test asserts the old path's absence itself, so it is
	// excluded to avoid a false positive (it references the literal only to
	// check it is gone).
	scanDirs := []string{
		filepath.Join(root, "cmd"),
		filepath.Join(root, "internal"),
		filepath.Join(root, "scripts"),
		filepath.Join(root, "eval"),
		filepath.Join(root, ".pi"),
	}
	// Files to skip inside scanned dirs (paths relative to root).
	// eval/tests/test_harness_config.py legitimately contains the old path
	// literal only to assert its absence.
	skipFiles := map[string]bool{
		"eval/tests/test_harness_config.py": true,
	}

	// loreMaintainedSection returns the content of b with the lore-managed
	// knowledge block (from the "maintained by the coding agent via lore"
	// marker to EOF) removed. AGENTS.md is scanned for hardcoded paths in its
	// human-written instructions, but its lore block is auto-generated
	// documentation that intentionally references historical paths and the old
	// module path (e.g. the Open-source migration decision) — those are facts
	// about the project's history, not shipped runtime artifacts, so they must
	// not fail the portability scan.
	loreMaintainedSection := func(b []byte) []byte {
		const marker = "<!-- This section is maintained by the coding agent via lore"
		if before, _, ok0 := bytes.Cut(b, []byte(marker)); ok0 {
			return before
		}
		return b
	}
	scanFiles := []string{
		filepath.Join(root, "README.md"),
		filepath.Join(root, "AGENTS.md"),
		filepath.Join(root, "LICENSE"),
		filepath.Join(root, "go.mod"),
	}

	skipDirs := map[string]bool{
		".git": true, ".worktrees": true, "bin": true, "sessions": true,
		".venv": true, "__pycache__": true, ".pytest_cache": true, "node_modules": true,
	}

	check := func(path, rel string) {
		b, err := os.ReadFile(path)
		if err != nil {
			return
		}
		text := string(loreMaintainedSection(b))
		if strings.Contains(text, "/Users/forrestthomas"+"/") {
			t.Errorf("hardcoded user path in %s", rel)
		}
		if strings.Contains(text, "github.com/forrestthomas/"+"harness") {
			t.Errorf("old module path in %s", rel)
		}
	}

	for _, d := range scanDirs {
		filepath.WalkDir(d, func(path string, ent os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			if skipFiles[rel] {
				if ent.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if ent.IsDir() {
				if skipDirs[ent.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			check(path, rel)
			return nil
		})
	}
	for _, f := range scanFiles {
		if fi, err := os.Stat(f); err == nil && !fi.IsDir() {
			rel, _ := filepath.Rel(root, f)
			check(f, rel)
		}
	}
}
