package tool

import (
	"agent/config"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

var ToolParams = []anthropic.ToolParam{
	{
		Name:        "bash",
		Description: anthropic.String("Run a shell command."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]interface{}{
				"command": map[string]interface{}{
					"type": "string",
				},
			},
			Required: []string{"command"},
		},
	},
	{
		Name:        "read_file",
		Description: anthropic.String("Read file contents."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]interface{}{
				"path": map[string]interface{}{
					"type": "string",
				},
				"limit": map[string]interface{}{
					"type": "integer",
				},
			},
			Required: []string{"path"},
		},
	},
	{
		Name:        "write_file",
		Description: anthropic.String("Write content to a file."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]interface{}{
				"path": map[string]interface{}{
					"type": "string",
				},
				"content": map[string]interface{}{
					"type": "string",
				},
			},
			Required: []string{"path", "content"},
		},
	},
	{
		Name:        "edit_file",
		Description: anthropic.String("Replace exact text in a file once."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]interface{}{
				"path": map[string]interface{}{
					"type": "string",
				},
				"old_text": map[string]interface{}{
					"type": "string",
				},
				"new_text": map[string]interface{}{
					"type": "string",
				},
			},
			Required: []string{"path", "old_text", "new_text"},
		},
	},
	{
		Name:        "glob",
		Description: anthropic.String("Find files matching a glob pattern."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]interface{}{
				"pattern": map[string]interface{}{
					"type": "string",
				},
			},
			Required: []string{"pattern"},
		},
	},
	{
		Name:        "todo_write",
		Description: anthropic.String("Create and manage a task list ..."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]interface{}{
				"todos": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"content": map[string]interface{}{
								"type": "string",
							},
							"status": map[string]interface{}{
								"type": "string",
								"enum": []string{"pending", "in_progress", "completed"},
							},
						},
					},
				},
			},
			Required: []string{"todos"},
		},
	},
	{
		Name:        "task",
		Description: anthropic.String("Launch a subagent to handle a complex subtask. Returns only the final conclusion."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]interface{}{
				"description": map[string]interface{}{
					"type": "string",
				},
			},
			Required: []string{"description"},
		},
	},
}

type ToolHandler func(map[string]any) (string, error)

var ToolHandlers = map[string]ToolHandler{
	"bash":       runBash,
	"read_file":  runRead,
	"write_file": runWrite,
	"edit_file":  runEdit,
	"glob":       runGlob,
	"todo_write": runTodoWrite,
}

var SubHandlers = map[string]ToolHandler{
	"bash":       runBash,
	"read_file":  runRead,
	"write_file": runWrite,
	"edit_file":  runEdit,
	"glob":       runGlob,
}

func runBash(input map[string]any) (string, error) {
	// command string
	command := input["command"].(string)
	dangerous := []string{"rm -rf /", "sudo", "shutdown", "reboot", "> /dev/"}
	for _, danger := range dangerous {
		if strings.Contains(command, danger) {
			return "", fmt.Errorf("Dangerous command blocked")
		}
	}
	ctx, cancel := context.WithTimeout(
		context.Background(),
		120*time.Second,
	)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = config.WORKDIR
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("Timeout (120s)")
	}
	result := strings.TrimSpace(string(out))
	if result == "" {
		if err != nil {
			return "", err
		}
		return "(no output)", nil
	}
	if len(result) > 50000 {
		result = result[:50000]
	}
	return result, nil

}

func runRead(input map[string]any) (string, error) {
	// path string, limit int
	path, err := safePath(input)
	if err != nil {
		return "", err
	}
	limit := 0

	if v, exists := input["limit"]; exists {
		switch n := v.(type) {
		case float64:
			limit = int(n)
		case int:
			limit = n
		default:
			return "", fmt.Errorf("limit must be a number")
		}
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	text := string(file)
	if limit > 0 && len(file) > limit {
		text = string(text)[:limit]
	}
	return text, nil
}

func runWrite(input map[string]any) (string, error) {
	// path string, content string
	path, err := safePath(input)
	if err != nil {
		return "", err
	}

	content := input["content"].(string)
	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("wrote %v bytes to %v\n", len([]byte(content)), path), nil
}

func runEdit(input map[string]any) (string, error) {
	// path string, old_text string, new_text string
	path, err := safePath(input)
	if err != nil {
		return "", err
	}
	oldText := input["old_text"].(string)
	newText := input["new_text"].(string)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	text := string(data)
	if !strings.Contains(text, oldText) {
		return "", fmt.Errorf("text not found")
	}
	text = strings.Replace(text, oldText, newText, 1)
	err = os.WriteFile(path, []byte(text), 0644)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Edited %v\n", path), nil
}

func runGlob(input map[string]any) (string, error) {
	// pattern string
	pattern := input["pattern"].(string)
	path := filepath.Join(config.WORKDIR, pattern)
	matches, err := filepath.Glob(path)
	if err != nil {
		return "", err
	}
	for i, m := range matches {
		rel, err := filepath.Rel(config.WORKDIR, m)
		if err == nil {
			matches[i] = rel
		}
	}

	return strings.Join(matches, "\n"), nil
}

func safePath(input map[string]any) (string, error) {
	path, ok := input["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(config.WORKDIR, path)
	}
	path = filepath.Clean(path)

	rel, err := filepath.Rel(config.WORKDIR, path)
	if err != nil {
		return "", err
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes workspace: %s", path)
	}

	return path, nil
}
