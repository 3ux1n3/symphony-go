package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrValidation = errors.New("config_validation_error")

type Config struct {
	WorkflowPath string
	Tracker      TrackerConfig
	ClickUp      ClickUpConfig
	Polling      PollingConfig
	Workspace    WorkspaceConfig
	Hooks        HooksConfig
	Agent        AgentConfig
	OpenCode     OpenCodeConfig
}

type TrackerConfig struct {
	Kind             string
	APIKey           string
	ListIDs          []string
	ActiveStatuses   []string
	TerminalStatuses []string
}

type ClickUpConfig struct {
	RunningStatus    string
	ReviewStatus     string
	BlockedStatus    string
	CommentOnStart   bool
	CommentOnSuccess bool
	CommentOnFailure bool
	CommentOnRetry   bool
}

type PollingConfig struct {
	Interval time.Duration
}

type WorkspaceConfig struct {
	Root string
}

type HooksConfig struct {
	AfterCreate  string
	BeforeRun    string
	AfterRun     string
	BeforeRemove string
	Timeout      time.Duration
}

type AgentConfig struct {
	MaxConcurrentAgents int
	MaxTurns            int
	MaxRetries          int
	MaxRetryBackoff     time.Duration
}

type OpenCodeConfig struct {
	Command string
	Host    string
	Model   string
	Agent   string
}

func FromMap(raw map[string]any, workflowPath string) (Config, error) {
	workflowDir := "."
	if workflowPath != "" {
		workflowDir = filepath.Dir(workflowPath)
	}

	cfg := Config{
		WorkflowPath: workflowPath,
		Tracker: TrackerConfig{
			Kind:             "clickup",
			ActiveStatuses:   []string{"open"},
			TerminalStatuses: []string{"completed"},
		},
		ClickUp: ClickUpConfig{
			RunningStatus:    "in progress",
			ReviewStatus:     "in review",
			BlockedStatus:    "blocked",
			CommentOnStart:   true,
			CommentOnSuccess: true,
			CommentOnFailure: true,
			CommentOnRetry:   true,
		},
		Polling: PollingConfig{Interval: 30 * time.Second},
		Workspace: WorkspaceConfig{
			Root: filepath.Join(os.TempDir(), "symphony_workspaces"),
		},
		Hooks: HooksConfig{Timeout: time.Minute},
		Agent: AgentConfig{
			MaxConcurrentAgents: 2,
			MaxTurns:            1,
			MaxRetries:          3,
			MaxRetryBackoff:     5 * time.Minute,
		},
		OpenCode: OpenCodeConfig{
			Command: "opencode serve",
			Host:    "127.0.0.1",
		},
	}

	tracker := mapValue(raw, "tracker")
	cfg.Tracker.Kind = stringValue(tracker, "kind", cfg.Tracker.Kind)
	cfg.Tracker.APIKey = resolveEnv(stringValue(tracker, "api_key", cfg.Tracker.APIKey))
	cfg.Tracker.ListIDs = stringSliceValue(tracker, "list_ids", cfg.Tracker.ListIDs)
	cfg.Tracker.ActiveStatuses = normalizeStatuses(stringSliceValue(tracker, "active_statuses", cfg.Tracker.ActiveStatuses))
	cfg.Tracker.TerminalStatuses = normalizeStatuses(stringSliceValue(tracker, "terminal_statuses", cfg.Tracker.TerminalStatuses))

	clickup := mapValue(raw, "clickup")
	cfg.ClickUp.RunningStatus = normalizeStatus(stringValue(clickup, "running_status", cfg.ClickUp.RunningStatus))
	cfg.ClickUp.ReviewStatus = normalizeStatus(stringValue(clickup, "review_status", cfg.ClickUp.ReviewStatus))
	cfg.ClickUp.BlockedStatus = normalizeStatus(stringValue(clickup, "blocked_status", cfg.ClickUp.BlockedStatus))
	cfg.ClickUp.CommentOnStart = boolValue(clickup, "comment_on_start", cfg.ClickUp.CommentOnStart)
	cfg.ClickUp.CommentOnSuccess = boolValue(clickup, "comment_on_success", cfg.ClickUp.CommentOnSuccess)
	cfg.ClickUp.CommentOnFailure = boolValue(clickup, "comment_on_failure", cfg.ClickUp.CommentOnFailure)
	cfg.ClickUp.CommentOnRetry = boolValue(clickup, "comment_on_retry", cfg.ClickUp.CommentOnRetry)

	polling := mapValue(raw, "polling")
	cfg.Polling.Interval = millisDuration(intValue(polling, "interval_ms", int(cfg.Polling.Interval/time.Millisecond)))

	workspace := mapValue(raw, "workspace")
	cfg.Workspace.Root = resolvePath(stringValue(workspace, "root", cfg.Workspace.Root), workflowDir)

	hooks := mapValue(raw, "hooks")
	cfg.Hooks.AfterCreate = stringValue(hooks, "after_create", cfg.Hooks.AfterCreate)
	cfg.Hooks.BeforeRun = stringValue(hooks, "before_run", cfg.Hooks.BeforeRun)
	cfg.Hooks.AfterRun = stringValue(hooks, "after_run", cfg.Hooks.AfterRun)
	cfg.Hooks.BeforeRemove = stringValue(hooks, "before_remove", cfg.Hooks.BeforeRemove)
	cfg.Hooks.Timeout = millisDuration(intValue(hooks, "timeout_ms", int(cfg.Hooks.Timeout/time.Millisecond)))

	agent := mapValue(raw, "agent")
	cfg.Agent.MaxConcurrentAgents = intValue(agent, "max_concurrent_agents", cfg.Agent.MaxConcurrentAgents)
	cfg.Agent.MaxTurns = intValue(agent, "max_turns", cfg.Agent.MaxTurns)
	cfg.Agent.MaxRetries = intValue(agent, "max_retries", cfg.Agent.MaxRetries)
	cfg.Agent.MaxRetryBackoff = millisDuration(intValue(agent, "max_retry_backoff_ms", int(cfg.Agent.MaxRetryBackoff/time.Millisecond)))

	opencode := mapValue(raw, "opencode")
	cfg.OpenCode.Command = stringValue(opencode, "command", cfg.OpenCode.Command)
	cfg.OpenCode.Host = stringValue(opencode, "host", cfg.OpenCode.Host)
	cfg.OpenCode.Model = stringValue(opencode, "model", cfg.OpenCode.Model)
	cfg.OpenCode.Agent = stringValue(opencode, "agent", cfg.OpenCode.Agent)

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	var problems []string
	if strings.ToLower(strings.TrimSpace(c.Tracker.Kind)) != "clickup" {
		problems = append(problems, "tracker.kind must be clickup")
	}
	if strings.TrimSpace(c.Tracker.APIKey) == "" {
		problems = append(problems, "tracker.api_key is required")
	}
	if len(c.Tracker.ListIDs) == 0 {
		problems = append(problems, "tracker.list_ids must include at least one list")
	}
	if len(c.Tracker.ActiveStatuses) == 0 {
		problems = append(problems, "tracker.active_statuses must include at least one status")
	}
	if strings.TrimSpace(c.ClickUp.RunningStatus) == "" {
		problems = append(problems, "clickup.running_status is required")
	}
	if strings.TrimSpace(c.ClickUp.ReviewStatus) == "" {
		problems = append(problems, "clickup.review_status is required")
	}
	if strings.TrimSpace(c.ClickUp.BlockedStatus) == "" {
		problems = append(problems, "clickup.blocked_status is required")
	}
	if c.Polling.Interval <= 0 {
		problems = append(problems, "polling.interval_ms must be positive")
	}
	if strings.TrimSpace(c.Workspace.Root) == "" {
		problems = append(problems, "workspace.root is required")
	}
	if c.Hooks.Timeout <= 0 {
		problems = append(problems, "hooks.timeout_ms must be positive")
	}
	if c.Agent.MaxConcurrentAgents <= 0 {
		problems = append(problems, "agent.max_concurrent_agents must be positive")
	}
	if c.Agent.MaxTurns <= 0 {
		problems = append(problems, "agent.max_turns must be positive")
	}
	if c.Agent.MaxRetries < 0 {
		problems = append(problems, "agent.max_retries must be zero or positive")
	}
	if c.Agent.MaxRetryBackoff <= 0 {
		problems = append(problems, "agent.max_retry_backoff_ms must be positive")
	}
	if strings.TrimSpace(c.OpenCode.Command) == "" {
		problems = append(problems, "opencode.command is required")
	}
	if strings.TrimSpace(c.OpenCode.Host) == "" {
		problems = append(problems, "opencode.host is required")
	}

	if len(problems) > 0 {
		return fmt.Errorf("%w: %s", ErrValidation, strings.Join(problems, "; "))
	}
	return nil
}

