package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// mcpFixture runs a scripted MCP exchange against runMCPServerWith with
// in-memory buffers and returns the non-empty response lines and the exit code.
// Each input element is one request line (a newline is appended if missing).
func mcpFixture(t *testing.T, root string, input ...string) ([]string, int) {
	t.Helper()
	var in bytes.Buffer
	for _, l := range input {
		in.WriteString(l)
		if !strings.HasSuffix(l, "\n") {
			in.WriteString("\n")
		}
	}
	var out bytes.Buffer
	code := runMCPServerWith(&in, &out, root)
	var lines []string
	for _, l := range strings.Split(out.String(), "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	return lines, code
}

// rpcResp is the test-side shape of one JSON-RPC response line.
type rpcResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func mustRPC(t *testing.T, line string) rpcResp {
	t.Helper()
	var resp rpcResp
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.Fatalf("bad response line %q: %v", line, err)
	}
	return resp
}

// toolCallResp is the test-side shape of a tools/call result.
type toolCallResp struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool `json:"isError"`
}

func toolCallResult(t *testing.T, line string) toolCallResp {
	t.Helper()
	resp := mustRPC(t, line)
	if resp.Error != nil {
		t.Fatalf("unexpected JSON-RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	var tc toolCallResp
	if err := json.Unmarshal(resp.Result, &tc); err != nil {
		t.Fatalf("bad tool result %q: %v", resp.Result, err)
	}
	return tc
}

func TestUsageMentionsMCPServer(t *testing.T) {
	if !strings.Contains(usage, "mcp-server") {
		t.Fatal("usage must document the mcp-server command")
	}
}

func TestMCPInitialize(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	lines, code := mcpFixture(t, t.TempDir(),
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{}}}`,
	)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if len(lines) != 1 {
		t.Fatalf("got %d response lines, want 1: %v", len(lines), lines)
	}
	resp := mustRPC(t, lines[0])
	if resp.JSONRPC != "2.0" {
		t.Fatalf("jsonrpc = %q, want 2.0", resp.JSONRPC)
	}
	if string(resp.ID) != "1" {
		t.Fatalf("id = %s, want 1 (echoed)", resp.ID)
	}
	var init struct {
		ProtocolVersion string `json:"protocolVersion"`
		Capabilities    struct {
			Tools struct {
				ListChanged bool `json:"listChanged"`
			} `json:"tools"`
		} `json:"capabilities"`
		ServerInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}
	if err := json.Unmarshal(resp.Result, &init); err != nil {
		t.Fatalf("bad initialize result %q: %v", resp.Result, err)
	}
	if init.ProtocolVersion != "2025-03-26" {
		t.Fatalf("protocolVersion = %q, want 2025-03-26", init.ProtocolVersion)
	}
	if init.ServerInfo.Name != "pi-run" {
		t.Fatalf("serverInfo.name = %q, want pi-run", init.ServerInfo.Name)
	}
	if init.ServerInfo.Version != Version {
		t.Fatalf("serverInfo.version = %q, want %q", init.ServerInfo.Version, Version)
	}
	if init.Capabilities.Tools.ListChanged {
		t.Fatal("tools.listChanged must be false")
	}
}

func TestMCPInitializeNewerProtocolVersion(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	lines, _ := mcpFixture(t, t.TempDir(),
		`{"jsonrpc":"2.0","id":"init-1","method":"initialize","params":{"protocolVersion":"2026-06-01","capabilities":{}}}`,
	)
	resp := mustRPC(t, lines[0])
	if string(resp.ID) != `"init-1"` {
		t.Fatalf("id = %s, want \"init-1\"", resp.ID)
	}
	var init struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(resp.Result, &init); err != nil {
		t.Fatal(err)
	}
	if init.ProtocolVersion != "2025-03-26" {
		t.Fatalf("protocolVersion = %q, want 2025-03-26 (implemented version)", init.ProtocolVersion)
	}
}

func TestMCPToolsList(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	lines, _ := mcpFixture(t, t.TempDir(), `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	resp := mustRPC(t, lines[0])
	if resp.Error != nil {
		t.Fatalf("tools/list errored: %s", lines[0])
	}
	var list struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &list); err != nil {
		t.Fatalf("bad tools/list result %q: %v", resp.Result, err)
	}
	if len(list.Tools) != 3 {
		t.Fatalf("got %d tools, want 3: %v", len(list.Tools), list.Tools)
	}
	want := []string{"providers", "cost", "benchmark_dry_run"}
	for i, w := range want {
		if list.Tools[i].Name != w {
			t.Fatalf("tool[%d] = %q, want %q", i, list.Tools[i].Name, w)
		}
	}
}

