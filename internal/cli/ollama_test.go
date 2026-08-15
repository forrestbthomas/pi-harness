package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaDaemonBaseURL(t *testing.T) {
	for in, want := range map[string]string{
		"http://localhost:11434/v1":   "http://localhost:11434",
		"http://localhost:11434/v1/":  "http://localhost:11434",
		"http://ollama.local:8080/v1": "http://ollama.local:8080",
		"":                            "http://localhost:11434",
	} {
		if got := ollamaDaemonBaseURL(in); got != want {
			t.Fatalf("ollamaDaemonBaseURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOllamaDefaultModelPrefersChatCapable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[
			{"name":"nomic-embed-text:latest","capabilities":["embedding"]},
			{"name":"qwen3:0.6b","capabilities":["completion","tools"]},
			{"name":"qwen2.5-coder:1.5b-base","capabilities":["completion"]}
		]}`))
	}))
	defer server.Close()

	got := ollamaDefaultModel(server.URL + "/v1")
	if got != "qwen3:0.6b" {
		t.Fatalf("ollamaDefaultModel = %q, want first chat-capable %q", got, "qwen3:0.6b")
	}
}

func TestOllamaDefaultModelFallsBackToFirstTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:0.6b"}]}`))
	}))
	defer server.Close()

	if got := ollamaDefaultModel(server.URL + "/v1"); got != "qwen3:0.6b" {
		t.Fatalf("ollamaDefaultModel = %q, want first tag %q", got, "qwen3:0.6b")
	}
}

func TestOllamaDefaultModelUnavailableReturnsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	server.Close() // closed server => connection refused

	if got := ollamaDefaultModel(server.URL + "/v1"); got != "" {
		t.Fatalf("ollamaDefaultModel with dead daemon = %q, want empty", got)
	}
}

func TestLaunchModelForOllamaUsesLiveDefaultOnlyWhenUnflagged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:0.6b","capabilities":["completion"]}]}`))
	}))
	defer server.Close()

	ollama := Provider{Name: "ollama", PiProvider: "ollama", DefaultModel: "ollama/llama3.1", BaseURL: server.URL + "/v1", Keyless: true}

	if got := launchModelForOllama(ollama, "ollama/llama3.1", "", "", "chat"); got != "qwen3:0.6b" {
		t.Fatalf("unflagged ollama launch model = %q, want live default %q", got, "qwen3:0.6b")
	}
	if got := launchModelForOllama(ollama, "ollama/qwen3.6:35b-a3b", "ollama/qwen3.6:35b-a3b", "", "chat"); got != "ollama/qwen3.6:35b-a3b" {
		t.Fatalf("explicit --model must win, got %q", got)
	}
	if got := launchModelForOllama(ollama, "ollama/llama3.1", "", "fast", "chat"); got != "ollama/llama3.1" {
		t.Fatalf("tiered launch must keep the resolved tier model, got %q", got)
	}
}

func TestLaunchModelForOllamaResumeSkipsLiveLookup(t *testing.T) {
	hit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:0.6b","capabilities":["completion"]}]}`))
	}))
	defer server.Close()

	ollama := Provider{Name: "ollama", PiProvider: "ollama", DefaultModel: "ollama/llama3.1", BaseURL: server.URL + "/v1", Keyless: true}
	if got := launchModelForOllama(ollama, "ollama/llama3.1", "", "", "resume"); got != "ollama/llama3.1" {
		t.Fatalf("resume must keep the resolved session model, got %q", got)
	}
	if hit {
		t.Fatal("resume must not query the Ollama daemon for a default model")
	}
}

func TestBenchmarkLaunchModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:0.6b","capabilities":["completion"]}]}`))
	}))
	defer server.Close()

	ollama := Provider{Name: "ollama", PiProvider: "ollama", DefaultModel: "ollama/llama3.1", BaseURL: server.URL + "/v1", Keyless: true}
	if got := benchmarkLaunchModel(ollama, ""); got != "qwen3:0.6b" {
		t.Fatalf("unflagged ollama benchmark model = %q, want live default %q", got, "qwen3:0.6b")
	}
	if got := benchmarkLaunchModel(ollama, "ollama/qwen3.6:35b-a3b"); got != "ollama/qwen3.6:35b-a3b" {
		t.Fatalf("explicit benchmark --model must win, got %q", got)
	}
	openai := Provider{Name: "openai", PiProvider: "openai", DefaultModel: "openai/gpt-5.6-terra"}
	if got := benchmarkLaunchModel(openai, ""); got != "openai/gpt-5.6-terra" {
		t.Fatalf("non-ollama benchmark model changed to %q", got)
	}
}

func TestLaunchModelForOllamaLeavesOtherProvidersAlone(t *testing.T) {
	openai := Provider{Name: "openai", PiProvider: "openai", DefaultModel: "openai/gpt-5.6-terra"}
	if got := launchModelForOllama(openai, "openai/gpt-5.6-terra", "", "", "chat"); got != "openai/gpt-5.6-terra" {
		t.Fatalf("non-ollama provider model changed to %q", got)
	}
}

func TestLaunchModelForOllamaFallsBackWhenDaemonDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	server.Close()

	ollama := Provider{Name: "ollama", PiProvider: "ollama", DefaultModel: "ollama/llama3.1", BaseURL: server.URL + "/v1", Keyless: true}
	got := launchModelForOllama(ollama, "ollama/llama3.1", "", "", "chat")
	if got != "ollama/llama3.1" {
		t.Fatalf("dead daemon should keep the static default, got %q", got)
	}
	if strings.Contains(got, "qwen") {
		t.Fatalf("unexpected live model %q", got)
	}
}
