## Harness-Specific Guardrails

- Do not install packages globally. Use the provided Python virtual environment (`eval/.venv`) and Pi project packages (`.pi/npm/`).
- Do not mutate git history unless explicitly asked.
- Do not expose API keys in session output or committed files.
- When running Pi in evaluation mode, prefer `--no-session` or store sessions under `.pi/sessions`.
- If a requested action conflicts with project safety rules, pause and ask for confirmation.
- File paths: use absolute paths (`/Users/forrestthomas/...`); `~` resolves to the project directory, not your home, in file tools.
