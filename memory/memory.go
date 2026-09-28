package memory

import (
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
)

func ReactiveCompact(messages []anthropic.MessageParam) []anthropic.MessageParam {
	fmt.Println("  \033[31m[reactive compact] trimming to last 5 messages\033[0m")
	start := len(messages) - 5
	if start < 0 {
		start = 0
	}

	tail := messages[start:]
	result := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(
			"[Reactive compact] Earlier conversation trimmed. Continue from where you left off.",
		)),
	}
	result = append(result, tail...)
	return result
}
