package memory

import (
	"agent/config"

	"github.com/anthropics/anthropic-sdk-go"
)

type toolResultPos struct {
	messageIndex int
	blockIndex   int
}

func MicroCompact(messages []anthropic.MessageParam) []anthropic.MessageParam {
	toolResults := collectToolResultBlocks(messages)
	if len(toolResults) < config.KEEP_RECENT_TOOL_RESULTS {
		return messages
	}
	oldResults := toolResults[:len(toolResults)-config.KEEP_RECENT_TOOL_RESULTS]
	for _, pos := range oldResults {
		block := messages[pos.messageIndex].Content[pos.blockIndex]
		text := block.GetContent()
		if text == nil {
			continue
		}
	}

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
