// SECURITY: `pi-run mcp-server` is a LOCAL-ONLY, READ-ONLY MCP server speaking
// line-delimited JSON-RPC 2.0 over stdio (MCP spec 2025-03-26). It must NEVER
// launch agents, resolve API keys, or accept remote connections. The providers
// tool exposes env-var NAMES (keyEnv) only — never key values; the cost tool
// aggregates spend from local session files; benchmark_dry_run validates task
// format without Docker or keys. Do not add write, exec, or network paths here.
package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// mcpProtocolVersion is the MCP spec version this server implements. Even if a
// client requests a newer version, we respond with the version we implement.
const mcpProtocolVersion = "2025-03-26"

// JSON-RPC 2.0 error codes.
const (
	rpcParseError     = -32700
	rpcInvalidRequest = -32600
	rpcMethodNotFound = -32601
)

// rpcRequest is one line-delimited JSON-RPC 2.0 request. ID is kept raw so it
// can be echoed verbatim; a missing ID makes this a notification.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// rpcResponse is the envelope written for every request that carries an id.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// rpcError is a JSON-RPC 2.0 error object.
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// toolContent is one MCP tool-result content item (text-only).
type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// toolResult is the MCP tools/call result: text content plus an optional
// isError flag for tool-level failures (never a JSON-RPC error).
type toolResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// mcpTool is one entry of the tools/list result.
type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// mcpServerInfo identifies the MCP server implementation.
type mcpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// mcpInitializeResult is the MCP initialize response payload.
type mcpInitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      mcpServerInfo  `json:"serverInfo"`
}

// mcpProviderInfo is the read-only provider description exposed by the
// providers tool. It carries the env-var NAME (keyEnv) only — never a key
// value.
type mcpProviderInfo struct {
	Name         string `json:"name"`
	DefaultModel string `json:"defaultModel"`
	KeyEnv       string `json:"keyEnv"`
	BaseURL      string `json:"baseURL,omitempty"`
}

// mcpBenchmarkDryRun is the machine-readable benchmark_dry_run summary,
// mirroring the CLI dry-run accounting (total = valid + invalid).
type mcpBenchmarkDryRun struct {
	Tasks   []string `json:"tasks"`
	Errors  []string `json:"errors,omitempty"`
	Total   int      `json:"total"`
	Valid   int      `json:"valid"`
	Invalid int      `json:"invalid"`
}

// runMCPServer serves the MCP protocol on stdin/stdout until EOF, then exits
// 0. Only a fatal internal error (I/O failure) yields exit 1.
func runMCPServer() int {
	return runMCPServerWith(os.Stdin, os.Stdout, repoRoot())
}

// runMCPServerWith serves the MCP protocol line by line: each input line is
// one JSON-RPC 2.0 message, each response is one output line. Malformed lines
// and unknown methods produce JSON-RPC errors and the server keeps serving
// subsequent lines.
func runMCPServerWith(in io.Reader, out io.Writer, root string) int {
	r := bufio.NewReader(in)
	w := bufio.NewWriter(out)
	for {
		line, err := r.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			if werr := serveLine(w, line, root); werr != nil {
				return 1
			}
			if werr := w.Flush(); werr != nil {
				return 1
			}
		}
		if err != nil {
			if err == io.EOF {
				return 0
			}
			return 1
		}
	}
}

// serveLine handles one request line. It returns an error only for fatal
// output failures (the server must stop); all client-facing problems are
// answered with a JSON-RPC or tool error and serving continues.
func serveLine(w *bufio.Writer, line []byte, root string) error {
	req, err := parseRequest(line)
	if err != nil {
		return writeRPC(w, rpcResponse{
			JSONRPC: "2.0",
			ID:      json.RawMessage("null"),
			Error:   &rpcError{Code: rpcParseError, Message: "parse error: " + err.Error()},
		})
	}
	// notifications/initialized never gets a response, even if a client
	// mislabels it with an id.
	if req.Method == "notifications/initialized" {
		return nil
	}
	// Requests without an id (notifications) get no response.
	if len(req.ID) == 0 || string(req.ID) == "null" {
		return nil
	}
	if req.JSONRPC != "2.0" {
		return writeRPC(w, rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: rpcInvalidRequest, Message: `invalid request: jsonrpc must be "2.0"`},
		})
	}
	if req.Method == "" {
		return writeRPC(w, rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: rpcInvalidRequest, Message: "invalid request: missing method"},
		})
	}

	switch req.Method {
	case "initialize":
		return writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: mcpInitializeResult{
			ProtocolVersion: mcpProtocolVersion,
			Capabilities:    map[string]any{"tools": map[string]any{"listChanged": false}},
			ServerInfo:      mcpServerInfo{Name: "pi-run", Version: Version},
		}})
	case "ping":
		return writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}})
	case "tools/list":
		return writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": mcpTools()}})
	case "tools/call":
		return writeRPC(w, rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: callMCPTool(req.Params, root)})
	default:
		return writeRPC(w, rpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: rpcMethodNotFound, Message: fmt.Sprintf("method not found: %s", req.Method)},
		})
	}
}

// parseRequest decodes one request line.
func parseRequest(line []byte) (rpcRequest, error) {
	var req rpcRequest
	if err := json.Unmarshal(line, &req); err != nil {
		return req, err
	}
	return req, nil
}

// writeRPC serializes one response as a single output line.
func writeRPC(w io.Writer, resp rpcResponse) error {
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}

