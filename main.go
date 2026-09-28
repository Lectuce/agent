package main

import (
	"agent/hook"
	"agent/loop"
	"agent/prompt"
	"bufio"
	"context"
	"fmt"
	"os"
)

func main() {
	hook.Register()
	fmt.Println("输入问题，回车发送。输入 q 退出。")
	fmt.Printf("s04 >> ")
	scanner := bufio.NewScanner(os.Stdin)
	promptContext, err := prompt.UpdateContext(prompt.PromptContext{}, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	for {
		ok := scanner.Scan()
		if !ok {
			return
		}
		query := scanner.Text()
		if query == "q" || query == "exit" || query == "" {
			return
		}
		hook.TriggerHooks(hook.UserPromptSubmit,
			&hook.HookContext{
				Query: query,
			},
		)
		ctx := context.Background()
		err := loop.AgentLoop(query, ctx, promptContext)
		if err != nil {
			fmt.Printf("agent loop error: %v\n", err)
			return
		}
		// fmt.Println("输入问题，回车发送。输入 q 退出。")
		fmt.Printf("s04 >> ")

	}

}
