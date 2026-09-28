package memory

import (
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
)

// 裁剪掉旧message
func SnipCompact(messages []anthropic.MessageParam, maxMessage int) []anthropic.MessageParam {
	if len(messages) < maxMessage {
		return messages
	}
	// 保留头部3条和尾部47条
	headEnd := 3
	tailStart := len(messages) - (maxMessage - 3)
	if messageHasToolUse(messages[headEnd-1]) {
		for headEnd < len(messages) && isToolResultMessage(messages[headEnd]) {
			headEnd += 1
		}
	}
	if isToolResultMessage(messages[tailStart]) && messageHasToolUse(messages[tailStart-1]) {
		tailStart -= 1
	}
	snipped := tailStart - headEnd
	placeholder := anthropic.NewUserMessage(
		anthropic.NewTextBlock(fmt.Sprintf("[snipped %v messages from conversation middle]", snipped)),
	)
	result := make([]anthropic.MessageParam, 0)
	result = append(result, messages[:headEnd]...)
	result = append(result, placeholder)
	result = append(result, messages[tailStart:]...)
	return result

}

func messageHasToolUse(message anthropic.MessageParam) bool {
	if message.Role != anthropic.MessageParamRoleAssistant {
		return false
	}
	content := message.Content
	for _, block := range content {
		blockType := block.GetType()
		if blockType != nil && *blockType == "tool_use" {
			return true
		}
	}
	return false

}

func isToolResultMessage(message anthropic.MessageParam) bool {
	if message.Role != anthropic.MessageParamRoleUser {
		return false
	}
	content := message.Content
	for _, block := range content {
		blockType := block.GetType()
		if blockType != nil && *blockType == "tool_result" {
			return true
		}
	}
	return false
}