// mcpTools returns the tool table advertised by tools/list.
func mcpTools() []mcpTool {
	return []mcpTool{
		{
			Name:        "providers",
			Description: "List configured providers (name, defaultModel, keyEnv, baseURL). keyEnv is the environment-variable NAME only — never a key value.",
			InputSchema: objectSchema(nil),
		},
		{
			Name:        "cost",
			Description: "Aggregate spend from local Pi session files under <root>/.pi/sessions. Optional \"since\" (YYYY-MM-DD) counts only sessions modified at/after that date. Zero sessions produce a zero report.",
			InputSchema: objectSchema(map[string]any{
				"since": map[string]any{"type": "string", "description": "Only count sessions modified at/after this date (YYYY-MM-DD)"},
			}),
		},
		{
			Name:        "benchmark_dry_run",
			Description: "Validate benchmark tasks under <root>/eval/benchmarks (no Docker, no keys). A missing benchmarks directory yields a zero-count summary, not an error.",
			InputSchema: objectSchema(nil),
		},
	}
}

// objectSchema builds a JSON Schema object inputSchema with the given
// properties (nil = no parameters).
func objectSchema(props map[string]any) map[string]any {
	schema := map[string]any{"type": "object", "properties": map[string]any{}}
	if props != nil {
		schema["properties"] = props
	}
	return schema
}

// callMCPTool executes one tools/call. Failures (unknown tool, bad arguments)
// are returned as a tool result with isError — never as a JSON-RPC error.
func callMCPTool(params json.RawMessage, root string) toolResult {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &call); err != nil {
			return toolError("invalid tools/call parameters: " + err.Error())
		}
	}
	switch call.Name {
	case "providers":
		return callProvidersTool(call.Arguments)
	case "cost":
		return callCostTool(call.Arguments, root)
	case "benchmark_dry_run":
		return callBenchmarkDryRunTool(call.Arguments, root)
	default:
		return toolError(fmt.Sprintf("unknown tool %q (available: providers, cost, benchmark_dry_run)", call.Name))
	}
}

// toolError builds an isError tool result with explanatory text.
func toolError(text string) toolResult {
	return toolResult{Content: []toolContent{{Type: "text", Text: text}}, IsError: true}
}

// toolText builds a successful tool result carrying stringified JSON.
func toolText(text string) toolResult {
	return toolResult{Content: []toolContent{{Type: "text", Text: text}}}
}

// rejectArguments verifies that a no-parameter tool received no arguments.
func rejectArguments(arguments json.RawMessage, tool string) error {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return nil
	}
	var args map[string]json.RawMessage
	if err := json.Unmarshal(arguments, &args); err != nil {
		return fmt.Errorf("invalid arguments: %v", err)
	}
	if len(args) > 0 {
		return fmt.Errorf("tool %q takes no parameters", tool)
	}
	return nil
}

// callProvidersTool lists the active provider table, env-var NAMES only.
func callProvidersTool(arguments json.RawMessage) toolResult {
	if err := rejectArguments(arguments, "providers"); err != nil {
		return toolError(err.Error())
	}
	infos := make([]mcpProviderInfo, 0, len(Providers))
	for _, p := range Providers {
		infos = append(infos, mcpProviderInfo{
			Name:         p.Name,
			DefaultModel: p.DefaultModel,
			KeyEnv:       p.KeyEnv,
			BaseURL:      p.BaseURL,
		})
	}
	b, err := json.Marshal(infos)
	if err != nil {
		return toolError("providers: " + err.Error())
	}
	return toolText(string(b))
}

// callCostTool aggregates spend from local session files, optionally filtered
// by an explicit "since" date (reuses collectCostReport/parseSinceDate).
func callCostTool(arguments json.RawMessage, root string) toolResult {
	var args struct {
		Since string `json:"since"`
	}
	if len(bytes.TrimSpace(arguments)) > 0 {
		if err := json.Unmarshal(arguments, &args); err != nil {
			return toolError("cost: invalid arguments: " + err.Error())
		}
	}
	since, err := parseSinceDate(args.Since)
	if err != nil {
		return toolError(fmt.Sprintf("cost: invalid \"since\" parameter: %v", err))
	}
	report, err := collectCostReport(root, since)
	if err != nil {
		return toolError("cost: " + err.Error())
	}
	// Marshal [] instead of null for an empty report so strict consumers
	// (and this tool's sibling benchmark_dry_run) get a consistent array.
	if report.Rows == nil {
		report.Rows = []costRow{}
	}
	b, err := json.Marshal(report)
	if err != nil {
		return toolError("cost: " + err.Error())
	}
	return toolText(string(b))
}

// callBenchmarkDryRunTool validates benchmark tasks under <root>/eval/benchmarks
// (reuses loadBenchmarkTasks). A missing benchmarks directory yields a
// zero-count summary, not an error.
func callBenchmarkDryRunTool(arguments json.RawMessage, root string) toolResult {
	if err := rejectArguments(arguments, "benchmark_dry_run"); err != nil {
		return toolError(err.Error())
	}
	if _, err := os.Stat(filepath.Join(root, "eval", "benchmarks")); err != nil {
		empty, err := json.Marshal(mcpBenchmarkDryRun{Tasks: []string{}, Errors: []string{}})
		if err != nil {
			return toolError("benchmark_dry_run: " + err.Error())
		}
		return toolText(string(empty))
	}
	tasks, errs := loadBenchmarkTasks(root, "")
	summary := mcpBenchmarkDryRun{
		Tasks:   make([]string, 0, len(tasks)),
		Errors:  make([]string, 0, len(errs)),
		Total:   len(tasks) + len(errs),
		Valid:   len(tasks),
		Invalid: len(errs),
	}
	for _, t := range tasks {
		summary.Tasks = append(summary.Tasks, t.ID)
	}
	for _, e := range errs {
		summary.Errors = append(summary.Errors, e.Error())
	}
	sort.Strings(summary.Tasks)
	b, err := json.Marshal(summary)
	if err != nil {
		return toolError("benchmark_dry_run: " + err.Error())
	}
	return toolText(string(b))
}
