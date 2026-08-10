package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// secretCmdTimeout bounds secret-manager helpers so a locked or hung backend
// cannot indefinitely block chat, print, resume, or doctor. It is a variable
// only so hermetic tests can use a short timeout.
var secretCmdTimeout = 30 * time.Second

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
		return "", fmt.Errorf("%s: secret lookup failed", b.Name())
	}
	out, err := runSecretCommand(b.Name(), "lookup", p, name)
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return "", fmt.Errorf("%s: secret lookup returned an empty value", b.Name())
	}
	return v, nil
}

func (b *bitwardenBackend) Status() (string, error) {
	p, err := b.bwGetPath()
	if err != nil {
		return "", fmt.Errorf("%s: secret status failed", b.Name())
	}
	out, err := runSecretCommand(b.Name(), "status", p, "--status")
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
	out, err := runSecretCommand(b.Name(), "lookup", "op", "read", ref)
	if err != nil {
		return "", err
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return "", fmt.Errorf("%s: secret lookup returned an empty value", b.Name())
	}
	return v, nil
}

func (b *onePasswordBackend) Status() (string, error) {
	out, err := runSecretCommand(b.Name(), "status", "op", "account", "list")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// runSecretCommand executes a secret-manager helper under the shared timeout.
// It never returns helper output in errors because helper output may be secret
// material.
func runSecretCommand(backend, operation, command string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), secretCmdTimeout)
	defer cancel()

	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("%s: secret %s timed out", backend, operation)
		}
		return nil, fmt.Errorf("%s: secret %s failed", backend, operation)
	}
	return out.Bytes(), nil
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
