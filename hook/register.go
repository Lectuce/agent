package hook

func Register() {
	registerHook(UserPromptSubmit, contextInjectHook)
	registerHook(PreToolUse, permissionHook)
	registerHook(PreToolUse, logHook)
	registerHook(PostToolUse, largeOutputHook)
	registerHook(Stop, summaryHook)
}
