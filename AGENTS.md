# Symphony Go Agent Guide

## Project Purpose

Symphony Go is a long-running Go service that orchestrates OpenCode agents against ClickUp tasks.

The implementation intentionally diverges from the original `SPEC.md`:

- Use ClickUp instead of Linear for issue tracking.
- Use OpenCode server instead of Codex app-server for coding-agent execution.
- Use high-trust defaults and auto-approve OpenCode permissions.
- Use workspace hooks for repository checkout/preparation; do not add built-in Git clone behavior unless explicitly requested.
- Keep optional extensions out of v1 unless explicitly requested.

## Repository Layout

- `cmd/symphony/`: CLI entrypoint.
- `internal/workflow/`: `WORKFLOW.md` loading, YAML front matter, prompt rendering.
- `internal/config/`: typed config resolution, defaults, validation.
- `internal/clickup/`: ClickUp REST client and task normalization/writebacks.
- `internal/workspace/`: per-task workspaces, hooks, path safety.
- `internal/opencode/`: OpenCode server process and HTTP/SSE client.
- `internal/orchestrator/`: polling, claiming, retries, reconciliation.
- `internal/logging/`: structured logs.
- `docs/`: adapted implementation specification and design notes.

## Build Rules

- Prefer small, focused packages with clear interfaces.
- Keep the orchestrator as the only owner of scheduling state.
- Do not log ClickUp tokens, OpenCode credentials, or secret environment values.
- Preserve workspaces after successful runs.
- Treat workspace path containment as a hard safety invariant.
- Run commands from the repository root unless a package-specific command requires otherwise.

## Verification

- Use `go test ./...` for code changes.
- Use integration tests only when explicitly enabled by environment variables.
- Skip real ClickUp/OpenCode integration tests by default when credentials or binaries are unavailable.
