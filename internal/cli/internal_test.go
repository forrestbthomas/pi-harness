package cli

import "testing"

func TestRepoRootEnvOverride(t *testing.T) {
	t.Setenv("HARNESS_ROOT", "/tmp/fake-harness")
	if got := repoRoot(); got != "/tmp/fake-harness" {
		t.Fatalf("got %q, want HARNESS_ROOT override", got)
	}
}
