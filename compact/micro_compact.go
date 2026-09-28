package compact

import (
	"agent/config"

	"github.com/anthropics/anthropic-sdk-go"
)

type toolResultPos struct {
	messageIndex int
	blockIndex   int
}

// 旧工具结果占位
func MicroCompact(messages []anthropic.MessageParam) []anthropic.MessageParam {
	toolResults := collectToolResultBlocks(messages)
	if len(toolResults) < config.KEEP_RECENT_TOOL_RESULTS {
		return messages
	}
	oldResults := toolResults[:len(toolResults)-config.KEEP_RECENT_TOOL_RESULTS]
	for _, pos := range oldResults {
		block := messages[pos.messageIndex].Content[pos.blockIndex]
		if block.OfToolResult == nil {
			continue
		}
		toolResult := block.OfToolResult

		totalLen := 0
		for _, content := range toolResult.Content {
			if content.OfText != nil {
				totalLen += len(content.OfText.Text)
			}
		}

		if totalLen <= 120 {
			continue
		}

		isError := false
		if toolResult.IsError.Valid() {
			isError = toolResult.IsError.Value
		}

		// 压缩
		messages[pos.messageIndex].Content[pos.blockIndex] =
			anthropic.NewToolResultBlock(
				toolResult.ToolUseID,
				"[Earlier tool result compacted. Re-run if needed.]",
				isError,
			)

	}
	return messages
}

func collectToolResultBlocks(messages []anthropic.MessageParam) []toolResultPos {
	blocks := []toolResultPos{}
	for mi, msg := range messages {
		if msg.Role != anthropic.MessageParamRoleUser {
			continue
		}

		for bi, block := range msg.Content {
			blockType := block.GetType()

			if blockType != nil && *blockType == "tool_result" {
				blocks = append(blocks, toolResultPos{
					messageIndex: mi,
					blockIndex:   bi,
				})
			}
		}
	}

	return blocks
}
