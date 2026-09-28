package memory

import (
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

type ExtractedMemory struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Body        string `json:"body"`
}

func messageText(msg anthropic.MessageParam) string {
	var texts []string

	for _, block := range msg.Content {
		if block.OfText != nil {
			texts = append(texts, block.OfText.Text)
		}
	}

	return strings.Join(texts, " ")
}

func extractResponseText(response *anthropic.Message) string {
	if response == nil {
		return ""
	}

	var texts []string

	for _, block := range response.Content {
		anyBlock := block.AsAny()

		switch b := anyBlock.(type) {
		case anthropic.TextBlock:
			texts = append(texts, b.Text)
		}
	}

	return strings.TrimSpace(strings.Join(texts, "\n"))
}
