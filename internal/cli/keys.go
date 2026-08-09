package cli

import (
	"fmt"
	"os"
)

// resolveSecret returns the value of env var name, falling back to the
// configured secret backend (PI_SECRET_BACKEND; default bitwarden via bw_get).
// Never logs values.
func resolveSecret(name string) (string, error) {
	if v := os.Getenv(name); v != "" {
		return v, nil
	}
	be, err := newSecretBackend()
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", name, err)
	}
	return be.Resolve(name)
}
