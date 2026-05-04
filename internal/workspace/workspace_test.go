package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/3ux1n3/symphony-go/internal/config"
)

func TestSanitizeKey(t *testing.T) {
	key, err := SanitizeKey("CU 123: fix/auth")
	if err != nil {
		t.Fatalf("SanitizeKey returned error: %v", err)
	}
	if key != "CU_123__fix_auth" {
		t.Fatalf("key = %q", key)
	}
}

func TestSanitizeKeyRejectsEmptyIdentifier(t *testing.T) {
	_, err := SanitizeKey("   ")
	if !errors.Is(err, ErrInvalidIdentifier) {
		t.Fatalf("error = %v, want ErrInvalidIdentifier", err)
	}
}

func TestPrepareCreatesAndReusesWorkspace(t *testing.T) {
	root := t.TempDir()
	mgr := newTestManager(t, root, config.HooksConfig{})

	ws, err := mgr.Prepare(context.Background(), "CU-1")
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if !ws.CreatedNow {
		t.Fatalf("CreatedNow = false, want true")
	}
	if ws.Key != "CU-1" {
		t.Fatalf("Key = %q", ws.Key)
	}
	if info, err := os.Stat(ws.Path); err != nil || !info.IsDir() {
		t.Fatalf("workspace dir not created: info=%v err=%v", info, err)
	}

	reused, err := mgr.Prepare(context.Background(), "CU-1")
	if err != nil {
		t.Fatalf("second Prepare returned error: %v", err)
	}
	if reused.CreatedNow {
		t.Fatalf("CreatedNow = true on reused workspace")
	}
	if reused.Path != ws.Path {
		t.Fatalf("reused path = %q, want %q", reused.Path, ws.Path)
	}
}

func TestPrepareFailsForExistingNonDirectory(t *testing.T) {
	root := t.TempDir()
	mgr := newTestManager(t, root, config.HooksConfig{})
	path, _, err := mgr.PathForIdentifier("CU-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not a dir"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = mgr.Prepare(context.Background(), "CU-1")
	if !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("error = %v, want ErrNotDirectory", err)
	}
}

func TestEnsureInsideRootRejectsOutsidePath(t *testing.T) {
	mgr := newTestManager(t, t.TempDir(), config.HooksConfig{})
	err := mgr.EnsureInsideRoot(filepath.Join(t.TempDir(), "elsewhere"))
	if !errors.Is(err, ErrPathOutsideRoot) {
		t.Fatalf("error = %v, want ErrPathOutsideRoot", err)
	}
}

func TestAfterCreateHookRunsOnlyOnNewWorkspace(t *testing.T) {
	root := t.TempDir()
	mgr := newTestManager(t, root, config.HooksConfig{
		AfterCreate: "printf x >> hook-count",
		Timeout:     time.Second,
	})

	ws, err := mgr.Prepare(context.Background(), "CU-1")
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if _, err := mgr.Prepare(context.Background(), "CU-1"); err != nil {
		t.Fatalf("second Prepare returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(ws.Path, "hook-count"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "x" {
		t.Fatalf("hook-count = %q, want x", data)
	}
}

func TestBeforeRunHookFailure(t *testing.T) {
	mgr := newTestManager(t, t.TempDir(), config.HooksConfig{
		BeforeRun: "printf nope; exit 7",
		Timeout:   time.Second,
	})
	ws, err := mgr.Prepare(context.Background(), "CU-1")
	if err != nil {
		t.Fatal(err)
	}

	result, err := mgr.RunBeforeRun(context.Background(), ws)
	if !errors.Is(err, ErrHookFailed) {
		t.Fatalf("error = %v, want ErrHookFailed", err)
	}
	if !strings.Contains(result.Output, "nope") {
		t.Fatalf("output = %q, want hook output", result.Output)
	}
}

func TestHookTimeout(t *testing.T) {
	mgr := newTestManager(t, t.TempDir(), config.HooksConfig{
		BeforeRun: "sleep 1",
		Timeout:   10 * time.Millisecond,
	})
	ws, err := mgr.Prepare(context.Background(), "CU-1")
	if err != nil {
		t.Fatal(err)
	}

	_, err = mgr.RunBeforeRun(context.Background(), ws)
	if !errors.Is(err, ErrHookTimeout) {
		t.Fatalf("error = %v, want ErrHookTimeout", err)
	}
}

func TestRemoveRunsBeforeRemoveAndDeletesWorkspace(t *testing.T) {
	root := t.TempDir()
	marker := filepath.Join(root, "removed.txt")
	mgr := newTestManager(t, root, config.HooksConfig{
		BeforeRemove: "printf removed > ../removed.txt",
		Timeout:      time.Second,
	})
	ws, err := mgr.Prepare(context.Background(), "CU-1")
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.Remove(context.Background(), ws); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}
	if _, err := os.Stat(ws.Path); !os.IsNotExist(err) {
		t.Fatalf("workspace still exists or stat failed unexpectedly: %v", err)
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "removed" {
		t.Fatalf("marker = %q", data)
	}
}

func newTestManager(t *testing.T, root string, hooks config.HooksConfig) *Manager {
	t.Helper()
	mgr, err := NewManager(root, hooks)
	if err != nil {
		t.Fatalf("NewManager returned error: %v", err)
	}
	return mgr
}
