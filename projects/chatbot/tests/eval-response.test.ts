import { describe, expect, it } from "vitest";

import { runSingleEvaluation } from "../src/eval-response.js";
import type { ChatMessage } from "../src/types.js";

describe("runSingleEvaluation", () => {
  it("passes one input to a fresh conversation", async () => {
    const service = {
      reply: async (history: readonly ChatMessage[], input: string) =>
        `${history.length}:${input}`,
    };

    await expect(runSingleEvaluation("hello", service)).resolves.toBe(
      "0:hello",
    );
  });
});
