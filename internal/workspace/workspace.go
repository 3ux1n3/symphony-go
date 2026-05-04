package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/3ux1n3/symphony-go/internal/config"
)

var (
	ErrInvalidIdentifier = errors.New("invalid_workspace_identifier")
	ErrPathOutsideRoot   = errors.New("workspace_path_outside_root")
	ErrNotDirectory      = errors.New("workspace_path_not_directory")
	ErrHookFailed        = errors.New("workspace_hook_failed")
	ErrHookTimeout       = errors.New("workspace_hook_timeout")
)

const maxHookOutputBytes = 16 * 1024

var unsafeWorkspaceChar = regexp.MustCompile(`[^A-Za-z0-9._-]`)

type Manager struct {
	root  string
	hooks config.HooksConfig
}

type Workspace struct {
	Path       string
	Key        string
	CreatedNow bool
}

type HookResult struct {
	Name     string
	Output   string
	Duration time.Duration
}

func NewManager(root string, hooks config.HooksConfig) (*Manager, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("workspace root is required")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root: %w", err)
	}

	if hooks.Timeout <= 0 {
		hooks.Timeout = time.Minute
	}

	return &Manager{root: filepath.Clean(absRoot), hooks: hooks}, nil
}

func (m *Manager) Root() string {
	return m.root
}

func SanitizeKey(identifier string) (string, error) {
	key := unsafeWorkspaceChar.ReplaceAllString(strings.TrimSpace(identifier), "_")
	if key == "" {
		return "", ErrInvalidIdentifier
	}
	return key, nil
}

func (m *Manager) PathForIdentifier(identifier string) (string, string, error) {
	key, err := SanitizeKey(identifier)
	if err != nil {
		return "", "", err
	}

	path := filepath.Join(m.root, key)
	path, err = filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("resolve workspace path: %w", err)
	}
	path = filepath.Clean(path)

	if err := m.EnsureInsideRoot(path); err != nil {
		return "", "", err
	}

	return path, key, nil
}

func (m *Manager) Prepare(ctx context.Context, identifier string) (Workspace, error) {
	path, key, err := m.PathForIdentifier(identifier)
	if err != nil {
		return Workspace{}, err
	}

	created, err := ensureDir(path)
	if err != nil {
		return Workspace{}, err
	}

	ws := Workspace{Path: path, Key: key, CreatedNow: created}
	if created && strings.TrimSpace(m.hooks.AfterCreate) != "" {
		if _, err := m.runHook(ctx, "after_create", m.hooks.AfterCreate, path); err != nil {
			return Workspace{}, err
		}
	}

	return ws, nil
}

func (m *Manager) RunBeforeRun(ctx context.Context, ws Workspace) (HookResult, error) {
	return m.runHook(ctx, "before_run", m.hooks.BeforeRun, ws.Path)
}

func (m *Manager) RunAfterRun(ctx context.Context, ws Workspace) (HookResult, error) {
	return m.runHook(ctx, "after_run", m.hooks.AfterRun, ws.Path)
}

func (m *Manager) Remove(ctx context.Context, ws Workspace) error {
	if err := m.EnsureInsideRoot(ws.Path); err != nil {
		return err
	}
	if strings.TrimSpace(m.hooks.BeforeRemove) != "" {
		_, _ = m.runHook(ctx, "before_remove", m.hooks.BeforeRemove, ws.Path)
	}
	return os.RemoveAll(ws.Path)
}

func (m *Manager) EnsureInsideRoot(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve workspace path: %w", err)
	}
	absPath = filepath.Clean(absPath)

	rel, err := filepath.Rel(m.root, absPath)
	if err != nil {
		return fmt.Errorf("compare workspace path: %w", err)
	}
	parentPrefix := ".." + string(filepath.Separator)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, parentPrefix) || filepath.IsAbs(rel) {
		return fmt.Errorf("%w: %s is not under %s", ErrPathOutsideRoot, absPath, m.root)
	}
	return nil
}

func (m *Manager) runHook(ctx context.Context, name, script, cwd string) (HookResult, error) {
	start := time.Now()
	if strings.TrimSpace(script) == "" {
		return HookResult{Name: name}, nil
	}
	if err := m.EnsureInsideRoot(cwd); err != nil {
		return HookResult{Name: name}, err
	}

	hookCtx, cancel := context.WithTimeout(ctx, m.hooks.Timeout)
	defer cancel()

	cmd := exec.CommandContext(hookCtx, "sh", "-lc", script)
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	result := HookResult{Name: name, Output: truncateOutput(string(output)), Duration: time.Since(start)}

	if hookCtx.Err() == context.DeadlineExceeded {
		return result, fmt.Errorf("%w: %s exceeded %s", ErrHookTimeout, name, m.hooks.Timeout)
	}
	if err != nil {
		return result, fmt.Errorf("%w: %s: %v: %s", ErrHookFailed, name, err, result.Output)
	}
	return result, nil
}

func ensureDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return false, fmt.Errorf("%w: %s", ErrNotDirectory, path)
		}
		return false, nil
	}
	if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return false, err
	}
	return true, nil
}

func truncateOutput(output string) string {
	if len(output) <= maxHookOutputBytes {
		return output
	}
	return output[:maxHookOutputBytes] + "\n[truncated]"
}
