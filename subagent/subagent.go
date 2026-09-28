package subagent

import (
	"agent/config"
	"agent/hook"
	"agent/tool"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

var subTools = []anthropic.ToolParam{
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
}

func ExtractText(content []anthropic.ContentBlockParamUnion) string {
	var texts []string

	for _, block := range content {
		blockType := block.GetType()

		if blockType != nil && *blockType == "text" {
			text := block.GetText()
			if text != nil {
				texts = append(texts, *text)
			}
		}
	}

	return strings.Join(texts, "\n")
}

func SpawnSubagent(input map[string]any, ctx context.Context) (string, error) {
	// description string, ctx context.Context
	description, ok := input["description"].(string)
	if !ok {
		return "", fmt.Errorf("description is required")
	}

	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(
			description,
		)),
	}
	subts := make([]anthropic.ToolUnionParam, len(subTools))
	for i, t := range subTools {
		subts[i] = anthropic.ToolUnionParam{
			OfTool: &t,
		}
	}
	for i := 0; i < 30; i++ {
		message, err := config.Client.Messages.New(
			ctx,
			anthropic.MessageNewParams{
				MaxTokens: config.DEFAULT_MAX_TOOKENS,
				Model:     config.PRIMARY_MODEL,
				System:    config.SUBSYSTEM,
				Messages:  messages,
				Tools:     subts,
			},
		)
		if err != nil {
			return "", err
		}

		messages = append(messages, anthropic.NewAssistantMessage(message.ToParam().Content...))
		if message.StopReason != anthropic.StopReasonToolUse {
			return ExtractText(message.ToParam().Content), nil
		}

		toolResults := make([]anthropic.ContentBlockParamUnion, 0)
		for _, block := range message.Content {
			switch block := block.AsAny().(type) {
			case anthropic.ToolUseBlock:
				var input map[string]any
				err = json.Unmarshal([]byte(block.JSON.Input.Raw()), &input)
				if err != nil {
					return "", err
				}
				blocked := hook.TriggerHooks(hook.PreToolUse, &hook.HookContext{
					Block: &block,
					Args:  input,
				})
				if blocked != "" {
					toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, blocked, true))
					continue
				}
				handler, ok := tool.SubHandlers[block.Name]
				if !ok {
					return "", fmt.Errorf("unknown tool: %v", block.Name)
				}
				output, err := handler(input)
				if err != nil {
					return "", err
				}
				hook.TriggerHooks(hook.PostToolUse, &hook.HookContext{
					Block:  &block,
					Output: output,
				})
				toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, output, false))
			}

		}
		if len(toolResults) == 0 {
			return "", fmt.Errorf("subagent produced tool_use but no tool results")
		}
		messages = append(messages, anthropic.NewUserMessage(toolResults...))

	}
	return ExtractText(messages[len(messages)-1].Content), nil
}
