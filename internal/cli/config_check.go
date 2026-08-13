package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// runConfigCheck runs deterministic harness checks. No keys, no network.
func runConfigCheck() int {
	root := repoRoot()
	home, _ := os.UserHomeDir()
	fail := false
	check := func(name string, ok bool) {
		mark := "[ok]  "
		if !ok {
			mark = "[FAIL]"
			fail = true
		}
		fmt.Printf("  %s %s\n", mark, name)
	}

	fmt.Println("== pi-run config-check ==")

	// 1. Project settings.
	project := filepath.Join(root, ".pi", "settings.json")
	if s, err := loadJSON(project); err == nil {
		check(".pi/settings.json valid JSON", true)
		check("defaultProvider is openai", s["defaultProvider"] == "openai")
		check("defaultModel is openai/gpt-5.6-terra", s["defaultModel"] == "openai/gpt-5.6-terra")
		if enabled, ok := s["enabledModels"].([]any); ok {
			patterns := toStrings(enabled)
			check("enabledModels has openai/* before openrouter/*",
				slices.Index(patterns, "openai/*") < slices.Index(patterns, "openrouter/*"))
		} else {
			check("enabledModels present", false)
		}
		check("pi-subagents pinned (exact version) in .pi/settings.json", piSubagentsPinned(s))
	} else {
		check(".pi/settings.json valid JSON", false)
	}
	check("no literal API keys in .pi/settings.json", !hasLiteralSecret(readFile(project)))
	// 2. Global settings (personal-machine check unless PI_RUN_PERSONAL=1).
	// A missing global ~/.pi/agent/settings.json is normal on a fresh machine
	// or CI runner — it is NOT a harness defect, so it must not fail
	// config-check; only a present-but-invalid file is a real failure.
	global := filepath.Join(home, ".pi", "agent", "settings.json")
	if _, err := os.Stat(global); err != nil {
		if personalMode() {
			fmt.Println("  [info] ~/.pi/agent/settings.json not found (personal install check; set PI_RUN_PERSONAL=1 to require)")
		} else {
			fmt.Println("  [info] ~/.pi/agent/settings.json not found (not a failure on a fresh machine/CI)")
		}
	} else if s, err := loadJSON(global); err == nil {
		check("~/.pi/agent/settings.json valid JSON", true)
		model, _ := s["defaultModel"].(string)
		check("global defaultProvider is openai", s["defaultProvider"] == "openai")
		check("global defaultModel is openai/gpt-*", strings.HasPrefix(model, "openai/") || strings.HasPrefix(model, "gpt-"))
	} else {
		check("~/.pi/agent/settings.json valid JSON", false)
	}

	// 3. Binary link (personal-machine check unless PI_RUN_PERSONAL=1).
	if personalMode() {
		link := filepath.Join(home, "bin", "pi-run")
		target, err := os.Readlink(link)
		check("pi-run symlinks to <root>/bin/pi-run",
			err == nil && target == filepath.Join(root, "bin", "pi-run"))
	} else {
		fmt.Println("  [info] symlink check skipped (set PI_RUN_PERSONAL=1 to enable)")
	}

	// 4. Dotfiles: no pi-harness functions, no static secret exports
	//    (personal-machine checks unless PI_RUN_PERSONAL=1).
	if personalMode() {
		secretVars := []string{"OPENROUTER_API_KEY", "OPENAI_API_KEY", "DEEPSEEK_API_KEY", "KIMI_API_KEY"}
		for _, rc := range []string{filepath.Join(home, ".zshrc"), filepath.Join(home, ".bashrc")} {
			text := readFile(rc)
			name := filepath.Base(rc)
			check(name+" has no pi-harness functions", !strings.Contains(string(text), "pi-harness()"))
			for _, v := range secretVars {
				for line := range strings.SplitSeq(string(text), "\n") {
					if strings.HasPrefix(strings.TrimSpace(line), "export "+v+"=") && !strings.Contains(line, "bw_get") {
						check(name+" has no static "+v+" export", false)
					}
				}
			}
		}
	} else {
		fmt.Println("  [info] dotfile checks skipped (set PI_RUN_PERSONAL=1 to enable)")
	}

	// 5. Makefile gone.
	check("Makefile removed", !pathExists(filepath.Join(root, "Makefile")))

	// 6. Skills.
	spDir := filepath.Join(home, ".agents", "skills")
	count := 0
	if entries, err := os.ReadDir(spDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				count++
			}
		}
	}
	// Personal-machine skill checks: informational unless PI_RUN_PERSONAL=1 so
	// config-check passes on a fresh clone that has not installed the curated
	// skill collections.
	if personalMode() {
		check("superpowers skills installed (>= 14)", count >= 14)
		check("agent-skills clone present", pathExists(filepath.Join(home, "Projects", "tmp", "agent-skills", "skills")))
	} else {
		fmt.Println("  [info] superpowers/agent-skills checks skipped (set PI_RUN_PERSONAL=1 to enable)")
	}

	// 7. providers.json: the repo tier maps must pass validateTiers
	//    (deterministic; no keys, no network). A missing file means the
	//    embedded defaults apply (valid by construction); only a present-but-
	//    invalid table fails.
	tiersOK := true
	if b := readFile(filepath.Join(root, "providers.json")); len(b) > 0 {
		if ps, err := ProvidersFromJSON(b); err != nil {
			tiersOK = false
		} else {
			for _, p := range ps {
				if err := validateTiers(p); err != nil {
					tiersOK = false
					break
				}
			}
		}
	}
	check("providers.json modelTiers valid", tiersOK)

	if fail {
		fmt.Println("== config-check: FAILURES FOUND ==")
		return 1
	}
	fmt.Println("== config-check: all checks passed ==")
	return 0
}

func loadJSON(path string) (map[string]any, error) {
	var m map[string]any
	err := json.Unmarshal(readFile(path), &m)
	return m, err
}

func readFile(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return b
}

func toStrings(v []any) []string {
	out := make([]string, 0, len(v))
	for _, x := range v {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

var secretRe = regexp.MustCompile(`sk-[A-Za-z0-9_-]{8,}`)

func hasLiteralSecret(b []byte) bool {
	return secretRe.Match(b)
}

// piSubagentsPinned reports whether the project settings pin pi-subagents to an
// exact version (npm:pi-subagents@<semver>) so a settings-triggered npm refresh
// cannot silently drift the subagent tooling (BACKLOG #2).
func piSubagentsPinned(s map[string]any) bool {
	pkgs, ok := s["packages"].([]any)
	if !ok {
		return false
	}
	for _, p := range pkgs {
		spec, ok := p.(string)
		if !ok || !strings.HasPrefix(spec, "npm:pi-subagents") {
			continue
		}
		rest := strings.TrimPrefix(spec, "npm:pi-subagents")
		// Require @<exact semver>: a bare name or a ^/~ range is not pinned.
		if len(rest) < 2 || rest[0] != '@' {
			return false
		}
		v := rest[1:]
		if v == "" || strings.ContainsAny(v, "^~") {
			return false
		}
		return v[0] >= '0' && v[0] <= '9'
	}
	return false
}
