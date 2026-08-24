package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeScanFixture(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestScanSecretsFindsVaultDumpCommand(t *testing.T) {
	p := writeScanFixture(t, "transcript.jsonl", `{"role":"bashExecution","command":"bw list items","output":"[{\"name\":\"OPENROUTER_API_KEY\",\"password\":\"abc\"}]"}`+"\n")
	findings, err := scanSecrets([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Marker != "vault-dump-command" {
		t.Fatalf("expected vault-dump-command finding, got %+v", findings)
	}
	if findings[0].Line != 1 {
		t.Fatalf("expected line 1, got %d", findings[0].Line)
	}
}

func TestScanSecretsFindsOpenRouterKey(t *testing.T) {
	key := "sk-or-" + strings.Repeat("a", 30)
	p := writeScanFixture(t, "s.jsonl", `{"content":"the key is `+key+` now"}`+"\n")
	findings, err := scanSecrets([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Marker != "openrouter-key" {
		t.Fatalf("expected openrouter-key finding, got %+v", findings)
	}
	if strings.Contains(findings[0].Snippet, key) {
		t.Fatalf("snippet must be redacted, got %q", findings[0].Snippet)
	}
}

func TestScanSecretsFindsGitHubPAT(t *testing.T) {
	p := writeScanFixture(t, "s.jsonl", "token ghp_"+strings.Repeat("b", 36)+"\n")
	findings, err := scanSecrets([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Marker != "github-pat" {
		t.Fatalf("expected github-pat finding, got %+v", findings)
	}
}

func TestScanSecretsFindsPasswordField(t *testing.T) {
	p := writeScanFixture(t, "s.jsonl", `{"login":{"password":"aVeryLongRealLookingPasswordValue"}}`+"\n")
	findings, err := scanSecrets([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Marker != "json-password-field" {
		t.Fatalf("expected json-password-field finding, got %+v", findings)
	}
}

func TestScanSecretsCleanDoesNotFlagFixtures(t *testing.T) {
	// Test fixtures/prose with short sk- fragments must not trip the scanner
	// (precision over recall).
	p := writeScanFixture(t, "test.go", `t.Setenv("OPENAI_API_KEY", "sk-openai")
t.Setenv("DEEPSEEK_API_KEY", "sk-x")
const note = "the sk- prefix alone is not a secret"`)
	findings, err := scanSecrets([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings for fixtures, got %+v", findings)
	}
}

func TestScanSecretsRecursiveDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.jsonl"), []byte("{\"command\":\"bw list items\",\"output\":\"...\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("fine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	findings, err := scanSecrets([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %+v", findings)
	}
}

func TestRunScanSecretsCmdUsageAndClean(t *testing.T) {
	if code := runScanSecretsCmd([]string{"--bogus"}); code != 2 {
		t.Fatalf("unknown flag: want exit 2, got %d", code)
	}
	if code := runScanSecretsCmd([]string{"--help"}); code != 0 {
		t.Fatalf("--help: want exit 0, got %d", code)
	}
	p := writeScanFixture(t, "clean.txt", "nothing here\n")
	if code := runScanSecretsCmd([]string{p}); code != 0 {
		t.Fatalf("clean file: want exit 0, got %d", code)
	}
	dirty := writeScanFixture(t, "dirty.jsonl", "{\"command\":\"bw get item X\"}\n")
	if code := runScanSecretsCmd([]string{"--json", dirty}); code != 1 {
		t.Fatalf("dirty file: want exit 1, got %d", code)
	}
}

func TestScanSecretsIgnoresProseMentioningBw(t *testing.T) {
	// Precision: a security discussion that MENTIONS "bw list items" must not
	// trip the vault-dump marker (the 2026-08-23 false-positive class — every
	// incident discussion in transcripts flagged).
	p := writeScanFixture(t, "discussion.jsonl", `{"role":"assistant","content":"Never run 'bw list items' in an agent session; it dumps the vault."}`+"\n")
	findings, err := scanSecrets([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("prose mentioning bw must not flag; got %+v", findings)
	}
}
