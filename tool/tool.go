package tool

import (
	"sort"

	"github.com/anthropics/anthropic-sdk-go"
)

var ClientTools = []anthropic.ToolParam{
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

	{
		Name:        "load_skill",
		Description: anthropic.String("Load the full content of a skill by name."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]any{
				"name": map[string]any{
					"type": "string",
				},
			},
			Required: []string{"name"},
		},
	},

	{
		Name:        "calculator",
		Description: anthropic.String("Evaluate a mathematical expression."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Type: "object",
			Properties: map[string]any{
				"expression": map[string]any{
					"type":        "string",
					"description": "Evaluate an arithmetic expression containing numbers, parentheses, and arithmetic operators such as +, -, *, /, and %.",
				},
			},
			Required: []string{"expression"},
		},
	},
}

var ServerTools = []anthropic.ToolUnionParam{
	{
		OfWebSearchTool20250305: &anthropic.WebSearchTool20250305Param{
			MaxUses: anthropic.Int(5),
		},
	},
}

func BuildTools() []anthropic.ToolUnionParam {
	tools := make([]anthropic.ToolUnionParam, 0, len(ClientTools)+len(ServerTools))

	for i := range ClientTools {
		toolParam := ClientTools[i]

		tools = append(tools,
			anthropic.ToolUnionParam{
				OfTool: &toolParam,
			},
		)
	}

	tools = append(tools, ServerTools...)

	return tools
}

func ToolNames() []string {
	names := make([]string, 0)

	for _, clientTool := range ClientTools {
		names = append(names, string(clientTool.Name))
	}

	names = append(names, "web_search")
	sort.Strings(names)

	return names
}
