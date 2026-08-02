# Examples

## 1. Custom Evaluation Metric

Add a metric in `eval/tests/test_code_quality.py` (see the existing
`CodeQualityMetric`) and register it in a test. Run `pi-run eval` to score agent
outputs against it.

## 2. Add a Provider Without Recompiling

Edit `providers.json` (add a row), then `pi-run providers` to verify it lists,
and `pi-run print --provider <name> "hello"` to use it.

## 3. Run Against a Local OpenAI-Compatible Model

```json
{ "name": "local", "keyEnv": "LOCAL_API_KEY", "piProvider": "openai", "defaultModel": "local/model", "baseURL": "http://localhost:11434/v1" }
```
Start Ollama (`ollama serve`), then:
```bash
export LOCAL_API_KEY=ollama   # any non-empty value; local server ignores it
pi-run print --provider local "Explain recursion"
```

## 4. Compare Providers Side-by-Side

```bash
pi-run print --provider openai    "Explain the CAP theorem" > /tmp/openai.txt
pi-run print --provider deepseek  "Explain the CAP theorem" > /tmp/deepseek.txt
diff /tmp/openai.txt /tmp/deepseek.txt
```
