package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
				firstIndexOf(patterns, "openai/*") < firstIndexOf(patterns, "openrouter/*"))
		} else {
			check("enabledModels present", false)
		}
	} else {
		check(".pi/settings.json valid JSON", false)
	}
	check("no literal API keys in .pi/settings.json", !hasLiteralSecret(readFile(project)))

	// 2. Global settings.
	global := filepath.Join(home, ".pi", "agent", "settings.json")
	if s, err := loadJSON(global); err == nil {
		check("~/.pi/agent/settings.json valid JSON", true)
		model, _ := s["defaultModel"].(string)
		check("global defaultProvider is openai", s["defaultProvider"] == "openai")
		check("global defaultModel is openai/gpt-*", strings.HasPrefix(model, "openai/") || strings.HasPrefix(model, "gpt-"))
	} else {
		check("~/.pi/agent/settings.json valid JSON", false)
	}

	// 3. Binary link.
	link := filepath.Join(home, "bin", "pi-run")
	target, err := os.Readlink(link)
	check("~/bin/pi-run symlinks to <root>/bin/pi-run",
		err == nil && target == filepath.Join(root, "bin", "pi-run"))

	// 4. Dotfiles: no pi-harness functions, no static secret exports.
	secretVars := []string{"OPENROUTER_API_KEY", "OPENAI_API_KEY", "DEEPSEEK_API_KEY", "KIMI_API_KEY"}
	for _, rc := range []string{filepath.Join(home, ".zshrc"), filepath.Join(home, ".bashrc")} {
		text := readFile(rc)
		name := filepath.Base(rc)
		check(name+" has no pi-harness functions", !strings.Contains(string(text), "pi-harness()"))
		for _, v := range secretVars {
			for _, line := range strings.Split(string(text), "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "export "+v+"=") && !strings.Contains(line, "bw_get") {
					check(name+" has no static "+v+" export", false)
				}
			}
		}
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
	check("superpowers skills installed (>= 14)", count >= 14)
	check("agent-skills clone present", pathExists(filepath.Join(home, "Projects", "tmp", "agent-skills", "skills")))

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

func firstIndexOf(list []string, want string) int {
	for i, s := range list {
		if s == want {
			return i
		}
	}
	return len(list) // not found sorts last
}

var secretRe = regexp.MustCompile(`sk-[A-Za-z0-9_-]{8,}`)

func hasLiteralSecret(b []byte) bool {
	return secretRe.Match(b)
}
