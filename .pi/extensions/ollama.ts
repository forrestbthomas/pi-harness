/**
 * Project-local Ollama provider extension (issue #157).
 *
 * Registers a Pi provider id `ollama` whose model catalog is derived from the
 * locally configured Ollama daemon (`/api/tags`), so the `/model` picker shows
 * the exact installed tags instead of Pi's cached OpenAI catalog.
 *
 * Design notes:
 * - Static imports from `@earendil-works/pi-ai` are the documented extension
 *   pattern; Pi's loader aliases them to its bundled modules. Dynamic imports
 *   would bypass that alias and fail to resolve in the installed runtime.
 * - Pure discovery helpers live in `./lib/ollama-catalog.ts` so Node's
 *   hermetic tests can exercise them without a Pi runtime or a daemon.
 * - The daemon is keyless: auth declares availability without prompting for or
 *   persisting credentials. `createProvider` restores the last successful
 *   catalog from Pi's models store when discovery fails, so `/model` keeps a
 *   usable list when the daemon is briefly unavailable.
 */
import { createProvider, openAICompletionsApi } from "@earendil-works/pi-ai";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

import {
  fetchOllamaModels,
  resolveOllamaBaseUrl,
  type PickerModel,
} from "./lib/ollama-catalog.ts";

const daemonBaseUrl = resolveOllamaBaseUrl();
const OLLAMA_API_BASE_URL = `${daemonBaseUrl}/v1`;

export default async function registerOllamaProvider(pi: ExtensionAPI) {
  // Best-effort initial discovery: register whatever the daemon reports right
  // now (empty on failure). A later refresh restores Pi's persisted catalog
  // and retries the fetch, so an unavailable daemon never blocks startup.
  let initialModels: PickerModel[] = [];
  try {
    initialModels = await fetchOllamaModels();
  } catch (error) {
    console.warn(
      `[ollama] Could not discover ${daemonBaseUrl}/api/tags at startup; ` +
        `using the last successful catalog when available: ${error instanceof Error ? error.message : String(error)}`,
    );
  }

  pi.registerProvider(createProvider({
    id: "ollama",
    name: "Ollama",
    baseUrl: OLLAMA_API_BASE_URL,
    auth: {
      // Ollama's local daemon is keyless. Declaring keyless API-key auth marks
      // the provider available without prompting for or persisting credentials.
      apiKey: {
        name: "Ollama local daemon",
        async check() {
          return { type: "api_key", source: "local Ollama daemon" };
        },
        async resolve() {
          // Keyless local daemon: Pi's OpenAI-compatible stream layer refuses
          // to send a request without an authorization header, so supply a
          // dummy bearer token that the daemon ignores. No real credential is
          // read, prompted for, or persisted.
          return { auth: { headers: { authorization: "Bearer ollama" } }, source: "local Ollama daemon" };
        },
      },
    },
    models: initialModels,
    fetchModels: async (context: { signal: AbortSignal }) => {
      try {
        return await fetchOllamaModels({ signal: context.signal });
      } catch (error) {
        // createProvider restores the persisted catalog first and leaves it
        // untouched when this fetch fails, so /model keeps the last good list.
        console.warn(
          `[ollama] Could not refresh ${daemonBaseUrl}/api/tags; retaining the last discovered catalog: ${error instanceof Error ? error.message : String(error)}`,
        );
        throw error;
      }
    },
    api: openAICompletionsApi(),
  }));
}
