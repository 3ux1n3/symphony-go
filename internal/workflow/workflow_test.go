package workflow

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseWithFrontMatter(t *testing.T) {
	config, prompt, err := Parse([]byte("---\ntracker:\n  kind: clickup\n---\nHello\n"))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	tracker := config["tracker"].(map[string]any)
	if tracker["kind"] != "clickup" {
		t.Fatalf("tracker.kind = %v, want clickup", tracker["kind"])
	}
	if prompt != "Hello" {
		t.Fatalf("prompt = %q, want Hello", prompt)
	}
}

func TestParseWithoutFrontMatter(t *testing.T) {
	config, prompt, err := Parse([]byte("  Hello without config\n"))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(config) != 0 {
		t.Fatalf("config length = %d, want 0", len(config))
	}
	if prompt != "Hello without config" {
		t.Fatalf("prompt = %q", prompt)
	}
}

func TestParseRejectsNonMapFrontMatter(t *testing.T) {
	_, _, err := Parse([]byte("---\n- one\n- two\n---\nPrompt"))
	if !errors.Is(err, ErrFrontMatterNotMap) {
		t.Fatalf("error = %v, want ErrFrontMatterNotMap", err)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "WORKFLOW.md"))
	if !errors.Is(err, ErrMissingWorkflowFile) {
		t.Fatalf("error = %v, want ErrMissingWorkflowFile", err)
	}
}

func TestLoadReadsWorkflow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "WORKFLOW.md")
	if err := os.WriteFile(path, []byte("---\ntracker:\n  kind: clickup\n---\nPrompt"), 0o600); err != nil {
		t.Fatal(err)
	}

	def, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if def.Path != path {
		t.Fatalf("Path = %q, want %q", def.Path, path)
	}
	if def.PromptTemplate != "Prompt" {
		t.Fatalf("PromptTemplate = %q", def.PromptTemplate)
	}
}

func TestRenderStrictSuccess(t *testing.T) {
	out, err := Render("Task {{ .task.identifier }}: {{ .task.title }}", map[string]any{
		"task": map[string]any{
			"identifier": "CU-1",
			"title":      "Build it",
		},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if out != "Task CU-1: Build it" {
		t.Fatalf("out = %q", out)
	}
}

func TestRenderStrictMissingKey(t *testing.T) {
	_, err := Render("{{ .task.missing }}", map[string]any{"task": map[string]any{}})
	if !errors.Is(err, ErrTemplateRender) {
		t.Fatalf("error = %v, want ErrTemplateRender", err)
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("error = %v, want missing key detail", err)
	}
}
