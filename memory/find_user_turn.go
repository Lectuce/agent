package memory

import "github.com/anthropics/anthropic-sdk-go"

func FindUserTurn(messages []anthropic.MessageParam, query string) int {
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]

		if message.Role != anthropic.MessageParamRoleUser {
			continue
		}

		for _, block := range message.Content {
			if block.OfText != nil && block.OfText.Text == query {
				return i
			}
		}
	}

	return -1
}
