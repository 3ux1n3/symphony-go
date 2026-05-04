package workflow

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

var (
	ErrMissingWorkflowFile     = errors.New("missing_workflow_file")
	ErrWorkflowParse           = errors.New("workflow_parse_error")
	ErrFrontMatterNotMap       = errors.New("workflow_front_matter_not_a_map")
	ErrTemplateParse           = errors.New("template_parse_error")
	ErrTemplateRender          = errors.New("template_render_error")
	ErrUnterminatedFrontMatter = errors.New("workflow_unterminated_front_matter")
)

type Definition struct {
	Path           string
	Config         map[string]any
	PromptTemplate string
}

func Load(path string) (Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Definition{}, fmt.Errorf("%w: %s", ErrMissingWorkflowFile, path)
		}
		return Definition{}, fmt.Errorf("%w: read %s: %v", ErrWorkflowParse, path, err)
	}

	config, body, err := Parse(data)
	if err != nil {
		return Definition{}, err
	}

	return Definition{
		Path:           path,
		Config:         config,
		PromptTemplate: body,
	}, nil
}

func Parse(data []byte) (map[string]any, string, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") && strings.TrimSpace(text) != "---" {
		return map[string]any{}, strings.TrimSpace(text), nil
	}

	lines := strings.Split(text, "\n")
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return nil, "", fmt.Errorf("%w: missing closing ---", ErrUnterminatedFrontMatter)
	}

	frontMatter := strings.Join(lines[1:end], "\n")
	body := strings.TrimSpace(strings.Join(lines[end+1:], "\n"))
	if strings.TrimSpace(frontMatter) == "" {
		return map[string]any{}, body, nil
	}

	var decoded any
	if err := yaml.Unmarshal([]byte(frontMatter), &decoded); err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrWorkflowParse, err)
	}

	config, ok := normalizeMap(decoded).(map[string]any)
	if !ok {
		return nil, "", ErrFrontMatterNotMap
	}

	return config, body, nil
}

func Render(prompt string, data any) (string, error) {
	tmpl, err := template.New("workflow_prompt").Option("missingkey=error").Parse(prompt)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTemplateParse, err)
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("%w: %v", ErrTemplateRender, err)
	}

	return out.String(), nil
}

func normalizeMap(v any) any {
	switch value := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			out[k] = normalizeMap(v)
		}
		return out
	case map[any]any:
		out := make(map[string]any, len(value))
		for k, v := range value {
			key, ok := k.(string)
			if !ok {
				key = fmt.Sprint(k)
			}
			out[key] = normalizeMap(v)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, item := range value {
			out[i] = normalizeMap(item)
		}
		return out
	default:
		return value
	}
}
