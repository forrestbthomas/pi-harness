package cli

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// maybeExportOTLPSpan exports one GenAI agent "invoke_agent" span per launch
// as OTLP/HTTP JSON to <PI_OTLP_ENDPOINT>/v1/traces, following the OTel GenAI
// semantic conventions (Development status:
// https://opentelemetry.io/docs/specs/semconv/gen-ai/).
//
// The feature is env-gated: when PI_OTLP_ENDPOINT is unset this is a no-op
// with zero overhead (no payload, no HTTP). Best-effort contract: any failure
// prints exactly one warning line to stderr and is never propagated, so
// telemetry can never change the pi-run exit code.
func maybeExportOTLPSpan(p Provider, model, mode string, start, end time.Time, exitCode int) {
	endpoint := os.Getenv("PI_OTLP_ENDPOINT")
	if endpoint == "" {
		return // feature off: zero overhead
	}

	payload, err := buildOTLPTracesRequest(p.Name, model, mode, start, end, exitCode)
	if err != nil {
		otelExportWarning(err)
		return
	}

	url := strings.TrimRight(endpoint, "/") + "/v1/traces"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		otelExportWarning(err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		otelExportWarning(err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		otelExportWarning(fmt.Errorf("collector returned status %s", resp.Status))
	}
}

// otelExportWarning is the single best-effort failure channel: one warning
// line on stderr, never an error return.
func otelExportWarning(err error) {
	fmt.Fprintf(os.Stderr, "pi-run: warning: telemetry export failed: %v\n", err)
}

// buildOTLPTracesRequest renders the OTLP/HTTP JSON "ExportTraceServiceRequest"
// body. Encoding follows the OTLP/JSON mapping: uint64 timestamps are strings,
// enums (kind, status.code) are numbers.
//
// Attribute names pin the OTel GenAI semantic conventions (Development status:
// https://opentelemetry.io/docs/specs/semconv/gen-ai/):
//
//   - gen_ai.agent.name      name of the agent (GenAI agent attributes)
//   - gen_ai.provider.name   name of the LLM provider (gen_ai.provider.name)
//   - gen_ai.agent.model     agent model identifier (gen_ai.agent.* model)
//   - pi_harness.run.mode    pi-harness-specific run telemetry (custom
//   - pi_harness.run.exit_code  attributes namespaced under pi_harness)
//
// The span name "invoke_agent" with kind CLIENT follows the GenAI agent span
// conventions; status.code uses the OTLP status enum (1 = OK, 2 = ERROR).
func buildOTLPTracesRequest(providerName, model, mode string, start, end time.Time, exitCode int) ([]byte, error) {
	traceID, err := randomID(16) // OTLP trace ids are 16 bytes (32 hex chars)
	if err != nil {
		return nil, fmt.Errorf("generate trace id: %w", err)
	}
	spanID, err := randomID(8) // OTLP span ids are 8 bytes (16 hex chars)
	if err != nil {
		return nil, fmt.Errorf("generate span id: %w", err)
	}

	statusCode := 2 // OTEL_STATUS_CODE_ERROR
	if exitCode == 0 {
		statusCode = 1 // OTEL_STATUS_CODE_OK
	}

	req := otlpTracesRequest{
		ResourceSpans: []otlpResourceSpans{{
			Resource: otlpResource{
				Attributes: []otlpKeyValue{
					{Key: "service.name", Value: otlpAnyValue{StringValue: "pi-harness"}},
				},
			},
			ScopeSpans: []otlpScopeSpans{{
				Scope: otlpScope{Name: "pi-harness.agent"},
				Spans: []otlpSpan{{
					TraceID:           traceID,
					SpanID:            spanID,
					Name:              "invoke_agent",
					Kind:              2, // SPAN_KIND_CLIENT
					StartTimeUnixNano: strconv.FormatUint(uint64(start.UnixNano()), 10),
					EndTimeUnixNano:   strconv.FormatUint(uint64(end.UnixNano()), 10),
					Attributes: []otlpKeyValue{
						{Key: "gen_ai.agent.name", Value: otlpAnyValue{StringValue: providerName}},
						{Key: "gen_ai.provider.name", Value: otlpAnyValue{StringValue: providerName}},
						{Key: "gen_ai.agent.model", Value: otlpAnyValue{StringValue: model}},
						{Key: "pi_harness.run.mode", Value: otlpAnyValue{StringValue: mode}},
						{Key: "pi_harness.run.exit_code", Value: otlpAnyValue{IntValue: int64(exitCode)}},
					},
					Status: otlpSpanStatus{Code: statusCode},
				}},
			}},
		}},
	}
	return json.Marshal(req)
}

// randomID returns n random bytes as lowercase hex. OTLP trace ids are 16
// bytes (32 hex chars) and span ids are 8 bytes (16 hex chars); keep them
// distinct so real collectors do not truncate the trace id.
func randomID(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// otlpTracesRequest mirrors the OTLP/HTTP JSON traces request shape
// (opentelemetry-proto trace_service and trace model, JSON mapping per the
// OTLP JSON encoding spec). Hand-rolled structs keep this stdlib-only.
type otlpTracesRequest struct {
	ResourceSpans []otlpResourceSpans `json:"resourceSpans"`
}

type otlpResourceSpans struct {
	Resource   otlpResource     `json:"resource"`
	ScopeSpans []otlpScopeSpans `json:"scopeSpans"`
}

type otlpResource struct {
	Attributes []otlpKeyValue `json:"attributes"`
}

type otlpScopeSpans struct {
	Scope otlpScope  `json:"scope"`
	Spans []otlpSpan `json:"spans"`
}

type otlpScope struct {
	Name string `json:"name"`
}

type otlpKeyValue struct {
	Key   string       `json:"key"`
	Value otlpAnyValue `json:"value"`
}

type otlpAnyValue struct {
	StringValue string `json:"stringValue,omitempty"`
	IntValue    int64  `json:"intValue,omitempty"`
}

type otlpSpan struct {
	TraceID           string         `json:"traceId"`
	SpanID            string         `json:"spanId"`
	Name              string         `json:"name"`
	Kind              int            `json:"kind"`
	StartTimeUnixNano string         `json:"startTimeUnixNano"`
	EndTimeUnixNano   string         `json:"endTimeUnixNano"`
	Attributes        []otlpKeyValue `json:"attributes"`
	Status            otlpSpanStatus `json:"status"`
}

type otlpSpanStatus struct {
	Code int `json:"code"`
}
