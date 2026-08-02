package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runDoctor reports harness health. Exits 1 if any hard check fails.
func runDoctor() int {
	root := repoRoot()
	fail := false
	check := func(name string, ok bool) {
		mark := "[ok]  "
		if !ok {
			mark = "[FAIL]"
			fail = true
		}
		fmt.Printf("  %s %s\n", mark, name)
	}

	fmt.Println("== pi-run doctor ==")

	home, _ := os.UserHomeDir()
	nodeVersion := os.Getenv("PI_NODE_VERSION")
	if nodeVersion == "" {
		nodeVersion = "v22.19.0"
	}
	_, err := nodeBinDir(home, nodeVersion)
	check("node "+nodeVersion, err == nil)

	piPath := filepath.Join(home, ".nvm", "versions", "node", nodeVersion, "bin", "pi")
	check("pi CLI present", pathExists(piPath))

	// Vault status (informational; never a value).
	if out, err := exec.Command("bw_get", "--status").Output(); err == nil {
		fmt.Printf("  [info] Bitwarden vault: %s", out)
	} else {
		check("Bitwarden vault reachable", false)
	}

	// Key presence per provider (never the value).
	for _, p := range Providers {
		if _, err := resolveSecret(p.KeyEnv); err == nil {
			check(p.KeyEnv+" present", true)
		} else {
			check(p.KeyEnv+" present", false)
		}
	}

	// Default models resolvable (informational; offline; PATH via execPi).
	if out, err := piListModels(home, nodeVersion); err == nil {
		for _, p := range Providers {
			if p.Name == "openrouter" {
				continue // openrouter default model lives in the openai catalog
			}
			fmt.Printf("  [info] model %s resolvable: %v\n", p.DefaultModel, modelListed(string(out), p.DefaultModel))
		}
	} else {
		fmt.Printf("  [info] pi --offline --list-models unavailable: %v\n", err)
	}

	venv := filepath.Join(root, "eval", ".venv", "bin", "python")
	check("eval/.venv present", pathExists(venv))

	link := filepath.Join(home, "bin", "pi-run")
	check("~/bin/pi-run symlink present", pathExists(link))

	if fail {
		fmt.Println("== doctor: FAILURES FOUND ==")
		return 1
	}
	fmt.Println("== doctor: all checks passed ==")
	return 0
}

// piListModels runs `pi --offline --list-models` with the nvm node bin dir on
// PATH and returns its stdout.
func piListModels(home, nodeVersion string) ([]byte, error) {
	binDir, err := nodeBinDir(home, nodeVersion)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(filepath.Join(binDir, "pi"), "--offline", "--list-models")
	cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return cmd.Output()
}

func pathExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// modelListed reports whether the model id (suffix after the last "/") appears
// in the `pi --offline --list-models` output, whose columnar format puts the
// provider and model id in separate columns (e.g. "deepseek  deepseek-v4-flash").
func modelListed(out, model string) bool {
	suffix := model
	if i := strings.LastIndex(suffix, "/"); i >= 0 {
		suffix = suffix[i+1:]
	}
	return suffix != "" && strings.Contains(out, suffix)
}
