package memory

import (
	"agent/config"
	"agent/recovery"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

func selectRelevantMemories(messages []anthropic.MessageParam, ctx context.Context, maxItems int) ([]string, error) {
	files, err := listMemoryFiles()
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}

	// 收集最近三条
	recentTexts := make([]string, 0)
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != anthropic.MessageParamRoleUser {
			continue
		}
		text := strings.TrimSpace(messageText(msg))
		if text == "" {
			continue
		}
		recentTexts = append(recentTexts, text)

		if len(recentTexts) >= 3 {
			break
		}
	}
	slices.Reverse(recentTexts)

	recent := strings.Join(recentTexts, "\n")

	if len(recent) > 2000 {
		recent = recent[:2000]
	}

	if strings.TrimSpace(recent) == "" {
		return []string{}, nil
	}

	// memory catalog
	var catalogLines []string

	for i, file := range files {
		catalogLines = append(catalogLines, fmt.Sprintf("%v: %v — %v", i, file.Name, file.Description))
	}

	catalog := strings.Join(catalogLines, "\n")

	prompt := "Given the recent conversation and the memory catalog below, " +
		"select the indices of memories that are clearly relevant. " +
		"Return ONLY a JSON array of integers, e.g. [0, 3]. " +
		"If none are relevant, return [].\n\n" +
		"Recent conversation:\n" + recent + "\n\n" +
		"Memory catalog:\n" + catalog

	state := recovery.InitRecoveryState()

	response, err := recovery.WithRetry(
		func() (*anthropic.Message, error) {
			return config.Client.Messages.New(
				ctx,
				anthropic.MessageNewParams{
					Model:     state.CurrentModel,
					MaxTokens: config.DEFAULT_MEMORY_TOOKENS,
					Messages: []anthropic.MessageParam{
						anthropic.NewUserMessage(
							anthropic.NewTextBlock(prompt),
						),
					},
				},
			)
		},
		state,
		10,
	)

	if err == nil {
		text := extractResponseText(response)

		indices, ok := extractIntArray(text)
		if ok {
			var selected []string

			for _, idx := range indices {
				if idx < 0 || idx >= len(files) {
					continue
				}

				selected = append(
					selected,
					files[idx].Filename,
				)

				if len(selected) >= maxItems {
					break
				}
			}

			return selected, nil
		}
	}

	// LLM 失败时 fallback：关键词匹配
	keywords := []string{}
	for _, word := range strings.Fields(strings.ToLower(recent)) {
		if len(word) > 3 {
			keywords = append(keywords, word)
		}
	}
	var selected []string
	for _, file := range files {
		text := strings.ToLower(
			file.Name + " " + file.Description,
		)
		matched := false
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				matched = true
				break
			}
		}
		if matched {
			selected = append(selected, file.Filename)

			if len(selected) >= maxItems {
				break
			}
		}
	}

	return selected, nil
}

// 从LLM返回值里找[0, 3]
func extractIntArray(text string) ([]int, bool) {
	re := regexp.MustCompile(`(?s)\[.*?\]`)

	match := re.FindString(text)
	if match == "" {
		return nil, false
	}

	var indices []int

	if err := json.Unmarshal([]byte(match), &indices); err != nil {
		return nil, false
	}

	return indices, true
}
