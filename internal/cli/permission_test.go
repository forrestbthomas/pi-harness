package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

// hermeticLaunchEnv pins the env vars that runLaunch reads so tests never
// depend on ambient credentials, budgets, or permission modes. Missing key +
// nonexistent BW_GET means launch paths stop at key resolution (exit 3).
func hermeticLaunchEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HARNESS_ROOT", t.TempDir())
	t.Setenv("PI_PROVIDER", "openai")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("BW_GET", filepath.Join(t.TempDir(), "nonexistent-bw-get"))
	t.Setenv("PI_PERMISSION_MODE", "")
	t.Setenv("PI_MAX_BUDGET_USD", "")
}

func TestSplitLaunchArgsPermissionMode(t *testing.T) {
	_, _, _, pm, rest := splitLaunchArgs([]string{"--permission-mode", "acceptEdits", "hello"})
	if pm != "acceptEdits" || len(rest) != 1 || rest[0] != "hello" {
		t.Fatalf("got permissionMode=%q rest=%v", pm, rest)
	}
}

func TestSplitLaunchArgsPermissionModeEquals(t *testing.T) {
	_, _, _, pm, _ := splitLaunchArgs([]string{"--permission-mode=plan", "hi"})
	if pm != "plan" {
		t.Fatalf("got permissionMode=%q, want plan", pm)
	}
}

func TestSplitLaunchArgsReadOnlyAlias(t *testing.T) {
	_, _, _, pm, rest := splitLaunchArgs([]string{"--read-only"})
	if pm != "plan" {
		t.Fatalf("--read-only must alias --permission-mode plan, got %q", pm)
	}
	if len(rest) != 0 {
		t.Fatalf("--read-only must not leak into pass-through args, got %v", rest)
	}
}

func TestSplitLaunchArgsReadOnlyLastWins(t *testing.T) {
	_, _, _, pm, _ := splitLaunchArgs([]string{"--read-only", "--permission-mode", "acceptEdits"})
	if pm != "acceptEdits" {
		t.Fatalf("later --permission-mode must win, got %q", pm)
	}
}

func TestSplitLaunchArgsPermissionModeNotInRest(t *testing.T) {
	_, _, _, pm, rest := splitLaunchArgs([]string{"--permission-mode", "plan", "--tools", "read", "x"})
	if pm != "plan" || len(rest) != 3 || rest[0] != "--tools" || rest[1] != "read" || rest[2] != "x" {
		t.Fatalf("got permissionMode=%q rest=%v", pm, rest)
	}
}

func TestResolvePermissionModeFlagWinsOverEnv(t *testing.T) {
	t.Setenv("PI_PERMISSION_MODE", "plan")
	got, err := resolvePermissionMode("acceptEdits")
	if err != nil {
		t.Fatal(err)
	}
	if got != "acceptEdits" {
		t.Fatalf("got %q, want acceptEdits (flag wins over env)", got)
	}
}

func TestResolvePermissionModeEnvDefault(t *testing.T) {
	t.Setenv("PI_PERMISSION_MODE", "plan")
	got, err := resolvePermissionMode("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "plan" {
		t.Fatalf("got %q, want plan from PI_PERMISSION_MODE", got)
	}
}

func TestResolvePermissionModeEmpty(t *testing.T) {
	t.Setenv("PI_PERMISSION_MODE", "")
	got, err := resolvePermissionMode("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("got %q, want empty (no explicit tier)", got)
	}
}

func TestResolvePermissionModeUnknown(t *testing.T) {
	t.Setenv("PI_PERMISSION_MODE", "")
	if _, err := resolvePermissionMode("bogus"); err == nil {
		t.Fatal("expected error for unknown permission mode")
	}
}

func TestResolvePermissionModeUnknownEnv(t *testing.T) {
	t.Setenv("PI_PERMISSION_MODE", "bogus")
	if _, err := resolvePermissionMode(""); err == nil {
		t.Fatal("expected error for unknown PI_PERMISSION_MODE value")
	}
}

