package memory

import "github.com/anthropics/anthropic-sdk-go"

func InjectMemories(messages []anthropic.MessageParam, memoryTurn int, memories string) []anthropic.MessageParam {

	result := append([]anthropic.MessageParam{}, messages...)

	msg := messages[memoryTurn]

	content := make([]anthropic.ContentBlockParamUnion, 0, len(msg.Content))

	for _, block := range msg.Content {
		if block.OfText != nil {
			content = append(
				content,
				anthropic.NewTextBlock(
					memories+"\n\n"+block.OfText.Text,
				),
			)
			continue
		}

		content = append(content, block)
	}

	result[memoryTurn] = anthropic.NewUserMessage(content...)

	return result
}
