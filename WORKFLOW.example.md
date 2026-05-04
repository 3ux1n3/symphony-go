---
tracker:
  kind: clickup
  api_key: $CLICKUP_API_TOKEN
  list_ids:
    - "YOUR_CLICKUP_LIST_ID"
  active_statuses:
    - open
  terminal_statuses:
    - completed

clickup:
  running_status: in progress
  review_status: in review
  blocked_status: blocked
  comment_on_start: true
  comment_on_success: true
  comment_on_failure: true
  comment_on_retry: true

polling:
  interval_ms: 30000

workspace:
  root: ./symphony_workspaces

hooks:
  after_create: |
    git clone git@github.com:your-org/your-repo.git .
  before_run: |
    git status
  after_run: ""
  before_remove: ""
  timeout_ms: 60000

agent:
  max_concurrent_agents: 2
  max_turns: 1
  max_retries: 3
  max_retry_backoff_ms: 300000

opencode:
  command: opencode serve
  host: 127.0.0.1
  model: ""
  agent: ""
---
You are working on this ClickUp task.

Task: {{ task.identifier }} - {{ task.title }}
Status: {{ task.status }}
URL: {{ task.url }}

Description:
{{ task.description }}

Implement the task in this workspace. When finished, leave the work ready for review.
