import { describe, expect, it, vi } from "vitest";

import {
  CredentialError,
  resolveOpenAIAPIKey,
} from "../src/credentials.js";

describe("resolveOpenAIAPIKey", () => {
  it("reads and trims the password field from the named item", async () => {
    const run = vi.fn().mockResolvedValue({ stdout: "  test-only-value\n" });

    await expect(resolveOpenAIAPIKey(run)).resolves.toBe("test-only-value");
    expect(run).toHaveBeenCalledWith("bw", [
      "get",
      "password",
      "OPENAI_API_KEY",
    ]);
  });

  it("does not include command output in errors", async () => {
    const run = vi
      .fn()
      .mockRejectedValue(new Error("vault failure: test-only-value"));

    await expect(resolveOpenAIAPIKey(run)).rejects.toEqual(
      expect.objectContaining<Partial<CredentialError>>({
        message: expect.not.stringContaining("test-only-value"),
      }),
    );
  });

  it("rejects an empty password field", async () => {
    await expect(
      resolveOpenAIAPIKey(async () => ({ stdout: " \n" })),
    ).rejects.toThrow(
      "Bitwarden item OPENAI_API_KEY has an empty password field",
    );
  });
});
