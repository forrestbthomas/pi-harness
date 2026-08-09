package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SecretBackend resolves API keys from a secret manager. Implementations must
// never log or print secret values — only presence and status.
type SecretBackend interface {
	Name() string
	// Resolve returns the secret value for name, or an error if unavailable.
	Resolve(name string) (string, error)
	// Status returns a human-readable backend status for pi-run doctor.
	Status() (string, error)
}

// newSecretBackend returns the configured SecretBackend, selected by the
// PI_SECRET_BACKEND env var. The empty string means "bitwarden" (backward
// compatible with the pre-pluggable behavior).
func newSecretBackend() (SecretBackend, error) {
	switch os.Getenv("PI_SECRET_BACKEND") {
	case "", "bitwarden":
		return &bitwardenBackend{}, nil
	case "1password", "op":
		return &onePasswordBackend{}, nil
	case "env-only", "env":
		return &envOnlyBackend{}, nil
	default:
		return nil, fmt.Errorf("unknown secret backend %q (want bitwarden, 1password, or env-only)", os.Getenv("PI_SECRET_BACKEND"))
	}
}

// bitwardenBackend resolves via bw_get (BW_GET override; default ~/bin/bw_get).
type bitwardenBackend struct{}

func (b *bitwardenBackend) Name() string { return "bitwarden" }

func (b *bitwardenBackend) bwGetPath() (string, error) {
	if p := os.Getenv("BW_GET"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find home dir: %w", err)
	}
	return filepath.Join(home, "bin", "bw_get"), nil
}

func (b *bitwardenBackend) Resolve(name string) (string, error) {
	p, err := b.bwGetPath()
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", name, err)
	}
	var out bytes.Buffer
	cmd := exec.Command(p, name)
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

func (b *bitwardenBackend) Status() (string, error) {
	p, err := b.bwGetPath()
	if err != nil {
		return "", err
	}
	out, err := exec.Command(p, "--status").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// onePasswordBackend resolves via `op read "op://<Vault>/<name>/credential"`.
// The op CLI must be installed and signed in.
type onePasswordBackend struct{}

func (b *onePasswordBackend) Name() string { return "1password" }

func (b *onePasswordBackend) Resolve(name string) (string, error) {
	vault := os.Getenv("OP_VAULT")
	if vault == "" {
		vault = "Personal"
	}
	ref := fmt.Sprintf("op://%s/%s/credential", vault, name)
	var out bytes.Buffer
	cmd := exec.Command("op", "read", ref)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("resolve %s: op read failed (is 1Password CLI signed in? run `op signin`): %w", name, err)
	}
	v := strings.TrimSpace(out.String())
	if v == "" {
		return "", fmt.Errorf("resolve %s: op read returned an empty value", name)
	}
	return v, nil
}

func (b *onePasswordBackend) Status() (string, error) {
	out, err := exec.Command("op", "account", "list").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// envOnlyBackend resolves only from the environment; no fallback.
type envOnlyBackend struct{}

func (b *envOnlyBackend) Name() string { return "env-only" }

func (b *envOnlyBackend) Resolve(name string) (string, error) {
	if v := os.Getenv(name); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("resolve %s: not set in environment (env-only backend)", name)
}

func (b *envOnlyBackend) Status() (string, error) {
	return "env-only (no secret manager)", nil
}
