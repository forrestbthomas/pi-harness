import readline from "node:readline/promises";
import { stdin as input, stdout as output } from "node:process";

import { ChatService, createOpenAIModelClient } from "./chat.js";
import type { ChatMessage } from "./types.js";

async function main(): Promise<void> {
  const service = new ChatService(await createOpenAIModelClient());
  const history: ChatMessage[] = [];
  const terminal = readline.createInterface({ input, output });

  process.stdout.write("Chatbot ready. Type /exit to quit.\n");
  try {
    while (true) {
      let userInput: string;
      try {
        userInput = await terminal.question("You: ");
      } catch {
        break;
      }

      const message = userInput.trim();
      if (message === "/exit") {
        break;
      }
      if (message.length === 0) {
        process.stdout.write("Chatbot: Please enter a message.\n");
        continue;
      }

      try {
        const answer = await service.reply(history, message);
        history.push(
          { role: "user", content: message },
          { role: "assistant", content: answer },
        );
        process.stdout.write(`Chatbot: ${answer}\n`);
      } catch {
        process.stderr.write(
          "Chatbot error: unable to respond. Check Bitwarden and model access.\n",
        );
      }
    }
  } finally {
    terminal.close();
  }
}

main().catch(() => {
  process.stderr.write(
    "Chatbot error: unable to start. Check Bitwarden and model access.\n",
  );
  process.exitCode = 1;
});