func TestMCPCallProviders(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	orig := Providers
	Providers = defaultProviders
	defer func() { Providers = orig }()
	lines, _ := mcpFixture(t, t.TempDir(),
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"providers","arguments":{}}}`,
	)
	tc := toolCallResult(t, lines[0])
	if tc.IsError {
		t.Fatalf("providers tool failed: %+v", tc.Content)
	}
	var infos []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(tc.Content[0].Text), &infos); err != nil {
		t.Fatalf("providers text is not a JSON array: %v", err)
	}
	found := false
	for _, info := range infos {
		var name, keyEnv string
		if err := json.Unmarshal(info["name"], &name); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(info["keyEnv"], &keyEnv); err != nil {
			t.Fatal(err)
		}
		if name == "openai" {
			found = true
		}
		// Only the four documented fields may appear; keyEnv must be an
		// env-var NAME, never a secret value.
		for k := range info {
			switch k {
			case "name", "defaultModel", "keyEnv", "baseURL":
			default:
				t.Fatalf("provider entry has unexpected field %q", k)
			}
		}
		if keyEnv == "" || keyEnv != strings.ToUpper(keyEnv) || strings.ContainsAny(keyEnv, " \t\"") {
			t.Fatalf("keyEnv %q is not an env-var name", keyEnv)
		}
	}
	if !found {
		t.Fatalf("providers array missing openai: %s", tc.Content[0].Text)
	}
	if strings.Contains(strings.ToLower(tc.Content[0].Text), "sk-") {
		t.Fatalf("providers payload contains a secret-shaped value: %s", tc.Content[0].Text)
	}
}

func TestMCPCallCostEmptyRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	lines, _ := mcpFixture(t, root,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"cost","arguments":{}}}`,
	)
	tc := toolCallResult(t, lines[0])
	if tc.IsError {
		t.Fatalf("cost tool must not error on an empty root: %+v", tc.Content)
	}
	var report costReport
	if err := json.Unmarshal([]byte(tc.Content[0].Text), &report); err != nil {
		t.Fatalf("cost text is not a report: %v", err)
	}
	if report.TotalCostUSD != 0 || report.Sessions != 0 || len(report.Rows) != 0 {
		t.Fatalf("expected zero-cost report, got %+v", report)
	}
}

func TestMCPCallCostWithSessions(t *testing.T) {
	root := costFixtureRoot(t, map[string]string{"a.jsonl": fixtureSessionA})
	lines, _ := mcpFixture(t, root,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"cost","arguments":{"since":"2026-08-01"}}}`,
	)
	tc := toolCallResult(t, lines[0])
	if tc.IsError {
		t.Fatalf("cost tool failed: %+v", tc.Content)
	}
	var report costReport
	if err := json.Unmarshal([]byte(tc.Content[0].Text), &report); err != nil {
		t.Fatal(err)
	}
	if report.Sessions != 1 || report.TotalCostUSD <= 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestMCPCallCostBadSince(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	lines, _ := mcpFixture(t, root,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"cost","arguments":{"since":"not-a-date"}}}`,
	)
	tc := toolCallResult(t, lines[0])
	if !tc.IsError {
		t.Fatal("cost with an invalid since date must be a tool error")
	}
	if len(tc.Content) == 0 || !strings.Contains(tc.Content[0].Text, "since") {
		t.Fatalf("error text missing since hint: %+v", tc.Content)
	}
}

