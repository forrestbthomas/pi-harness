import assert from "node:assert/strict";
import test from "node:test";

import {
  fetchOllamaModels,
  normalizeOllamaModels,
  OLLAMA_DEFAULT_BASE_URL,
  resolveOllamaBaseUrl,
} from "../lib/ollama-catalog.ts";

test("normalizes exact Ollama tags into picker models", () => {
  const models = normalizeOllamaModels(
    {
      models: [
        { name: "qwen3.6:35b-a3b" },
        { name: "qwen2.5-coder:0.5b" },
        { name: "qwen3.6:35b-a3b" },
        { name: "  " },
        {},
      ],
    },
    "http://localhost:11434/v1",
  );

  assert.deepEqual(
    models.map((model) => ({ id: model.id, name: model.name, provider: model.provider, baseUrl: model.baseUrl })),
    [
      { id: "qwen2.5-coder:0.5b", name: "qwen2.5-coder:0.5b", provider: "ollama", baseUrl: "http://localhost:11434/v1" },
      { id: "qwen3.6:35b-a3b", name: "qwen3.6:35b-a3b", provider: "ollama", baseUrl: "http://localhost:11434/v1" },
    ],
  );
});

test("resolves the daemon base URL from the harness endpoint and explicit override", () => {
  // The harness sets OPENAI_BASE_URL to the OpenAI-compatible /v1 endpoint;
  // discovery must hit the native /api/tags on the daemon root.
  assert.equal(
    resolveOllamaBaseUrl({ OPENAI_BASE_URL: "http://localhost:11434/v1" }),
    "http://localhost:11434",
  );
  assert.equal(
    resolveOllamaBaseUrl({ OPENAI_BASE_URL: "http://localhost:11434/v1/", OLLAMA_BASE_URL: "http://ollama.local:8080" }),
    "http://ollama.local:8080",
  );
  assert.equal(resolveOllamaBaseUrl({}), OLLAMA_DEFAULT_BASE_URL);
});

test("fetches the daemon tag endpoint with the bounded timeout signal", async () => {
  let requestedURL = "";
  let receivedSignal: AbortSignal | undefined;
  const models = await fetchOllamaModels({
    fetch: async (url, init) => {
      requestedURL = String(url);
      receivedSignal = init?.signal ?? undefined;
      return Response.json({ models: [{ name: "qwen3:0.6b" }] });
    },
    timeoutMs: 25,
  });

  assert.equal(requestedURL, `${OLLAMA_DEFAULT_BASE_URL}/api/tags`);
  assert.ok(receivedSignal, "discovery must use an abort signal");
  assert.equal(models[0]?.id, "qwen3:0.6b");
});

test("uses an explicit base URL when configured", async () => {
  let requestedURL = "";
  await fetchOllamaModels({
    baseUrl: "http://ollama.local:8080",
    fetch: async (url) => {
      requestedURL = String(url);
      return Response.json({ models: [] });
    },
    timeoutMs: 25,
  });
  assert.equal(requestedURL, "http://ollama.local:8080/api/tags");
});

test("times out instead of hanging startup when the daemon is wedged", async () => {
  await assert.rejects(
    fetchOllamaModels({
      fetch: async (_url, init) =>
        new Promise<Response>((_resolve, reject) => {
          init?.signal?.addEventListener("abort", () => reject(new Error("request aborted")), { once: true });
        }),
      timeoutMs: 5,
    }),
    /timed out/i,
  );
});

test("does not start a request when the caller signal is already aborted", async () => {
  const controller = new AbortController();
  controller.abort(new Error("caller aborted"));
  let called = false;
  await assert.rejects(
    fetchOllamaModels({
      signal: controller.signal,
      fetch: async () => {
        called = true;
        return Response.json({ models: [] });
      },
      timeoutMs: 25,
    }),
    /caller aborted/,
  );
  assert.equal(called, false, "no daemon request may be created for an already-aborted signal");
});

test("surfaces non-OK daemon responses", async () => {
  await assert.rejects(
    fetchOllamaModels({
      fetch: async () => ({ ok: false, status: 503, json: async () => ({}) }),
      timeoutMs: 25,
    }),
    /HTTP 503/i,
  );
});
