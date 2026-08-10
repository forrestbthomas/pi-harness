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
	nodeVersion, err := resolveNodeVersion(home)
	if err != nil {
		check("node (latest nvm)", false)
		fmt.Println("  [info] " + err.Error())
		nodeVersion = ""
	}
	if nodeVersion != "" {
		_, err := nodeBinDir(home, nodeVersion)
		check("node "+nodeVersion, err == nil)

		piPath := filepath.Join(home, ".nvm", "versions", "node", nodeVersion, "bin", "pi")
		check("pi CLI present", pathExists(piPath))
	}

	// Secret backend status is informational: users may supply a key directly
	// through the environment and do not need a configured secret manager.
	be, err := newSecretBackend()
	if err != nil {
		fmt.Println("  [info] secret backend: unavailable")
	} else if status, err := be.Status(); err == nil {
		fmt.Printf("  [info] %s backend: %s\n", be.Name(), status)
	} else {
		fmt.Printf("  [info] %s backend: unavailable\n", be.Name())
	}

	// Key presence per provider is informational: only one is needed to launch
	// chat or run live evaluations, and a fresh installation has none yet.
	anyKey := false
	for _, p := range Providers {
		if _, err := resolveSecret(p.KeyEnv); err == nil {
			anyKey = true
			fmt.Printf("  [info] %s: key present\n", p.KeyEnv)
		} else {
			fmt.Printf("  [info] %s: not configured\n", p.KeyEnv)
		}
	}
	if !anyKey {
		fmt.Println("  [info] no provider key configured — set a key (env or secret manager) to run chat/live evals")
	}

	// Default models resolvable (informational; offline; PATH via execPi).
	if nodeVersion != "" {
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
	}

	venv := filepath.Join(root, "eval", ".venv", "bin", "python")
	check("eval/.venv present", pathExists(venv))

	// Symlink onto PATH is an install convention; informational
	// unless PI_RUN_PERSONAL=1 so doctor passes on a fresh clone.
	if personalMode() {
		link := filepath.Join(home, "bin", "pi-run")
		check("pi-run symlink present", pathExists(link))
	} else {
		fmt.Println("  [info] pi-run symlink check skipped (set PI_RUN_PERSONAL=1 to enable)")
	}

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
	cmd.Env = childEnv(binDir, nil)
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
