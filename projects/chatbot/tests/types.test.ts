import { describe, expect, it } from "vitest";

import type { ChatRequest, ModelClient } from "../src/types.js";

describe("model contracts", () => {
  it("allows a model client to answer a chat request", async () => {
    const client: ModelClient = {
      async respond(request: ChatRequest) {
        return request.messages.at(-1)?.content ?? "";
      },
    };

    await expect(
      client.respond({ messages: [{ role: "user", content: "Hello" }] }),
    ).resolves.toBe("Hello");
  });
});
