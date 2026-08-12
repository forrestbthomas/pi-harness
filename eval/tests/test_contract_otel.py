"""Contract tests: pi-run OTLP telemetry export against a fake collector.

PI_OTLP_ENDPOINT gates a single best-effort POST to <endpoint>/v1/traces per
launch (internal/cli/otel.go). Failures print exactly one stderr warning and
never change the exit code; stdout stays clean (stdout-is-sacred).
"""

import json
import subprocess


def _span(payload):
    return payload["resourceSpans"][0]["scopeSpans"][0]["spans"][0]


def _attrs(span):
    return {attribute["key"]: attribute["value"] for attribute in span["attributes"]}


def _run_print(pi_run_bin, env):
    return subprocess.run(
        [pi_run_bin, "print", "hello"], env=env, capture_output=True, text=True, timeout=30
    )


def _launch_env(hermetic_env, fake_launch_env, endpoint=None, fake_pi_exit=None):
    base = hermetic_env() if endpoint is None else hermetic_env(PI_OTLP_ENDPOINT=endpoint)
    return fake_launch_env.env(base, OPENAI_API_KEY="testvalue", fake_pi_exit=fake_pi_exit)


def _warnings(result) -> list[str]:
    return [
        line
        for line in result.stderr.splitlines()
        if "pi-run: warning: telemetry export failed" in line
    ]


def test_otel_no_endpoint_no_requests(pi_run_bin, hermetic_env, fake_launch_env, fake_collector):
    result = _run_print(pi_run_bin, _launch_env(hermetic_env, fake_launch_env))
    assert result.returncode == 0, result.stderr
    assert fake_collector.requests == []


def test_otel_single_trace_post(pi_run_bin, hermetic_env, fake_launch_env, fake_collector):
    result = _run_print(
        pi_run_bin, _launch_env(hermetic_env, fake_launch_env, endpoint=fake_collector.url)
    )
    assert result.returncode == 0, result.stderr
    assert len(fake_collector.requests) == 1
    request = fake_collector.requests[0]
    assert request["method"] == "POST"
    assert request["path"] == "/v1/traces"
    assert request["headers"].get("Content-Type") == "application/json"
    span = _span(json.loads(request["body"]))
    assert span["name"] == "invoke_agent"
    assert span["kind"] == 2  # SPAN_KIND_CLIENT
    assert span["status"]["code"] == 1  # OTEL_STATUS_CODE_OK
    attrs = _attrs(span)
    assert attrs["gen_ai.agent.name"]["stringValue"] == "openai"
    assert attrs["gen_ai.provider.name"]["stringValue"] == "openai"
    assert attrs["gen_ai.agent.model"]["stringValue"] == "openai/gpt-5.6-terra"
    assert attrs["pi_harness.run.mode"]["stringValue"] == "print"
    # Go marshals the int64 with omitempty, so a zero exit code arrives as an
    # empty value object; pin the semantic contract either way.
    assert attrs["pi_harness.run.exit_code"].get("intValue", 0) == 0


def test_otel_exit_code_attribute_reflects_pi_exit(
    pi_run_bin, hermetic_env, fake_launch_env, fake_collector
):
    result = _run_print(
        pi_run_bin,
        _launch_env(
            hermetic_env, fake_launch_env, endpoint=fake_collector.url, fake_pi_exit=7
        ),
    )
    assert result.returncode == 7  # telemetry never changes the exit code
    assert len(fake_collector.requests) == 1
    span = _span(json.loads(fake_collector.requests[0]["body"]))
    assert span["status"]["code"] == 2  # OTEL_STATUS_CODE_ERROR
    assert _attrs(span)["pi_harness.run.exit_code"]["intValue"] == 7


def test_otel_collector_500_warns_and_exit_unchanged(
    pi_run_bin, hermetic_env, fake_launch_env, fake_collector
):
    fake_collector.status = 500
    result = _run_print(
        pi_run_bin, _launch_env(hermetic_env, fake_launch_env, endpoint=fake_collector.url)
    )
    assert result.returncode == 0
    warnings = _warnings(result)
    assert len(warnings) == 1, f"want exactly one warning, stderr:\n{result.stderr}"
    assert "500" in warnings[0]
    assert result.stdout == ""  # stdout-is-sacred


def test_otel_unreachable_endpoint_warns(pi_run_bin, hermetic_env, fake_launch_env):
    result = _run_print(
        pi_run_bin, _launch_env(hermetic_env, fake_launch_env, endpoint="http://127.0.0.1:1")
    )
    assert result.returncode == 0
    warnings = _warnings(result)
    assert len(warnings) == 1, f"want exactly one warning, stderr:\n{result.stderr}"
    assert result.stdout == ""  # stdout-is-sacred
