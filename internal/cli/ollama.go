package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// ollamaDiscoveryTimeout bounds the launch-time default-model lookup so a
// wedged local daemon cannot stall a chat launch.
const ollamaDiscoveryTimeout = 1500 * time.Millisecond

// ollamaDefaultAPIBaseURL is the fallback OpenAI-compatible endpoint when a
// providers table does not set one.
const ollamaDefaultAPIBaseURL = "http://localhost:11434/v1"

// ollamaDaemonBaseURL strips the OpenAI-compatible /v1 suffix from the API
// base URL so discovery can hit the daemon's native /api/tags endpoint.
func ollamaDaemonBaseURL(apiBaseURL string) string {
	if apiBaseURL == "" {
		apiBaseURL = ollamaDefaultAPIBaseURL
	}
	base := strings.TrimRight(apiBaseURL, "/")
	return strings.TrimSuffix(base, "/v1")
}

// ollamaDefaultModel asks the local daemon for its installed tags and returns
// the first chat-capable id (falling back to the first tag when the payload
// omits capabilities). It returns "" when the daemon is unreachable or the
// catalog is empty, so callers keep their static default.
func ollamaDefaultModel(apiBaseURL string) string {
	client := &http.Client{}
	url := ollamaDaemonBaseURL(apiBaseURL) + "/api/tags"
	ctx, cancel := context.WithTimeout(context.Background(), ollamaDiscoveryTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ""
	}
	var payload struct {
		Models []struct {
			Name         string   `json:"name"`
			Capabilities []string `json:"capabilities"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	var first string
	for _, model := range payload.Models {
		name := strings.TrimSpace(model.Name)
		if name == "" {
			continue
		}
		if first == "" {
			first = name
		}
		for _, capability := range model.Capabilities {
			if capability == "completion" {
				return name
			}
		}
	}
	return first
}

// launchModelForOllama replaces the static default model with a live, locally
// installed id when launching the Ollama provider with no explicit --model or
// tier. Resume and non-Ollama providers are never rewritten; explicit
// selections always win.
func launchModelForOllama(p Provider, model, modelFlag, tier, mode string) string {
	if mode == "resume" || p.Name != "ollama" || modelFlag != "" || tier != "" {
		return model
	}
	if live := ollamaDefaultModel(p.BaseURL); live != "" {
		return live
	}
	return model
}

// benchmarkLaunchModel resolves the model for a benchmark run: an explicit
// --model wins; otherwise Ollama gets its live installed default and other
// providers keep their static default.
func benchmarkLaunchModel(p Provider, optsModel string) string {
	if optsModel != "" {
		return optsModel
	}
	return launchModelForOllama(p, p.DefaultModel, "", "", "")
}
