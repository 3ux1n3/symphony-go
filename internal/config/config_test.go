package config

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestFromMapAppliesDefaultsAndResolvesEnv(t *testing.T) {
	t.Setenv("CLICKUP_API_TOKEN", "token-123")
	dir := t.TempDir()
	workflowPath := filepath.Join(dir, "nested", "WORKFLOW.md")

	cfg, err := FromMap(map[string]any{
		"tracker": map[string]any{
			"api_key":  "$CLICKUP_API_TOKEN",
			"list_ids": []any{"111", "222"},
		},
		"workspace": map[string]any{
			"root": "./workspaces",
		},
	}, workflowPath)
	if err != nil {
		t.Fatalf("FromMap returned error: %v", err)
	}

	if cfg.Tracker.Kind != "clickup" {
		t.Fatalf("Tracker.Kind = %q", cfg.Tracker.Kind)
	}
	if cfg.Tracker.APIKey != "token-123" {
		t.Fatalf("Tracker.APIKey = %q", cfg.Tracker.APIKey)
	}
	if len(cfg.Tracker.ListIDs) != 2 {
		t.Fatalf("ListIDs = %#v", cfg.Tracker.ListIDs)
	}
	if cfg.Tracker.ActiveStatuses[0] != "open" {
		t.Fatalf("ActiveStatuses = %#v", cfg.Tracker.ActiveStatuses)
	}
	if cfg.ClickUp.RunningStatus != "in progress" || cfg.ClickUp.ReviewStatus != "in review" {
		t.Fatalf("ClickUp statuses = %#v", cfg.ClickUp)
	}
	if cfg.Polling.Interval != 30*time.Second {
		t.Fatalf("Polling.Interval = %v", cfg.Polling.Interval)
	}
	if want := filepath.Join(dir, "nested", "workspaces"); cfg.Workspace.Root != want {
		t.Fatalf("Workspace.Root = %q, want %q", cfg.Workspace.Root, want)
	}
	if cfg.Agent.MaxRetries != 3 {
		t.Fatalf("MaxRetries = %d", cfg.Agent.MaxRetries)
	}
	if cfg.OpenCode.Command != "opencode serve" {
		t.Fatalf("OpenCode.Command = %q", cfg.OpenCode.Command)
	}
}

func TestFromMapOverridesValues(t *testing.T) {
	cfg, err := FromMap(map[string]any{
		"tracker": map[string]any{
			"api_key":           "literal-token",
			"list_ids":          []any{"333"},
			"active_statuses":   []any{"Open"},
			"terminal_statuses": []any{"Completed"},
		},
		"clickup": map[string]any{
			"running_status":     "In Progress",
			"review_status":      "In Review",
			"blocked_status":     "Blocked",
			"comment_on_success": false,
		},
		"polling": map[string]any{"interval_ms": 5000},
		"hooks":   map[string]any{"timeout_ms": 2000, "before_run": "make test"},
		"agent": map[string]any{
			"max_concurrent_agents": 4,
			"max_turns":             2,
			"max_retries":           5,
			"max_retry_backoff_ms":  10000,
		},
		"opencode": map[string]any{
			"command": "opencode serve --print-logs",
			"host":    "localhost",
			"model":   "anthropic/claude",
			"agent":   "build",
		},
	}, filepath.Join(t.TempDir(), "WORKFLOW.md"))
	if err != nil {
		t.Fatalf("FromMap returned error: %v", err)
	}

	if cfg.Tracker.ActiveStatuses[0] != "open" {
		t.Fatalf("ActiveStatuses = %#v", cfg.Tracker.ActiveStatuses)
	}
	if cfg.ClickUp.RunningStatus != "in progress" {
		t.Fatalf("RunningStatus = %q", cfg.ClickUp.RunningStatus)
	}
	if cfg.ClickUp.CommentOnSuccess {
		t.Fatalf("CommentOnSuccess = true, want false")
	}
	if cfg.Polling.Interval != 5*time.Second {
		t.Fatalf("Polling.Interval = %v", cfg.Polling.Interval)
	}
	if cfg.Hooks.Timeout != 2*time.Second || cfg.Hooks.BeforeRun != "make test" {
		t.Fatalf("Hooks = %#v", cfg.Hooks)
	}
	if cfg.Agent.MaxConcurrentAgents != 4 || cfg.Agent.MaxTurns != 2 || cfg.Agent.MaxRetries != 5 {
		t.Fatalf("Agent = %#v", cfg.Agent)
	}
	if cfg.Agent.MaxRetryBackoff != 10*time.Second {
		t.Fatalf("MaxRetryBackoff = %v", cfg.Agent.MaxRetryBackoff)
	}
	if cfg.OpenCode.Host != "localhost" || cfg.OpenCode.Model != "anthropic/claude" || cfg.OpenCode.Agent != "build" {
		t.Fatalf("OpenCode = %#v", cfg.OpenCode)
	}
}

func TestFromMapValidationErrors(t *testing.T) {
	_, err := FromMap(map[string]any{
		"tracker": map[string]any{
			"kind":     "linear",
			"api_key":  "$MISSING_CLICKUP_TOKEN",
			"list_ids": []any{},
		},
		"agent": map[string]any{"max_retries": -1},
	}, filepath.Join(t.TempDir(), "WORKFLOW.md"))
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
}
