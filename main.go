package main

import (
	"agent/hook"
	"agent/loop"
	"agent/prompt"
	"agent/session"
	"agent/skills"
	"context"
	"fmt"

	"github.com/chzyer/readline"
)

func main() {
	err := skills.ScanSkill()
	if err != nil {
		fmt.Println(err.Error())
	}
	hook.Register()
	fmt.Println("输入问题，回车发送。输入 q 退出。")
	sessionManager := session.NewSessionManager()

	r, err := readline.New("agent[default] >> ")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer r.Close()
	promptContext, err := prompt.UpdateContext(prompt.PromptContext{}, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for {
		r.SetPrompt(fmt.Sprintf("agent[%s] >> ", sessionManager.Current))

		query, err := r.Readline()
		if query == "q" || query == "exit" || query == "" {
			return
		}

		if sessionManager.HandleCommand(query) {
			continue
		}

		hookCtx := &hook.HookContext{
			Query: query,
		}
		hook.TriggerHooks(hook.UserPromptSubmit, hookCtx)
		query = hookCtx.Query

		ctx := context.Background()
		err = loop.AgentLoop(query, ctx, promptContext, sessionManager.CurrentSession())
		if err != nil {
			fmt.Printf("agent loop error: %v\n", err)
			return
		}

	}

}
