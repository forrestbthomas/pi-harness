.PHONY: help install pi pi-print pi-eval pi-eval-quick pi-config-check clean

help:
	@echo "Available targets:"
	@echo "  install        - create eval/.venv and install Python deps"
	@echo "  pi             - launch Pi interactively from the harness directory"
	@echo "  pi-print       - run Pi in print mode (set PROMPT=... first)"
	@echo "  pi-eval        - run the full DeepEval test suite"
	@echo "  pi-eval-quick  - run smoke tests that do not need an API key"
	@echo "  pi-config-check - validate harness config (OpenRouter defaults, skills, dotfiles) without an API key"
	@echo "  clean          - remove eval/.venv and generated artifacts"

install:
	python3 -m venv eval/.venv
	./eval/.venv/bin/pip install --upgrade pip
	./eval/.venv/bin/pip install -r eval/requirements.txt

# API key resolution: OPENROUTER_API_KEY env var, else Bitwarden via bw_get
# (item name == env var name). Set BW_GET to override the bw_get binary path.
# Unlock the vault first: `bw unlock` (or export BW_SESSION).
pi:
	@source ~/.nvm/nvm.sh && nvm use v22.19.0 && { \
	  [ -n "$$OPENROUTER_API_KEY" ] || export OPENROUTER_API_KEY="$$($${BW_GET:-$$HOME/bin/bw_get} OPENROUTER_API_KEY 2>/dev/null)"; \
	  pi; \
	}

pi-print:
	@source ~/.nvm/nvm.sh && nvm use v22.19.0 && { \
	  [ -n "$$OPENROUTER_API_KEY" ] || export OPENROUTER_API_KEY="$$($${BW_GET:-$$HOME/bin/bw_get} OPENROUTER_API_KEY 2>/dev/null)"; \
	  pi -p --no-session "$(PROMPT)"; \
	}

pi-eval:
	cd eval && source .venv/bin/activate && pytest tests/ -v

pi-eval-quick:
	cd eval && source .venv/bin/activate && pytest tests/test_code_quality.py tests/test_agent_task_completion.py::test_dataset_expected_outputs_are_non_empty -v

pi-config-check:
	cd eval && .venv/bin/pytest tests/test_harness_config.py -v

clean:
	rm -rf eval/.venv eval/__pycache__ eval/tests/__pycache__ eval/.pytest_cache
