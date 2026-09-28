package compact

import "github.com/anthropics/anthropic-sdk-go"

func EstimateSize(messages []anthropic.MessageParam) int {
	total := 0
	for _, message := range messages {
		for _, block := range message.Content {
			if block.OfText != nil {
				total += len(block.OfText.Text)
			}

			if block.OfToolResult != nil {
				for _, content := range block.OfToolResult.Content {
					if content.OfText != nil {
						total += len(content.OfText.Text)
					}
				}
			}
		}
	}
	return total

}
