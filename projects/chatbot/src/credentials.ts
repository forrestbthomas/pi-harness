import { execFile } from "node:child_process";
import { promisify } from "node:util";

const execFileAsync = promisify(execFile);
const bitwardenItem = "OPENAI_API_KEY";

export class CredentialError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "CredentialError";
  }
}

export type CredentialCommandRunner = (
  command: string,
  args: readonly string[],
) => Promise<{ stdout: string }>;

async function defaultRunner(
  command: string,
  args: readonly string[],
): Promise<{ stdout: string }> {
  const result = await execFileAsync(command, args, { encoding: "utf8" });
  return { stdout: result.stdout };
}

export async function resolveOpenAIAPIKey(
  run: CredentialCommandRunner = defaultRunner,
): Promise<string> {
  try {
    const { stdout } = await run("bw", [
      "get",
      "password",
      bitwardenItem,
    ]);
    const apiKey = stdout.trim();

    if (apiKey.length === 0) {
      throw new CredentialError(
        "Bitwarden item OPENAI_API_KEY has an empty password field. Update the item, then try again.",
      );
    }

    return apiKey;
  } catch (error) {
    if (error instanceof CredentialError) {
      throw error;
    }

    throw new CredentialError(
      "Could not retrieve the OpenAI credential from Bitwarden. Confirm bw is installed and your vault is logged in and unlocked, then try again.",
    );
  }
}
