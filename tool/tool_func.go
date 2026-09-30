package tool

import (
	"agent/config"
	"agent/skills"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Knetic/govaluate"
)

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

func loadSkill(input map[string]any) (string, error) {
	name, ok := input["name"].(string)
	if !ok || name == "" {
		return "", fmt.Errorf("name is required")
	}
	skill, ok := skills.SkillRegistry[name]
	if !ok {
		return "", fmt.Errorf("Skill not found: %v\n", skill)
	}
	return skill.Content, nil

}

func runCalculator(input map[string]any) (string, error) {
	expression, ok := input["expression"].(string)
	if !ok || expression == "" {
		return "", fmt.Errorf("expression is required")
	}

	expr, err := govaluate.NewEvaluableExpression(expression)
	if err != nil {
		return "", fmt.Errorf("invalid expression: %w", err)
	}

	result, err := expr.Evaluate(nil)
	if err != nil {
		return "", fmt.Errorf("calculate expression: %w", err)
	}

	return fmt.Sprintf("%v", result), nil
}
