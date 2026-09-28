package loop

import (
	"agent/config"
	"agent/hook"
	"agent/memory"
	"agent/prompt"
	"agent/recovery"
	"agent/subagent"
	"agent/tool"
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
)

var roundsSinceTodo = 0

func AgentLoop(query string, ctx context.Context, promptCtx *prompt.PromptContext) error {
	handlers := make(map[string]tool.ToolHandler, 0)
	for name, handler := range tool.ToolHandlers {
		handlers[name] = handler
	}

	handlers["task"] = func(input map[string]any) (string, error) {
		return subagent.SpawnSubagent(input, ctx)
	}

	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(
			anthropic.NewTextBlock(query),
		),
	}
	rawsystem := prompt.GetSystemPrompt(*promptCtx)
	system := []anthropic.TextBlockParam{
		{
			Text: rawsystem,
		},
	}
	state := recovery.InitRecoveryState()
	maxTookens := config.DEFAULT_MAX_TOOKENS

	for {
		if roundsSinceTodo >= 3 && len(messages) > 0 {
			messages = append(messages, anthropic.NewUserMessage(
				anthropic.NewTextBlock("<reminder>Update your todos.</reminder>"),
			))
			roundsSinceTodo = 0
		}
		tools := make([]anthropic.ToolUnionParam, len(tool.ToolParams))
		for i, toolParam := range tool.ToolParams {
			tools[i] = anthropic.ToolUnionParam{
				OfTool: &toolParam,
			}
		}
		message, err := recovery.WithRetry(
			func() (*anthropic.Message, error) {
				return config.Client.Messages.New(
					ctx,
					anthropic.MessageNewParams{
						MaxTokens: maxTookens,
						Messages:  messages,
						Model:     state.CurrentModel,
						Tools:     tools,
						System:    system,
					},
				)
			},
			state,
			10,
		)
		if err != nil {
			if recovery.IsPromptTooLongError(err) {
				if !state.HasAttemptedReactiveCompact {
					messages = memory.ReactiveCompact(messages)
					state.HasAttemptedReactiveCompact = true
					continue
				}
				fmt.Println("  \033[31m[unrecoverable] still too long after compact\033[0m")
				messages = append(messages, anthropic.NewAssistantMessage(
					anthropic.NewTextBlock("[Error] Context too large, cannot continue."),
				))
			}
			return err
		}

		fmt.Printf("  \033[90m[turn] stop_reason=%s input_tokens=%d output_tokens=%d max_tokens=%d\033[0m\n",
			message.StopReason, message.Usage.InputTokens, message.Usage.OutputTokens, maxTookens)

		if message.StopReason == anthropic.StopReasonMaxTokens {
			if !state.HasEscalated {
				prev := maxTookens
				maxTookens = config.ESCALATED_MAX_TOKENS
				state.HasEscalated = true
				fmt.Printf("  \033[33m[max_tokens] escalating max_tokens %d -> %d (stop_reason=max_tokens)\033[0m\n", prev, maxTookens)
				continue
			}
			messages = append(messages, anthropic.NewAssistantMessage(message.ToParam().Content...))
			if state.RecoveryCount < config.MAX_RECOVERY_RETRIES {
				state.RecoveryCount++
				fmt.Printf("  \033[33m[max_tokens] truncated again, requesting continuation %d/%d (max_tokens=%d)\033[0m\n", state.RecoveryCount, config.MAX_RECOVERY_RETRIES, maxTookens)
				messages = append(messages, anthropic.NewUserMessage(
					anthropic.NewTextBlock("Output token limit hit. Resume directly — no apology, no recap. Pick up mid-thought."),
				))
				continue
			}
			return fmt.Errorf("still truncated after %d continuations", config.MAX_RECOVERY_RETRIES)

		}

		messages = append(messages, anthropic.NewAssistantMessage(message.ToParam().Content...))

		if message.StopReason != anthropic.StopReasonToolUse {
			for _, block := range message.Content {
				if text, ok := block.AsAny().(anthropic.TextBlock); ok {
					fmt.Println(text.Text)
				}
			}
			return nil
		}

		roundsSinceTodo++

		// 工具执行
		toolResults := []anthropic.ContentBlockParamUnion{}
		for _, block := range message.Content {
			switch block := block.AsAny().(type) {
			case anthropic.ToolUseBlock:
				var input map[string]any
				err = json.Unmarshal([]byte(block.JSON.Input.Raw()), &input)
				if err != nil {
					return err
				}
				blocked := hook.TriggerHooks(hook.PreToolUse, &hook.HookContext{
					Block: &block,
					Args:  input,
				})
				if len(blocked) > 0 {
					toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, blocked, true))
					continue
				}
				handler, ok := handlers[block.Name]
				if !ok {
					return fmt.Errorf("unknown tool: %s", block.Name)
				}
				output, err := handler(input)
				if err != nil {
					return err
				}
				hook.TriggerHooks(hook.PostToolUse, &hook.HookContext{
					Block:  &block,
					Output: output,
					Args:   input,
				})
				if len(output) > 200 {
					fmt.Println(output[:200])
				}
				toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, output, false))
			}
		}
		if len(toolResults) == 0 {
			break
		}
		messages = append(messages, anthropic.NewUserMessage(toolResults...))
		promptCtx, err = prompt.UpdateContext(*promptCtx, messages)
		if err != nil {
			return err
		}
		rawsystem = prompt.GetSystemPrompt(*promptCtx)
		system = []anthropic.TextBlockParam{
			{
				Text: rawsystem,
			},
		}

	}
	return nil
}