func TestPiArgsIncludesPermissionMode(t *testing.T) {
	p, _ := LookupProvider("openai")
	// plan maps to a read-only tool allowlist, NOT a --permission-mode flag.
	got := piArgs(p, p.DefaultModel, "chat", []string{"--tools", "read", "refactor x"}, false, "plan")
	want := []string{"--provider", "openai", "--model", "openai/gpt-5.6-terra", "--offline", "--tools", "read,grep,find,ls", "--tools", "read", "refactor x"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestPiArgsPermissionModesMapToPiFlags(t *testing.T) {
	p, _ := LookupProvider("openai")
	joined := func(mode string) string { return strings.Join(piArgs(p, p.DefaultModel, "chat", nil, false, mode), " ") }

	// plan -> read-only tool allowlist
	if got := joined("plan"); !strings.Contains(got, "--tools read,grep,find,ls") {
		t.Fatalf("plan must map to --tools read,grep,find,ls, got %q", got)
	}
	// bypassPermissions -> --approve
	if got := joined("bypassPermissions"); !strings.Contains(got, "--approve") {
		t.Fatalf("bypassPermissions must map to --approve, got %q", got)
	}
	// acceptEdits -> no permission flag (Pi defaults permit edits)
	if got := joined("acceptEdits"); strings.Contains(got, "--approve") || strings.Contains(got, "--tools read,grep,find,ls") || strings.Contains(got, "--permission-mode") {
		t.Fatalf("acceptEdits must emit no permission flag, got %q", got)
	}
	// default -> no permission flag
	if got := joined("default"); strings.Contains(got, "--approve") || strings.Contains(got, "--tools read,grep,find,ls") || strings.Contains(got, "--permission-mode") {
		t.Fatalf("default must emit no permission flag, got %q", got)
	}
}

func TestPiArgsOmitsPermissionModeWhenEmpty(t *testing.T) {
	p, _ := LookupProvider("openai")
	got := strings.Join(piArgs(p, p.DefaultModel, "chat", nil, false, ""), " ")
	if strings.Contains(got, "--permission-mode") || strings.Contains(got, "--approve") || strings.Contains(got, "read,grep,find,ls") {
		t.Fatalf("empty permission mode must not emit any permission flag, got %q", got)
	}
}

func TestRunChatUnknownPermissionModeExit2(t *testing.T) {
	hermeticLaunchEnv(t)
	code, out := captureRunStderr(t, []string{"chat", "--permission-mode", "bogus"})
	if code != 2 {
		t.Fatalf("chat unknown permission mode exit = %d, want 2; stderr: %s", code, out)
	}
	if !strings.Contains(out, "permission mode") {
		t.Fatalf("stderr must mention permission mode, got %q", out)
	}
}

func TestRunChatReadOnlyAccepted(t *testing.T) {
	hermeticLaunchEnv(t)
	// --read-only must parse and validate; with no key the run proceeds to key
	// resolution and exits 3 — any other code means the alias was rejected.
	if code := Run([]string{"chat", "--read-only"}); code != 3 {
		t.Fatalf("chat --read-only exit = %d, want 3 (missing key; flag accepted)", code)
	}
}

func TestRunChatEnvPermissionModeDefaultAccepted(t *testing.T) {
	hermeticLaunchEnv(t)
	t.Setenv("PI_PERMISSION_MODE", "plan")
	if code := Run([]string{"chat"}); code != 3 {
		t.Fatalf("chat with PI_PERMISSION_MODE=plan exit = %d, want 3 (missing key; env accepted)", code)
	}
}

func TestRunChatUnknownEnvPermissionModeExit2(t *testing.T) {
	hermeticLaunchEnv(t)
	t.Setenv("PI_PERMISSION_MODE", "bogus")
	code, out := captureRunStderr(t, []string{"chat"})
	if code != 2 {
		t.Fatalf("chat with PI_PERMISSION_MODE=bogus exit = %d, want 2; stderr: %s", code, out)
	}
}

func TestUsageMentionsPermissionFlags(t *testing.T) {
	if !strings.Contains(usage, "--permission-mode") || !strings.Contains(usage, "--read-only") {
		t.Fatal("usage must document --permission-mode and --read-only")
	}
}