func mapValue(raw map[string]any, key string) map[string]any {
	v, ok := raw[key].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return v
}

func stringValue(raw map[string]any, key, fallback string) string {
	if v, ok := raw[key].(string); ok {
		return v
	}
	return fallback
}

func boolValue(raw map[string]any, key string, fallback bool) bool {
	if v, ok := raw[key].(bool); ok {
		return v
	}
	return fallback
}

func intValue(raw map[string]any, key string, fallback int) int {
	switch v := raw[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case uint64:
		return int(v)
	}
	return fallback
}

func stringSliceValue(raw map[string]any, key string, fallback []string) []string {
	v, ok := raw[key]
	if !ok {
		return fallback
	}

	switch values := v.(type) {
	case []string:
		return compactStrings(values)
	case []any:
		out := make([]string, 0, len(values))
		for _, item := range values {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	}
	return fallback
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}

func normalizeStatuses(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		status := normalizeStatus(value)
		if status != "" {
			out = append(out, status)
		}
	}
	return out
}

func normalizeStatus(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func resolveEnv(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "$") && len(trimmed) > 1 && !strings.ContainsAny(trimmed[1:], "/\\{} ") {
		return os.Getenv(trimmed[1:])
	}
	return value
}

func resolvePath(value, workflowDir string) string {
	resolved := resolveEnv(value)
	if strings.HasPrefix(resolved, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if resolved == "~" {
				resolved = home
			} else if strings.HasPrefix(resolved, "~/") {
				resolved = filepath.Join(home, resolved[2:])
			}
		}
	}
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(workflowDir, resolved)
	}
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return resolved
	}
	return abs
}

func millisDuration(ms int) time.Duration {
	return time.Duration(ms) * time.Millisecond
}
