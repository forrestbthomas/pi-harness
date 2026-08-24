package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// secretMarker is a high-precision pattern the deterministic scanner looks for.
// Precision over recall: the goal is to catch real credential material (the
// 2026-08-23 vault-dump incident) without flagging test fixtures or prose.
type secretMarker struct {
	Name    string
	Pattern *regexp.Regexp
	Why     string
}

var secretMarkers = []secretMarker{
	{
		Name: "vault-dump-command",
		// Match the session schema's execution field ("command":"bw list items")
		// — NOT prose that merely mentions bw (e.g. a security discussion or
		// "never run `bw list items`"), which caused heavy false positives in
		// the 2026-08-23 scan. A wrapped command (export ... && bw list items)
		// still matches via the [^"]* prefix.
		Pattern: regexp.MustCompile(`"command"\s*:\s*"[^"]*\bbw\s+(list|get|sync)\s+(items?|folders?)`),
		Why:     "a session EXECUTED `bw list items` — the whole Bitwarden vault lands in the transcript (2026-08-23 incident)",
	},
	{
		Name:    "openrouter-key",
		Pattern: regexp.MustCompile(`\bsk-or-[A-Za-z0-9]{20,}\b`),
		Why:     "OpenRouter API key",
	},
	{
		Name:    "anthropic-key",
		Pattern: regexp.MustCompile(`\bsk-ant-[A-Za-z0-9]{20,}\b`),
		Why:     "Anthropic API key",
	},
	{
		Name:    "github-pat",
		Pattern: regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{30,}\b`),
		Why:     "GitHub personal access token / OAuth token",
	},
	{
		Name:    "openai-style-key",
		Pattern: regexp.MustCompile(`\bsk-[A-Za-z0-9]{40,}\b`),
		Why:     "OpenAI/DeepSeek-style API key",
	},
	{
		Name:    "json-password-field",
		Pattern: regexp.MustCompile(`"password"\s*:\s*"[^"]{12,}"`),
		Why:     "a password field in captured JSON (vault dump shape)",
	},
}

// secretFinding is one detection: the file, line, marker name, and a redacted
// snippet (never the full value).
type secretFinding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Marker  string `json:"marker"`
	Snippet string `json:"snippet"` // redacted: 10 chars either side, match collapsed
}

// scanSecrets scans each path (a file, or a directory walked recursively) for
// secret markers. Returns findings sorted by file/line. Missing paths are an
// error (callers decide whether the default .pi/sessions absence is an error).
func scanSecrets(paths []string) ([]secretFinding, error) {
	var findings []secretFinding
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return nil, fmt.Errorf("scan %q: %w", p, err)
		}
		if !info.IsDir() {
			fs, err := scanFile(p)
			if err != nil {
				return nil, err
			}
			findings = append(findings, fs...)
			continue
		}
		err = filepath.Walk(p, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				return nil
			}
			fs, err := scanFile(path)
			if err != nil {
				return err
			}
			findings = append(findings, fs...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		return findings[i].Line < findings[j].Line
	})
	return findings, nil
}

// scanFile reads the whole file and checks each line for every secret marker.
// Whole-file read (not bufio.Scanner) so oversized transcript lines (tool
// outputs can exceed any fixed token limit) cannot abort the scan — a guard
// that errors out instead of flagging is a bypass.
func scanFile(path string) ([]secretFinding, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []secretFinding
	for i, text := range strings.Split(string(b), "\n") {
		for _, m := range secretMarkers {
			loc := m.Pattern.FindStringIndex(text)
			if loc == nil {
				continue
			}
			out = append(out, secretFinding{
				File:    path,
				Line:    i + 1,
				Marker:  m.Name,
				Snippet: redactSnippet(text, loc),
			})
		}
	}
	return out, nil
}

// redactSnippet returns up to 10 characters either side of the match with the
// match itself collapsed, so findings never echo full credential material.
func redactSnippet(text string, loc []int) string {
	const pad = 10
	start := loc[0] - pad
	if start < 0 {
		start = 0
	}
	end := loc[1] + pad
	if end > len(text) {
		end = len(text)
	}
	return text[start:loc[0]] + "…[redacted]…" + text[loc[1]:end]
}

const scanSecretsUsage = `Usage: pi-run scan-secrets [paths...] [--json]

Scan files (or directories, recursively) for high-precision secret markers:
vault-dump commands (bw list items), API key formats (sk-or-, sk-ant-, sk-,
ghp_/gho_/ghu_/ghs_), and password fields in captured JSON. The goal is real
credential material with minimal false positives.

Exits: 0 clean · 1 findings · 2 usage error. --json emits machine-readable
findings (file, line, marker, redacted snippet) so hooks/CI can gate on it.

Default path: .pi/sessions (live agent transcripts). This is the
2026-08-23 vault-dump incident class: a session ran 'bw list items' and the
entire Bitwarden vault landed in the transcript, which was then swept into a
commit. Run this before copying/capturing transcripts anywhere.
`

// runScanSecretsCmd implements `pi-run scan-secrets`.
func runScanSecretsCmd(args []string) int {
	jsonOut := false
	var paths []string
	for _, a := range args {
		switch a {
		case "--json":
			jsonOut = true
		case "--help", "-h":
			fmt.Print(scanSecretsUsage)
			return 0
		default:
			if strings.HasPrefix(a, "-") {
				fmt.Fprintf(os.Stderr, "pi-run: scan-secrets: unknown flag %q\n\n%s", a, scanSecretsUsage)
				return 2
			}
			paths = append(paths, a)
		}
	}
	if len(paths) == 0 {
		def := filepath.Join(repoRoot(), ".pi", "sessions")
		if _, err := os.Stat(def); err != nil {
			// No live transcripts: nothing to scan (not an error). In --json
			// mode emit valid JSON (an empty list) so parsers always work.
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				_ = enc.Encode([]secretFinding{})
			} else {
				fmt.Println("scan-secrets: clean (no .pi/sessions)")
			}
			return 0
		}
		paths = []string{def}
	}
	findings, err := scanSecrets(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pi-run: scan-secrets: %v\n", err)
		return 1
	}
	if jsonOut {
		// stdout is pure JSON — status text goes to stderr only.
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(findings)
	} else {
		for _, f := range findings {
			fmt.Printf("%s:%d: %s — %s\n", f.File, f.Line, f.Marker, f.Snippet)
		}
	}
	if len(findings) > 0 {
		fmt.Fprintf(os.Stderr, "scan-secrets: %d finding(s) — resolve/rotate and remove the material before committing or copying transcripts\n", len(findings))
		return 1
	}
	if !jsonOut {
		fmt.Printf("scan-secrets: clean (%d path(s))\n", len(paths))
	}
	return 0
}
