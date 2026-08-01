export type ChatRole = "user" | "assistant";

export interface ChatMessage {
  readonly role: ChatRole;
  readonly content: string;
}

export interface ChatRequest {
  readonly messages: readonly ChatMessage[];
}

export interface ModelClient {
  respond(request: ChatRequest): Promise<string>;
}
