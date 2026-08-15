/**
 * Pure Ollama catalog discovery helpers.
 *
 * This module intentionally has NO Pi runtime imports: it is unit-tested by
 * Node's built-in test runner (`node --test`) without a daemon, a network, or
 * a Pi runtime. The extension (`../ollama.ts`) owns the Pi registration.
 */

/** Default loopback daemon (the standard Ollama port). */
export const OLLAMA_DEFAULT_BASE_URL = "http://localhost:11434";

/** Bound on one discovery request so a wedged daemon cannot stall startup. */
export const OLLAMA_DISCOVERY_TIMEOUT_MS = 1_500;

export type FetchLike = (
  url: string,
  init?: { signal?: AbortSignal },
) => Promise<{
  ok: boolean;
  status: number;
  json(): Promise<unknown>;
}>;

export type PickerModel = {
  id: string;
  name: string;
  provider: "ollama";
  api: "openai-completions";
  baseUrl: string;
  reasoning: boolean;
  input: ["text"];
  cost: { input: number; output: number; cacheRead: number; cacheWrite: number };
  contextWindow: number;
  maxTokens: number;
};

type OllamaTagsResponse = { models?: Array<{ name?: unknown }> };

/** Convert any abort reason into an Error so callers can match on message. */
function abortError(reason: unknown): Error {
  return reason instanceof Error ? reason : new Error(String(reason ?? "aborted"));
}

/**
 * Resolve the daemon base URL (no `/v1` suffix). Precedence:
 * 1. `OLLAMA_BASE_URL` (explicit override)
 * 2. `OPENAI_BASE_URL` (set by the harness launcher to the configured endpoint)
 * 3. the default loopback daemon
 *
 * The harness routes OpenAI-compatible traffic to `<base>/v1`, so a configured
 * `/v1` suffix is stripped before discovery hits the native `/api/tags`.
 */
export function resolveOllamaBaseUrl(env: Record<string, string | undefined> = process.env): string {
  const candidate = env.OLLAMA_BASE_URL ?? env.OPENAI_BASE_URL ?? OLLAMA_DEFAULT_BASE_URL;
  return candidate.replace(/\/v1\/?$/u, "").replace(/\/+$/u, "");
}

/** Convert `/api/tags` results into the full Model objects Pi requires. */
export function normalizeOllamaModels(payload: unknown, apiBaseUrl: string): PickerModel[] {
  const tags = (payload as OllamaTagsResponse)?.models ?? [];
  const seen = new Set<string>();
  const ids = tags
    .map((model) => (typeof model?.name === "string" ? model.name.trim() : ""))
    .filter((id) => id !== "" && !seen.has(id) && Boolean(seen.add(id)))
    .sort((a, b) => a.localeCompare(b));

  return ids.map((id) => ({
    id,
    name: id,
    provider: "ollama",
    api: "openai-completions",
    baseUrl: apiBaseUrl,
    reasoning: false,
    input: ["text"],
    cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0 },
    // /api/tags does not expose these limits consistently, so use conservative
    // defaults rather than inventing tag-specific metadata.
    contextWindow: 128_000,
    maxTokens: 4_096,
  }));
}

/** Fetch the exact locally installed Ollama tags with a bounded request lifetime. */
export async function fetchOllamaModels(options: {
  fetch?: FetchLike;
  signal?: AbortSignal;
  timeoutMs?: number;
  baseUrl?: string;
} = {}): Promise<PickerModel[]> {
  const request = options.fetch ?? (globalThis.fetch as FetchLike);
  const timeoutMs = options.timeoutMs ?? OLLAMA_DISCOVERY_TIMEOUT_MS;
  const baseUrl = options.baseUrl ?? resolveOllamaBaseUrl();
  // A caller that is already aborted must fail fast and never start a request:
  // the abort listener below only fires for signals that abort later.
  if (options.signal?.aborted) {
    throw abortError(options.signal.reason);
  }
  const controller = new AbortController();
  let timedOut = false;
  const abortFromCaller = () => controller.abort(options.signal?.reason);
  options.signal?.addEventListener("abort", abortFromCaller, { once: true });
  const timer = setTimeout(() => {
    timedOut = true;
    controller.abort(new Error(`Ollama catalog request timed out after ${timeoutMs}ms`));
  }, timeoutMs);

  try {
    const response = await request(`${baseUrl}/api/tags`, { signal: controller.signal });
    if (!response.ok) {
      throw new Error(`Ollama catalog request failed with HTTP ${response.status}`);
    }
    return normalizeOllamaModels(await response.json(), `${baseUrl}/v1`);
  } catch (error) {
    if (timedOut) {
      throw new Error(`Ollama catalog request timed out after ${timeoutMs}ms`, { cause: error });
    }
    throw error;
  } finally {
    clearTimeout(timer);
    options.signal?.removeEventListener("abort", abortFromCaller);
  }
}
