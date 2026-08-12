package cli

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// otelTestProvider is the fixed provider used across the OTel export tests.
var otelTestProvider = Provider{Name: "openai", DefaultModel: "openai/gpt-5.6-terra"}

// captureOTLPRequest runs fn with PI_OTLP_ENDPOINT pointing at a local
// httptest collector and returns the captured method, path, content type, and
// body of the single request the collector received.
func captureOTLPRequest(t *testing.T, fn func()) (method, path, contentType string, body []byte) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, contentType = r.Method, r.URL.Path, r.Header.Get("Content-Type")
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	t.Setenv("PI_OTLP_ENDPOINT", srv.URL)
	fn()
	return method, path, contentType, body
}

// captureStderr runs fn with os.Stderr redirected and returns what was written.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stderr = old
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// attrValue looks up an OTLP attribute by key, failing the test if absent.
func attrValue(t *testing.T, attrs []otlpKeyValue, key string) otlpAnyValue {
	t.Helper()
	for _, a := range attrs {
		if a.Key == key {
			return a.Value
		}
	}
	t.Fatalf("attribute %q not found in %v", key, attrs)
	return otlpAnyValue{}
}

// TestMaybeExportOTLPSpanPostsOTLP exercises the happy path: one POST to
// /v1/traces with the OTLP/HTTP JSON body and the expected span contents.
func TestMaybeExportOTLPSpanPostsOTLP(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	method, path, contentType, body := captureOTLPRequest(t, func() {
		maybeExportOTLPSpan(otelTestProvider, "openai/gpt-5.6-terra", "print", start, start.Add(30*time.Second), 7)
	})

	if method != http.MethodPost {
		t.Fatalf("method = %q, want POST", method)
	}
	if path != "/v1/traces" {
		t.Fatalf("path = %q, want /v1/traces", path)
	}
	if contentType != "application/json" {
		t.Fatalf("content-type = %q, want application/json", contentType)
	}
	for _, want := range []string{
		"invoke_agent",
		"openai/gpt-5.6-terra", // model
		"service.name",
		"pi-harness",
		"pi_harness.run.exit_code",
		`"intValue":7`, // the exit-code value
	} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}

	// Unmarshal the OTLP body to verify ids, timestamps, and the ERROR status.
	var req otlpTracesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("unmarshal OTLP body: %v", err)
	}
	if len(req.ResourceSpans) != 1 || len(req.ResourceSpans[0].ScopeSpans) != 1 || len(req.ResourceSpans[0].ScopeSpans[0].Spans) != 1 {
		t.Fatalf("unexpected OTLP shape: %+v", req)
	}
	span := req.ResourceSpans[0].ScopeSpans[0].Spans[0]
	if span.TraceID == span.SpanID {
		t.Fatal("traceId and spanId must be distinct")
	}
	for name, id := range map[string]string{"traceId": span.TraceID, "spanId": span.SpanID} {
		if len(id) != 16 {
			t.Fatalf("%s = %q, want 16 hex chars", name, id)
		}
		if _, err := hex.DecodeString(id); err != nil {
			t.Fatalf("%s = %q is not hex: %v", name, id, err)
		}
	}
	if want := strconv.FormatUint(uint64(start.UnixNano()), 10); span.StartTimeUnixNano != want {
		t.Fatalf("startTimeUnixNano = %q, want %q", span.StartTimeUnixNano, want)
	}
	if want := strconv.FormatUint(uint64(start.Add(30*time.Second).UnixNano()), 10); span.EndTimeUnixNano != want {
		t.Fatalf("endTimeUnixNano = %q, want %q", span.EndTimeUnixNano, want)
	}
	if span.Status.Code != 2 { // exit code 7 -> ERROR
		t.Fatalf("status.code = %d, want 2 (ERROR) for non-zero exit", span.Status.Code)
	}
}

// TestMaybeExportOTLPSpanOKStatusAndAttributes verifies the OK status and the
// pinned GenAI / pi-harness attribute values for a successful launch.
func TestMaybeExportOTLPSpanOKStatusAndAttributes(t *testing.T) {
	_, _, _, body := captureOTLPRequest(t, func() {
		start := time.Unix(1_700_000_000, 500_000_000)
		maybeExportOTLPSpan(otelTestProvider, "openai/gpt-5.6-terra", "chat", start, start.Add(2*time.Second), 0)
	})

	var req otlpTracesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("unmarshal OTLP body: %v", err)
	}
	rss := req.ResourceSpans[0]
	span := rss.ScopeSpans[0].Spans[0]
	if span.Name != "invoke_agent" {
		t.Fatalf("span name = %q, want invoke_agent", span.Name)
	}
	if span.Kind != 2 { // SPAN_KIND_CLIENT
		t.Fatalf("kind = %d, want 2 (CLIENT)", span.Kind)
	}
	if span.Status.Code != 1 { // exit code 0 -> OK
		t.Fatalf("status.code = %d, want 1 (OK) for exit code 0", span.Status.Code)
	}
	if got := attrValue(t, rss.Resource.Attributes, "service.name").StringValue; got != "pi-harness" {
		t.Fatalf("service.name = %q, want pi-harness", got)
	}
	if got := rss.ScopeSpans[0].Scope.Name; got != "pi-harness.agent" {
		t.Fatalf("scope.name = %q, want pi-harness.agent", got)
	}
	for key, want := range map[string]string{
		"gen_ai.agent.name":    "openai",
		"gen_ai.provider.name": "openai",
		"gen_ai.agent.model":   "openai/gpt-5.6-terra",
		"pi_harness.run.mode":  "chat",
	} {
		if got := attrValue(t, span.Attributes, key).StringValue; got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
	if got := attrValue(t, span.Attributes, "pi_harness.run.exit_code").IntValue; got != 0 {
		t.Fatalf("pi_harness.run.exit_code = %d, want 0", got)
	}
}

// TestMaybeExportOTLPSpanUnreachableEndpoint verifies the best-effort
// contract: a dead collector must not panic or propagate; exactly one warning
// line goes to stderr.
func TestMaybeExportOTLPSpanUnreachableEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	url := srv.URL
	srv.Close() // no listener: connection refused
	t.Setenv("PI_OTLP_ENDPOINT", url)

	stderr := captureStderr(t, func() {
		maybeExportOTLPSpan(otelTestProvider, "openai/gpt-5.6-terra", "print", time.Now(), time.Now(), 0)
	})
	if !strings.Contains(stderr, "pi-run: warning: telemetry export failed:") {
		t.Fatalf("expected a best-effort warning on stderr, got %q", stderr)
	}
}

// TestMaybeExportOTLPSpanDisabledWhenUnset verifies the no-op contract: with
// PI_OTLP_ENDPOINT unset the export must not open any connection.
func TestMaybeExportOTLPSpanDisabledWhenUnset(t *testing.T) {
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("PI_OTLP_ENDPOINT", "") // explicitly empty/unset
	maybeExportOTLPSpan(otelTestProvider, "openai/gpt-5.6-terra", "print", time.Now(), time.Now(), 0)
	if hits.Load() != 0 {
		t.Fatalf("expected zero HTTP requests when PI_OTLP_ENDPOINT is unset, got %d", hits.Load())
	}
}

// TestUsageMentionsOTLPEndpoint keeps the usage note from silently disappearing.
func TestUsageMentionsOTLPEndpoint(t *testing.T) {
	if !strings.Contains(usage, "PI_OTLP_ENDPOINT") || !strings.Contains(usage, "invoke_agent") {
		t.Fatal("usage must document PI_OTLP_ENDPOINT telemetry export")
	}
}
