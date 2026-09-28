package memory

import (
	"context"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

func LoadMemories(messages []anthropic.MessageParam, ctx context.Context) (string, error) {
	selectedFiles, err := selectRelevantMemories(messages, ctx, 5)
	if err != nil {
		return "", err
	}

	if len(selectedFiles) == 0 {
		return "", nil
	}

	parts := []string{"<relevant_memories>"}

	for _, filename := range selectedFiles {
		content, err := readMemoryFile(filename)
		if err != nil {
			continue
		}

		if strings.TrimSpace(content) != "" {
			parts = append(parts, content)
		}
	}

	parts = append(parts, "</relevant_memories>")

	return strings.Join(parts, "\n\n"), nil
}
