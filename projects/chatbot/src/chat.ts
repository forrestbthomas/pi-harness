import OpenAI from "openai";

import { resolveOpenAIAPIKey } from "./credentials.js";
import type { ChatMessage, ChatRequest, ModelClient } from "./types.js";

export const DEFAULT_MODEL = "gpt-5.6-terra";

const SYSTEM_MESSAGE: ChatMessage = {
  role: "system",
  content:
    "You are a concise support assistant. Answer general questions helpfully. Do not claim access to account data, policies, or actions you cannot verify.",
};

export class ChatService {
  constructor(private readonly client: ModelClient) {}

  async reply(
    history: readonly ChatMessage[],
    userInput: string,
  ): Promise<string> {
    const message = userInput.trim();
    if (message.length === 0) {
      throw new Error("Message cannot be empty.");
    }

    return this.client.respond({
      messages: [SYSTEM_MESSAGE, ...history, { role: "user", content: message }],
    });
  }
}

class OpenAIModelClient implements ModelClient {
  constructor(
    private readonly client: OpenAI,
    private readonly model: string,
  ) {}

  async respond(request: ChatRequest): Promise<string> {
    const response = await this.client.responses.create({
      model: this.model,
      input: request.messages.map((message) => ({
        role: message.role,
        content: message.content,
      })),
    });
    const text = response.output_text.trim();

    if (text.length === 0) {
      throw new Error("The model returned an empty response.");
    }

    return text;
  }
}

export async function createOpenAIModelClient(): Promise<ModelClient> {
  const apiKey = await resolveOpenAIAPIKey();
  const model = process.env.OPENAI_MODEL?.trim() || DEFAULT_MODEL;
  return new OpenAIModelClient(new OpenAI({ apiKey }), model);
}
