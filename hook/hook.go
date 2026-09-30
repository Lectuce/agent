package hook

import (
	"agent/config"
	"agent/tool"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
)

type HookContext struct {
	Query    string
	Block    *anthropic.ToolUseBlock
	Args     map[string]any
	Output   string
	Messages []anthropic.MessageParam
}

type callback func(*HookContext) string

const (
	UserPromptSubmit = "UserPromptSubmit"
	PreToolUse       = "PreToolUse"
	PostToolUse      = "PostToolUse"
	Stop             = "Stop"
)

var Hooks = map[string][]callback{
	// 触发时机（用户输入提交后、进入 LLM 前）	用途（输入验证、注入上下文）
	UserPromptSubmit: make([]callback, 0),
	// 触发时机（工具执行前）	用途（权限检查、日志记录）
	PreToolUse: make([]callback, 0),
	// 触发时机（工具执行后）	用途（副作用（自动 git add 等）、输出检查）
	PostToolUse: make([]callback, 0),
	// 触发时机（循环即将退出时）	用途（收尾清理（CC 还支持强制续跑））
	Stop: make([]callback, 0),
}

func registerHook(event string, callback func(*HookContext) string) {
	Hooks[event] = append(Hooks[event], callback)
}

func TriggerHooks(event string, ctx *HookContext) string {
	for _, callback := range Hooks[event] {
		result := callback(ctx)
		if len(result) > 0 {
			return result
		}
	}
	return ""
}

func contextInjectHook(ctx *HookContext) string {
	ctx.Query = fmt.Sprintf("<context>\nWorking directory: %s\n</context>\n\n%s", config.WORKDIR, ctx.Query)
	// Inject current working directory info into every prompt.
	fmt.Printf("\033[90m[HOOK] UserPromptSubmit: working in %v\033[0m\n", config.WORKDIR)
	return "" // return None = no modification, let prompt through
}

func permissionHook(ctx *HookContext) string {
	block := ctx.Block
	if block.Name == "bash" {
		err := tool.CheckDenyList(ctx.Args["command"].(string))
		if err != nil {
			return err.Error()
		}
	}
	reason := tool.CheckRules(block.Name, ctx.Args)
	if len(reason) > 0 {
		decision := tool.AskUser(block.Name, ctx.Args, reason)
		if decision == "deny" {
			return fmt.Errorf("user deny").Error()
		}
		return ""
	}
	return ""
}

func logHook(ctx *HookContext) string {
	fmt.Printf("[Hook] %v(...)\n", ctx.Block.Name)
	return ""
}

func largeOutputHook(ctx *HookContext) string {
	if len(ctx.Output) > 100000 {
		fmt.Printf("[HOOK] ⚠ Large output from %v\n", ctx.Block.Name)
	}
	return ""
}

func summaryHook(ctx *HookContext) string {
	toolCount := 0
	for _, message := range ctx.Messages {
		for _, block := range message.Content {
			blockType := block.GetType()
			if blockType != nil {
				if *blockType == "tool_result" || *blockType == "web_search_tool_result" {
					toolCount++
				}
			}
		}
	}
	fmt.Printf(
		"\033[90m[HOOK] Stop: session used %d tool calls\033[0m\n",
		toolCount,
	)
	return ""

}
