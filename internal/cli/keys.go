package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// resolveSecret returns the value of env var name, falling back to Bitwarden
// via bw_get (BW_GET env override; default ~/bin/bw_get). Never logs values.
func resolveSecret(name string) (string, error) {
	if v := os.Getenv(name); v != "" {
		return v, nil
	}
	bwGet := os.Getenv("BW_GET")
	if bwGet == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve %s: cannot find home dir: %w", name, err)
		}
		bwGet = filepath.Join(home, "bin", "bw_get")
	}
	var out bytes.Buffer
	cmd := exec.Command(bwGet, name)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("resolve %s: bw_get failed (vault locked? run `bw unlock`): %w", name, err)
	}
	v := strings.TrimSpace(out.String())
	if v == "" {
		return "", fmt.Errorf("resolve %s: bw_get returned an empty value", name)
	}
	return v, nil
}
