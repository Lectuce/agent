package config

import (
	"agent/skills"
	"fmt"
	"os"
	"strconv"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const WORKDIR = "/home/yujun/projects/agent"

var Client = anthropic.NewClient(
	option.WithAPIKey(os.Getenv("API_KEY")),
	option.WithBaseURL(os.Getenv("BASE_URL")),
)
var PRIMARY_MODEL = os.Getenv("PRIMARY_MODEL")
var FALLBACK_MODEL = os.Getenv("FALLBACK_MODEL")

var KEEP_RECENT_TOOL_RESULTS = 3

func envInt64(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

var DEFAULT_MAX_TOOKENS = envInt64("DEFAULT_MAX_TOOKENS", 8000)
var ESCALATED_MAX_TOKENS = envInt64("ESCALATED_MAX_TOKENS", 64000)
var MAX_CONSECUTIVE_529 = 3
var MAX_RECOVERY_RETRIES = 3
var wd, _ = os.Getwd()

var SUBSYSTEM = []anthropic.TextBlockParam{
	{
		Text: fmt.Sprintf("You are a coding agent at %v.", wd),
	},
	{
		Text: "Complete the task you were given, then return a concise summary.",
	},
	{
		Text: "Do not delegate further.",
	},
}

func buildSystem() string {
	catalog := skills.ListSkill()

	return fmt.Sprintf("You are a coding agent at %v.\n Skills available:\n%v\n Use load_skill to get full details when needed.\n", WORKDIR, catalog)
}
