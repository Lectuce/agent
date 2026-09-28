package prompt

import (
	"agent/config"
	"agent/tool"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

var MEMORY_DIR = config.WORKDIR + "/" + ".memory"
var MEMORY_INDEX = MEMORY_DIR + "/" + "MEMORY.md"

var promptSections = map[string]string{

	"identity": "You are a coding agent. Act, don't explain.",

	"tools": "Available tools: bash, read_file, write_file.",

	"workspace": fmt.Sprintf("Working directory: %v", config.WORKDIR),

	"memory": "Relevant memories are injected below when available.",
}

type PromptContext struct {
	Memories     string   `json:"memories"`
	Workspace    string   `json:"workspace"`
	EnabledTools []string `json:"enabled_tools"`
}

var lastContextKey string
var lastPrompt string

func assembleSystemPrompt(context PromptContext) string {
	sections := make([]string, 0)

	sections = append(sections, promptSections["identity"])
	sections = append(sections, promptSections["tools"])
	sections = append(sections, promptSections["workspace"])

	memories := context.Memories
	if len(memories) > 0 {
		sections = append(sections, fmt.Sprintf("Relevant memories:\n%v\n", memories))
	}
	var s string
	for _, section := range sections {
		s += section + "\n"
	}
	return s
}

func GetSystemPrompt(context PromptContext) string {
	data, err := json.Marshal(context)
	if err != nil {
		return assembleSystemPrompt(context)
	}
	key := string(data)
	if key == lastContextKey && lastPrompt != "" {
		return lastPrompt
	}
	lastContextKey = key
	lastPrompt = assembleSystemPrompt(context)
	return lastPrompt
}

func UpdateContext(context PromptContext, messages []anthropic.MessageParam) (*PromptContext, error) {
	memories := ""

	_, err := os.Stat(MEMORY_INDEX)
	if err == nil {
		data, err := os.ReadFile(MEMORY_INDEX)
		if err != nil {
			return &PromptContext{}, err
		}

		content := strings.TrimSpace(string(data))
		if content != "" {
			memories = content
		}
	}

	enabledTools := make([]string, 0, len(tool.ToolHandlers))

	for name := range tool.ToolHandlers {
		enabledTools = append(enabledTools, name)
	}

	return &PromptContext{
		EnabledTools: enabledTools,
		Workspace:    config.WORKDIR,
		Memories:     memories,
	}, nil
}
