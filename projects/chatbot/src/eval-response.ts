import { ChatService, createOpenAIModelClient } from "./chat.js";

type EvaluationService = Pick<ChatService, "reply">;

export async function runSingleEvaluation(
  input: string,
  service: EvaluationService,
): Promise<string> {
  return service.reply([], input);
}

async function readStandardInput(): Promise<string> {
  const chunks: Buffer[] = [];
  for await (const chunk of process.stdin) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }
  return Buffer.concat(chunks).toString("utf8");
}

function parseInput(raw: string): string {
  const payload: unknown = JSON.parse(raw);
  if (
    typeof payload !== "object" ||
    payload === null ||
    !("input" in payload) ||
    typeof payload.input !== "string"
  ) {
    throw new Error("Expected JSON input with a string input property.");
  }
  return payload.input;
}

async function main(): Promise<void> {
  const input = parseInput(await readStandardInput());
  const service = new ChatService(await createOpenAIModelClient());
  const output = await runSingleEvaluation(input, service);
  process.stdout.write(`${JSON.stringify({ output })}\n`);
}

if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch(() => {
    process.stderr.write("Evaluation response failed. Check Bitwarden and model access.\n");
    process.exitCode = 1;
  });
}
