package compact

import (
	"agent/config"
	"agent/recovery"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
)

func CompactHistory(messages []anthropic.MessageParam, ctx context.Context) ([]anthropic.MessageParam, error) {
	transcriptPath, err := writeTranscript(messages)
	if err != nil {
		return nil, err
	}
	fmt.Printf("[transcript saved: %v]\n", transcriptPath)
	summary, err := summarizeHistory(messages, ctx)
	if err != nil {
		return nil, err
	}
	result := make([]anthropic.MessageParam, 0)
	result = append(result,
		anthropic.NewUserMessage(
			anthropic.NewTextBlock(
				summary,
			),
		),
	)
	return result, nil

}

// 保存完整对话
func writeTranscript(messages []anthropic.MessageParam) (string, error) {
	err := os.MkdirAll(config.TRANSCRIPT_DIR, 0755)
	if err != nil {
		return "", err
	}
	path := filepath.Join(
		config.TRANSCRIPT_DIR,
		fmt.Sprintf("transcript_%d.jsonl", time.Now().UnixNano()),
	)

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)

	for _, msg := range messages {
		if err := encoder.Encode(msg); err != nil {
			return "", err
		}
	}

	return path, nil

}

// LLM生成摘要
func summarizeHistory(messages []anthropic.MessageParam, ctx context.Context) (string, error) {
	compactState := recovery.InitRecoveryState()
	rawData, err := json.Marshal(messages)
	conversation := string(rawData)
	if len(conversation) > 80000 {
		conversation = conversation[:80000]
	}
	conversation = conversation[:80000]
	if err != nil {
		return "", err
	}
	prompts := "Summarize this coding-agent conversation so work can continue.\n" +
		"Preserve: 1. current goal, 2. key findings/decisions, 3. files read/changed, " +
		"4. remaining work, 5. user constraints.\nBe compact but concrete.\n\n" + conversation

	response, err := recovery.WithRetry(
		func() (*anthropic.Message, error) {
			return config.Client.Messages.New(
				ctx,
				anthropic.MessageNewParams{
					MaxTokens: config.DEFAULT_COMPACT_TOOKENS,
					Model:     compactState.CurrentModel,
					Messages: []anthropic.MessageParam{
						anthropic.NewUserMessage(
							anthropic.NewTextBlock(prompts),
						),
					},
				},
			)
		},
		compactState,
		10,
	)
	if err != nil {
		return "", err
	}
	texts := make([]string, 0)
	for _, block := range response.Content {
		anyBlock := block.AsAny()

		switch b := anyBlock.(type) {
		case anthropic.TextBlock:
			texts = append(texts, b.Text)
		}
	}

	summary := strings.TrimSpace(strings.Join(texts, "\n"))

	if summary == "" {
		summary = "(empty summary)"
	}

	return summary, nil
}

func ReactiveCompact(messages []anthropic.MessageParam, ctx context.Context) ([]anthropic.MessageParam, error) {
	_, err := writeTranscript(messages)
	if err != nil {
		return nil, err
	}
	summary, err := summarizeHistory(messages, ctx)
	if err != nil {
		return nil, err
	}
	tailStart := max(0, len(messages)-5)
	if tailStart > 0 && tailStart < len(messages) && isToolResultMessage(messages[tailStart]) && messageHasToolUse(messages[tailStart-1]) {
		tailStart -= 1
	}
	result := make([]anthropic.MessageParam, 0)
	result = append(result, anthropic.NewUserMessage(
		anthropic.NewTextBlock(
			fmt.Sprintf("[Reactive compact]\n\n%v", summary),
		),
	))
	result = append(result, messages[tailStart:]...)
	return result, nil
}
