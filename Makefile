# Agentic SRE — developer Makefile.
#
# Per-checkin verification runs LOCALLY (see RELEASING.md). The GitHub CI
# workflow is manual-only; `make verify` is the single command to run
# before opening or merging a PR.

.PHONY: verify
verify: ## Run the full local verification suite (mirrors ci.yml).
	bash scripts/verify-local.sh