func TestMCPCallUnknownTool(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	lines, _ := mcpFixture(t, t.TempDir(),
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"bogus_tool"}}`,
	)
	resp := mustRPC(t, lines[0])
	if resp.Error != nil {
		t.Fatalf("unknown tool must be a tool-level isError, not a JSON-RPC error: %s", lines[0])
	}
	var tc toolCallResp
	if err := json.Unmarshal(resp.Result, &tc); err != nil {
		t.Fatal(err)
	}
	if !tc.IsError {
		t.Fatal("unknown tool must set isError")
	}
	if len(tc.Content) == 0 || !strings.Contains(tc.Content[0].Text, "bogus_tool") {
		t.Fatalf("error text missing tool name: %+v", tc.Content)
	}
}

func TestMCPParseErrorThenValidRequest(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	lines, code := mcpFixture(t, t.TempDir(),
		`this is not json at all`,
		`{"jsonrpc":"2.0","id":8,"method":"ping"}`,
	)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if len(lines) != 2 {
		t.Fatalf("got %d response lines, want 2: %v", len(lines), lines)
	}
	first := mustRPC(t, lines[0])
	if first.Error == nil || first.Error.Code != -32700 {
		t.Fatalf("first response must be parse error -32700, got %s", lines[0])
	}
	if string(first.ID) != "null" {
		t.Fatalf("parse-error id = %s, want null", first.ID)
	}
	second := mustRPC(t, lines[1])
	if second.Error != nil {
		t.Fatalf("ping after parse error failed: %s", lines[1])
	}
	var res map[string]any
	if err := json.Unmarshal(second.Result, &res); err != nil {
		t.Fatal(err)
	}
	if len(res) != 0 {
		t.Fatalf("ping result must be an empty object, got %s", second.Result)
	}
}

func TestMCPPing(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	lines, code := mcpFixture(t, t.TempDir(), `{"jsonrpc":"2.0","id":9,"method":"ping"}`)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	resp := mustRPC(t, lines[0])
	if resp.Error != nil {
		t.Fatalf("ping errored: %s", lines[0])
	}
	var res map[string]any
	if err := json.Unmarshal(resp.Result, &res); err != nil {
		t.Fatal(err)
	}
	if len(res) != 0 {
		t.Fatalf("ping result = %s, want {}", resp.Result)
	}
	if string(resp.ID) != "9" {
		t.Fatalf("id = %s, want 9", resp.ID)
	}
}

func TestMCPNotificationGetsNoResponse(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	lines, code := mcpFixture(t, t.TempDir(),
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":10,"method":"ping"}`,
	)
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if len(lines) != 1 {
		t.Fatalf("got %d responses, want 1 (notification must be silent): %v", len(lines), lines)
	}
	if string(mustRPC(t, lines[0]).ID) != "10" {
		t.Fatalf("expected only the ping response, got %v", lines)
	}
}

func TestMCPUnknownMethod(t *testing.T) {
	t.Setenv("HARNESS_ROOT", t.TempDir())
	lines, _ := mcpFixture(t, t.TempDir(), `{"jsonrpc":"2.0","id":11,"method":"bogus/method"}`)
	resp := mustRPC(t, lines[0])
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Fatalf("expected -32601 method-not-found, got %s", lines[0])
	}
	if string(resp.ID) != "11" {
		t.Fatalf("id = %s, want 11", resp.ID)
	}
}

func TestMCPBenchmarkDryRun(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HARNESS_ROOT", root)
	// No benchmarks directory: zero-count summary, not an error.
	lines, _ := mcpFixture(t, root,
		`{"jsonrpc":"2.0","id":12,"method":"tools/call","params":{"name":"benchmark_dry_run","arguments":{}}}`,
	)
	tc := toolCallResult(t, lines[0])
	if tc.IsError {
		t.Fatalf("benchmark_dry_run without a benchmarks dir must not error: %+v", tc.Content)
	}
	var summary mcpBenchmarkDryRun
	if err := json.Unmarshal([]byte(tc.Content[0].Text), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Total != 0 || summary.Valid != 0 || summary.Invalid != 0 || len(summary.Tasks) != 0 {
		t.Fatalf("expected zero-count summary, got %+v", summary)
	}

	// With a valid task: total 1, valid 1, no errors.
	writeBenchmarkTask(t, root, "demo", `{"id": "demo", "prompt": "Fix it."}`)
	lines, _ = mcpFixture(t, root,
		`{"jsonrpc":"2.0","id":13,"method":"tools/call","params":{"name":"benchmark_dry_run"}}`,
	)
	tc = toolCallResult(t, lines[0])
	if tc.IsError {
		t.Fatalf("benchmark_dry_run failed: %+v", tc.Content)
	}
	if err := json.Unmarshal([]byte(tc.Content[0].Text), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Total != 1 || summary.Valid != 1 || summary.Invalid != 0 || len(summary.Tasks) != 1 || summary.Tasks[0] != "demo" {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}
