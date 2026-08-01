import { describe, expect, it, vi } from "vitest";

import { ChatService } from "../src/chat.js";
import type { ModelClient } from "../src/types.js";

describe("ChatService", () => {
  it("adds a baseline system instruction and preserves previous turns", async () => {
    const respond = vi.fn().mockResolvedValue("I can help with that.");
    const service = new ChatService({ respond } satisfies ModelClient);

    const answer = await service.reply(
      [{ role: "assistant", content: "Welcome" }],
      "How do I reset my password?",
    );

    expect(answer).toBe("I can help with that.");
    expect(respond).toHaveBeenCalledWith({
      messages: [
        expect.objectContaining({
          role: "assistant",
          content: expect.stringContaining("support assistant"),
        }),
        { role: "assistant", content: "Welcome" },
        { role: "user", content: "How do I reset my password?" },
      ],
    });
  });

  it("rejects blank user input without calling the model", async () => {
    const respond = vi.fn().mockResolvedValue("not used");
    const service = new ChatService({ respond } satisfies ModelClient);

    await expect(service.reply([], "   ")).rejects.toThrow(
      "Message cannot be empty.",
    );
    expect(respond).not.toHaveBeenCalled();
  });
});
