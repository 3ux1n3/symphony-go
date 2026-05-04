# Symphony Go Adapted Specification

Status: Draft v1

Purpose: Define a Go service that continuously reads eligible ClickUp tasks, prepares isolated workspaces, runs OpenCode agents, and writes operational progress back to ClickUp.

## 1. Intentional Divergences From Original SPEC.md

- Use ClickUp instead of Linear.
- Use OpenCode server instead of Codex app-server.
- Implement in Go.
- Use high-trust defaults and auto-approve OpenCode permissions.
- Use hooks to populate and prepare workspaces; v1 does not provide built-in Git clone behavior.
- Include first-class ClickUp writebacks in the orchestrator.
- Skip optional dashboard, SSH workers, durable retry persistence, and custom agent tools in v1.

## 2. ClickUp Status Policy

The first implementation targets a ClickUp workspace with these statuses:

- `planning`
- `open`
- `in progress`
- `blocked`
- `in review`
- `completed`

Dispatch behavior:

- Only tasks in `open` are eligible for dispatch.
- When Symphony claims a task, it moves the task from `open` to `in progress`.
- When OpenCode completes successfully, Symphony moves the task to `in review`.
- When retries are exhausted, Symphony moves the task to `blocked`.
- If a running task is manually moved to `planning`, `blocked`, or `in review`, Symphony stops the worker without deleting the workspace.
- If a running task is manually moved to `completed`, Symphony stops the worker and may clean the workspace when configured.

## 3. Main Components

1. Workflow Loader
   - Reads `WORKFLOW.md`.
   - Parses optional YAML front matter.
   - Returns raw config and prompt body.

2. Config Layer
   - Applies defaults.
   - Resolves explicit `$VAR` environment references.
   - Validates values required for dispatch.

3. ClickUp Client
   - Fetches candidate tasks from configured List IDs.
   - Refreshes task status for reconciliation.
   - Writes task comments and status transitions.
   - Normalizes ClickUp task payloads into internal task records.

4. Workspace Manager
   - Creates deterministic per-task workspace directories.
   - Enforces sanitized path names and root containment.
   - Runs lifecycle hooks.

5. OpenCode Runner
   - Starts `opencode serve` in the task workspace.
   - Creates an OpenCode session.
   - Sends the rendered task prompt.
   - Reads events and auto-approves permission requests under the high-trust policy.
   - Aborts and terminates the server on cancellation.

6. Orchestrator
   - Owns polling, claims, running state, retry queue, and reconciliation.
   - Decides when to dispatch, retry, stop, or release tasks.

7. Logging
   - Emits structured logs to stderr.
   - Must not log secrets.

## 4. Workflow File

The default workflow path is `./WORKFLOW.md`. A CLI argument may provide an explicit workflow path.

`WORKFLOW.md` contains optional YAML front matter followed by a Markdown prompt template. Unknown top-level config keys are ignored.

Required dispatch config:

- `tracker.kind: clickup`
- `tracker.api_key`, usually `$CLICKUP_API_TOKEN`
- `tracker.list_ids`, at least one ClickUp List ID
- `opencode.command`, default `opencode serve`

Recommended defaults:

- `tracker.active_statuses: ["open"]`
- `tracker.terminal_statuses: ["completed"]`
- `clickup.running_status: "in progress"`
- `clickup.review_status: "in review"`
- `clickup.blocked_status: "blocked"`
- `polling.interval_ms: 30000`
- `workspace.root: <system-temp>/symphony_workspaces`
- `hooks.timeout_ms: 60000`
- `agent.max_concurrent_agents: 2`
- `agent.max_turns: 1`
- `agent.max_retries: 3`
- `agent.max_retry_backoff_ms: 300000`

## 5. Task Model

Normalized task fields:

- `id`: ClickUp task ID.
- `identifier`: ClickUp custom ID when present, otherwise task ID.
- `title`: task name.
- `description`: markdown description when available, otherwise plain text.
- `status`: current ClickUp status name.
- `url`: ClickUp task URL when available.
- `priority`: ClickUp priority when available.
- `tags`: task tags.
- `assignees`: task assignees.
- `created_at`: creation timestamp.
- `updated_at`: update timestamp.
- `list_id`: ClickUp home list ID.

## 6. Workspace Rules

- Workspace path is `<workspace.root>/<sanitized_task_identifier>`.
- Sanitization replaces any character outside `[A-Za-z0-9._-]` with `_`.
- The final workspace path must remain inside the configured workspace root.
- OpenCode must only run with `cwd` equal to the task workspace path.
- Successful runs preserve workspaces.

Supported hooks:

- `hooks.after_create`: runs only when a workspace is created.
- `hooks.before_run`: runs before every attempt.
- `hooks.after_run`: runs after every attempt; failures are logged and ignored.
- `hooks.before_remove`: runs before cleanup; failures are logged and ignored.

Hooks run in the task workspace directory. Hooks are trusted configuration and are responsible for repository checkout, sync, dependency installation, or other preparation.

## 7. OpenCode High-Trust Policy

The v1 implementation runs in a trusted local environment.

- OpenCode permission requests are auto-approved unless explicitly denied by OpenCode configuration outside Symphony.
- User-input-required situations are treated as worker failures unless OpenCode can proceed without input.
- Symphony must still support cancellation and must not let workers stall forever.
- `opencode serve` binds to loopback by default.

## 8. Orchestration Flow

Startup:

1. Load and validate workflow config.
2. Initialize ClickUp and OpenCode runtime settings.
3. Start polling immediately.

Tick:

1. Reconcile running tasks.
2. Fetch `open` tasks from configured ClickUp lists.
3. Sort candidates by priority, created time, and identifier.
4. Dispatch while concurrency slots remain.

Dispatch:

1. Add task to in-memory claim set.
2. Move ClickUp status to `in progress`.
3. Create/reuse workspace.
4. Run hooks.
5. Start OpenCode worker.

Worker success:

1. Run `after_run` hook.
2. Comment on ClickUp when configured.
3. Move task to `in review`.
4. Release claim and preserve workspace.

Worker failure:

1. Run `after_run` hook best effort.
2. Comment on ClickUp when configured.
3. Retry with exponential backoff until `agent.max_retries` is reached.
4. After retry exhaustion, move task to `blocked` and release claim.

## 9. ClickUp Writebacks

The orchestrator owns these writebacks in v1:

- Status update on claim.
- Comment on start when configured.
- Comment on retry when configured.
- Comment on success when configured.
- Comment on final failure when configured.
- Status update to `in review` on success.
- Status update to `blocked` after retry exhaustion.

Writeback failures are logged. A failed claim status update prevents dispatch because it could cause duplicate workers.

## 10. Test Profile

Unit tests should cover:

- Workflow parsing.
- Config defaults and env resolution.
- ClickUp task normalization.
- ClickUp writeback request construction.
- Workspace sanitization and path containment.
- Hook timeout/failure semantics.
- Dispatch eligibility.
- Retry backoff and max retry exhaustion.
- Reconciliation decisions.

Integration tests should be skipped by default unless explicitly enabled with credentials and local OpenCode availability.
