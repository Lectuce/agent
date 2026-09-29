# Agent

一个使用 Go 实现的最小可用 Agent。

Claude Code / Claude Code Agent Harness 的实现思路，包括Tool Use、Permission、Hooks、Todo Write、Subagent、Skills、System Prompt、Error Recovery、Context Compact、Memory、Session 和 Log。

核心 Agent Runtime 自行实现，不依赖 LangGraph、OpenHands、OpenClaw 等现成 Agent 框架。

## 运行方式

配置模型 API：

```bash
export API_KEY="..."
export BASE_URL="..."
export MODEL="..."
```
启动：

```bash
go run main.go
```
## 一、工具与执行

### 1. Agent Loop

Agent 的核心是一个持续的 LLM → Tool → Result → LLM 循环。

当模型返回 tool_use 时，Runtime 执行对应工具，并将 tool_result 追加到 messages[]，然后再次调用模型；如果模型不再需要工具，则返回最终回答。

同时通过 MAX_AGENT_ROUNDS 限制单次任务的最大循环次数。

```text
              ┌──────────────────────────────┐
              │                              │
              ▼                              │
       ┌─────────────┐                       │
       │ messages[]  │                       │
       └──────┬──────┘                       │
              │                              │
              ▼                              │
       ┌─────────────┐                       │
       │     LLM     │                       │
       └──────┬──────┘                       │
              │                              │
              ▼                              │
       ┌─────────────┐                       │
       │ tool_use ?  │                       │
       └──────┬──────┘                       │
          是  │  否                          │
              │   └──────────► 最终回答      │
              ▼                              │
       ┌─────────────┐                       │
       │  run_bash   │                       │
       └──────┬──────┘                       │
              │                              │
              ▼                              │
       ┌─────────────┐                       │
       │ tool_result │                       │
       └──────┬──────┘                       │
              │                              │
              └──────────────────────────────┘
```

### 2. Tool Use

在基础 Agent Loop 上加入统一的 Tool Registry。每个工具通过名称、描述和参数 Schema 暴露给 LLM，模型根据任务自主选择工具，Runtime 再根据工具名称分发到对应 Handler。

Client Tools 由本地 Handler 执行；Server Tool 则由模型 API Provider 执行。所有工具最终统一注册到模型请求中。

```text
              ┌───────────────────────────────────┐
              │                                   │
              ▼                                   │
       ┌─────────────┐                            │
       │ messages[]  │                            │
       └──────┬──────┘                            │
              ▼                                   │
       ┌─────────────┐                            │
       │     LLM     │                            │
       │ tools schema│                            │
       └──────┬──────┘                            │
              ▼                                   │
       ┌─────────────┐                            │
       │ tool_use ?  │                            │
       └──────┬──────┘                            │
          是  │  否                               │
              │   └────────────► 最终回答         │
              ▼                                   │
       ┌─────────────────────┐                    │
       │   Tool Dispatch     │                    │
       │ TOOL_HANDLERS[name] │                    │
       └──────────┬──────────┘                    │
                  │                               │
        ┌─────────┼──────────┐                    │
        ▼         ▼          ▼                    │
     ┌──────┐ ┌────────┐ ┌────────────┐          │
     │ bash │ │read_file│ │ calculator │          │
     └──┬───┘ └───┬────┘ └─────┬──────┘          │
        │         │            │                  │
        └─────────┴─────┬──────┘                  │
                        ▼                         │
                 ┌─────────────┐                  │
                 │ tool_result │                  │
                 └──────┬──────┘                  │
                        │                         │
                        └─────────────────────────┘
```

### 3. Permission

在工具真正执行前加入 Permission Gate。

危险操作会被拦截，并转换成错误 tool_result 返回给模型，而不是直接终止整个 Agent Loop。这样模型仍然可以根据失败结果调整下一步行为。

```text
              ┌───────────────────────────────────┐
              │                                   │
              ▼                                   │
       ┌─────────────┐                            │
       │ messages[]  │                            │
       └──────┬──────┘                            │
              ▼                                   │
       ┌─────────────┐                            │
       │     LLM     │                            │
       └──────┬──────┘                            │
              ▼                                   │
       ┌─────────────┐                            │
       │ tool_use ?  │                            │
       └──────┬──────┘                            │
          是  │  否                               │
              │   └────────────► 最终回答         │
              ▼                                   │
       ┌─────────────────┐                        │
       │ Permission Gate │                        │
       └────────┬────────┘                        │
            allow│deny                            │
           ┌─────┴────────┐                       │
           ▼              ▼                       │
   ┌─────────────┐  ┌──────────────┐              │
   │Tool Dispatch│  │ error result │              │
   └──────┬──────┘  └──────┬───────┘              │
          │                │                       │
          ▼                │                       │
   ┌─────────────┐         │                       │
   │ Execute Tool│         │                       │
   └──────┬──────┘         │                       │
          └────────┬───────┘                       │
                   ▼                              │
            ┌─────────────┐                       │
            │ tool_result │                       │
            └──────┬──────┘                       │
                   │                              │
                   └──────────────────────────────┘
```

### 4. Hooks

将 Permission、Context Injection 等逻辑从 Agent Loop 中抽离，通过Hook 扩展 Agent 行为。

当前主要包括 UserPromptSubmit、PreToolUse、PostToolUse 和 Stop。这样新增行为时不需要不断修改核心 Loop。

```text
       ┌─────────────────────┐
       │ UserPromptSubmit    │
       │ Context Injection   │
       └──────────┬──────────┘
                  │
                  ▼
              ┌──────────────────────────────────┐
              │                                  │
              ▼                                  │
       ┌─────────────┐                           │
       │ messages[]  │                           │
       └──────┬──────┘                           │
              ▼                                  │
       ┌─────────────┐                           │
       │     LLM     │                           │
       └──────┬──────┘                           │
              ▼                                  │
       ┌─────────────┐                           │
       │ tool_use ?  │                           │
       └──────┬──────┘                           │
          是  │  否                              │
              │   └──────────────► Stop Hook ──► 最终回答
              ▼                                  │
       ┌─────────────────┐                       │
       │ PreToolUse Hook │                       │
       │ Permission      │                       │
       │ Log             │                       │
       └────────┬────────┘                       │
                ▼                                │
       ┌─────────────────┐                       │
       │ Execute Tool    │                       │
       └────────┬────────┘                       │
                ▼                                │
       ┌─────────────────┐                       │
       │PostToolUse Hook │                       │
       └────────┬────────┘                       │
                ▼                                │
       ┌─────────────────┐                       │
       │   tool_result   │                       │
       └────────┬────────┘                       │
                │                                │
                └───────────────────────────────┘
```

---

## 二、规划与协调

### 5. Todo Write

增加 todo_write 工具，用于复杂任务中的显式规划。

Todo 支持 pending、in_progress 和 completed 三种状态。Agent 可以在多轮工具调用过程中持续更新任务状态，减少长任务中的目标丢失。

```text
              ┌────────────────────────────────────┐
              │                                    │
              ▼                                    │
       ┌─────────────┐                             │
       │ messages[]  │                             │
       └──────┬──────┘                             │
              ▼                                    │
       ┌─────────────┐                             │
       │     LLM     │                             │
       └──────┬──────┘                             │
              ▼                                    │
       ┌─────────────┐                             │
       │ tool_use ?  │                             │
       └──────┬──────┘                             │
          是  │  否                                │
              │   └────────────► 最终回答          │
              ▼                                    │
       ┌─────────────────┐                         │
       │  Tool Dispatch  │                         │
       └────────┬────────┘                         │
                │                                  │
        ┌───────┴───────────┐                      │
        │                   │                      │
        ▼                   ▼                      │
 ┌─────────────┐      ┌─────────────┐              │
 │ 普通工具    │      │ todo_write  │              │
 └──────┬──────┘      └──────┬──────┘              │
        │                    ▼                      │
        │            ┌─────────────────┐            │
        │            │   Todo State    │            │
        │            │ pending         │            │
        │            │ in_progress     │            │
        │            │ completed       │            │
        │            └────────┬────────┘            │
        │                     │                     │
        └──────────┬──────────┘                     │
                   ▼                                │
            ┌─────────────┐                         │
            │ tool_result │                         │
            └──────┬──────┘                         │
                   │                                │
                   └────────────────────────────────┘
```

### 6. Subagent

通过 task 工具创建 Subagent。

Subagent 拥有独立的 messages[] 和自己的 Agent Loop，用于处理一个相对独立的子任务；完成后只将最终结果返回 Main Agent，避免中间过程污染主 Context。

```text
              ┌────────────────────────────────────────┐
              │                                        │
              ▼                                        │
       ┌────────────────┐                               │
       │ Main messages[]│                               │
       └───────┬────────┘                               │
               ▼                                        │
       ┌────────────────┐                               │
       │      LLM       │                               │
       └───────┬────────┘                               │
               ▼                                        │
        ┌─────────────┐                                 │
        │ tool_use ?  │                                 │
        └──────┬──────┘                                 │
           是  │  否                                    │
               │   └─────────────► 最终回答             │
               ▼                                        │
        ┌───────────────┐                               │
        │ Tool Dispatch │                               │
        └───────┬───────┘                               │
                │                                       │
        ┌───────┴───────────┐                           │
        │                   │                           │
        ▼                   ▼                           │
 ┌─────────────┐      ┌─────────────┐                   │
 │ 普通工具    │      │ task tool   │                   │
 └──────┬──────┘      └──────┬──────┘                   │
        │                    ▼                           │
        │             ┌────────────────┐                 │
        │             │   Subagent     │                 │
        │             │ sub messages[] │◄───────────┐    │
        │             └───────┬────────┘            │    │
        │                     ▼                     │    │
        │             ┌────────────────┐            │    │
        │             │    Sub LLM     │            │    │
        │             └───────┬────────┘            │    │
        │                     ▼                     │    │
        │             ┌────────────────┐            │    │
        │             │ Sub tool_use ? │            │    │
        │             └───────┬────────┘            │    │
        │                 是  │  否                 │    │
        │                     │   └─► final result  │    │
        │                     ▼                     │    │
        │               Sub Tool ─► result ─────────┘    │
        │                         │                      │
        └─────────────────────────┼──────────────────────┘
```

### 7. Skills

Skills 用于按需加载领域知识。

系统只需要先暴露 Skill 的名称和描述；当模型判断某个 Skill 与当前任务相关时，再加载对应 SKILL.md 的完整内容，从而减少无关知识对 Context 的占用。

```text
                    ┌─────────────────┐
                    │  Skill Catalog  │
                    │ name / desc     │
                    └────────┬────────┘
                             │
                             ▼
              ┌───────────────────────────────────┐
              │                                   │
              ▼                                   │
       ┌─────────────┐                            │
       │ messages[]  │                            │
       └──────┬──────┘                            │
              ▼                                   │
       ┌─────────────┐                            │
       │     LLM     │◄──── Skill Catalog         │
       └──────┬──────┘                            │
              ▼                                   │
       ┌─────────────┐                            │
       │ tool_use ?  │                            │
       └──────┬──────┘                            │
          是  │  否                               │
              │   └────────────► 最终回答         │
              ▼                                   │
       ┌─────────────────┐                        │
       │ Tool Dispatch   │                        │
       └───────┬─────────┘                        │
               │                                  │
       ┌───────┴────────────┐                     │
       │                    │                     │
       ▼                    ▼                     │
 普通工具 / task       ┌──────────────┐            │
                       │ load_skill   │            │
                       └──────┬───────┘            │
                              ▼                    │
                       ┌──────────────┐            │
                       │ SKILL.md     │            │
                       │ full content │            │
                       └──────┬───────┘            │
                              ▼                    │
                         tool_result               │
                              │                    │
                              └────────────────────┘
```

### 8. System Prompt

System Prompt 根据当前运行环境动态构建，主要用于提供 Agent Identity、Tools、Workspace、Skills、Memory 等稳定信息。

模型每次请求实际接收的核心信息可以概括为：

`System Prompt + messages[] + tools`

```text
       ┌──────────────┐
       │   identity   │
       └──────┬───────┘
              │
       ┌──────────────┐
       │    tools     │
       └──────┬───────┘
              │
       ┌──────────────┐
       │  workspace   │
       └──────┬───────┘
              │
       ┌──────────────┐
       │skills/memory │
       └──────┬───────┘
              │
              ▼
       ┌───────────────────┐
       │ GetSystemPrompt() │
       │ Runtime Assembly  │
       └─────────┬─────────┘
                 │
                 │ system
                 ▼
              ┌───────────────────────────────────┐
              │                                   │
              ▼                                   │
       ┌─────────────┐                            │
       │ messages[]  │                            │
       └──────┬──────┘                            │
              │                                   │
              ├───────────────┐                   │
              │               │                   │
              ▼               ▼                   │
          messages          system                │
              │               │                   │
              └───────┬───────┘                   │
                      ▼                           │
               ┌─────────────┐                    │
               │     LLM     │                    │
               └──────┬──────┘                    │
                      ▼                           │
               ┌─────────────┐                    │
               │ tool_use ?  │                    │
               └──────┬──────┘                    │
                  是  │  否                       │
                      │   └─────────► 最终回答     │
                      ▼                           │
               ┌─────────────┐                    │
               │    Tool     │                    │
               └──────┬──────┘                    │
                      ▼                           │
                 tool_result                     │
                      │                           │
                      └───────────────────────────┘
```

### 9. Error Recovery

Agent 对不同错误采用不同的恢复方式，使可恢复错误尽量不会直接终止整个任务。

- 429 / 529：Retry / Backoff

- max_tokens：提高 Token 上限或继续生成

- prompt_too_long：触发 Reactive Compact

- Tool Error：转换成错误 tool_result，让模型继续处理

```text
              ┌─────────────────────────────────────────┐
              │                                         │
              ▼                                         │
       ┌─────────────┐                                  │
       │ messages[]  │                                  │
       └──────┬──────┘                                  │
              ▼                                         │
       ┌─────────────────┐                              │
       │   Call LLM      │                              │
       │  WithRetry()    │                              │
       └────────┬────────┘                              │
                │                                       │
         ┌──────┴───────┐                               │
         │              │                               │
      success          error                             │
         │              │                               │
         │              ▼                               │
         │       ┌─────────────────┐                    │
         │       │ classify error  │                    │
         │       └────────┬────────┘                    │
         │                │                             │
         │      ┌─────────┼──────────┐                  │
         │      │         │          │                  │
         │      ▼         ▼          ▼                  │
         │   429/529  max_tokens  prompt_too_long       │
         │      │         │          │                  │
         │      ▼         ▼          ▼                  │
         │    retry    escalate    ReactiveCompact      │
         │   backoff   /continue        │               │
         │      │         │             │               │
         │      └─────────┴──────┬──────┘               │
         │                       │                      │
         │                       └──────► Call LLM      │
         ▼                                              │
  ┌─────────────┐                                       │
  │ tool_use ?  │                                       │
  └──────┬──────┘                                       │
     是  │  否                                          │
         │   └──────────────► 最终回答                  │
         ▼                                              │
     Execute Tool                                       │
         │                                              │
         ▼                                              │
    tool_result                                         │
         │                                              │
         └──────────────────────────────────────────────┘
```

---

## 三、记忆管理

### 10. Context Compact

随着对话、Tool Call 和 Tool Result 不断增加，messages[] 会持续增长，因此在调用 LLM 前加入分层 Context Compact。

当前主要包括：

- ToolResultBudget：限制大型 Tool Result 对 Context 的占用

- SnipCompact：裁剪中间历史

- MicroCompact：压缩较旧的 Tool Result

- AutoCompact：超过阈值后使用 LLM Summary

- ReactiveCompact：API 返回 prompt_too_long 后进行紧急压缩

压缩完成后仍然回到原有 Agent Loop。

```text
              ┌────────────────────────────────────────────┐
              │                                            │
              ▼                                            │
       ┌─────────────┐                                     │
       │ messages[]  │                                     │
       └──────┬──────┘                                     │
              ▼                                            │
       ┌──────────────────┐                                │
       │ToolResultBudget  │                                │
       └──────┬───────────┘                                │
              ▼                                            │
       ┌──────────────────┐                                │
       │ SnipCompact      │                                │
       └──────┬───────────┘                                │
              ▼                                            │
       ┌──────────────────┐                                │
       │ MicroCompact     │                                │
       └──────┬───────────┘                                │
              ▼                                            │
       ┌──────────────────┐                                │
       │ size > limit ?   │                                │
       └──────┬───────────┘                                │
          是  │  否                                       │
              │   └────────────┐                           │
              ▼                │                           │
       ┌──────────────────┐    │                           │
       │ AutoCompact      │    │                           │
       │ LLM Summary      │    │                           │
       └──────┬───────────┘    │                           │
              └───────────┬────┘                           │
                          ▼                                │
                   ┌─────────────┐                         │
                   │     LLM     │                         │
                   └──────┬──────┘                         │
                          ▼                                │
                   ┌─────────────┐                         │
                   │ tool_use ?  │                         │
                   └──────┬──────┘                         │
                      是  │  否                            │
                          │   └──────────► 最终回答        │
                          ▼                                │
                      Execute Tool                         │
                          │                                │
                          ▼                                │
                     tool_result                           │
                          │                                │
                          └────────────────────────────────┘
```

### 11. Memory

Memory 用于保存适合长期保留的信息，例如用户偏好、项目背景和历史反馈。

Memory 保存到：

```text
.memory/
├── MEMORY.md
└── *.md
```

#### Memory 召回时机

新的 Agent Turn 开始后，根据当前 Session 的对话和本轮 Query
调用 `LoadMemories()`，从长期 Memory 中选择与当前任务相关的信息。

#### Memory 放置方式

选中的 Memory 不会永久追加到 `Session.Messages`，
而是在调用 LLM 前临时注入当前请求。

本轮任务完成后，通过 `ExtractMemories()` 提取值得长期保存的信息并写回 `.memory/`。

因此：

- `Session.Messages`：当前 Session 的连续对话上下文
- `Memory`：可以跨 Session 使用的长期信息

```text
                  ┌───────────────────────┐
                  │       .memory/        │
                  │ MEMORY.md + *.md      │
                  └───────┬─────────▲─────┘
                          │         │
                     Load │         │ Extract
                          │         │
                          ▼         │
              ┌──────────────────────────────────────┐
              │                                      │
              ▼                                      │
       ┌─────────────┐                               │
       │ messages[]  │                               │
       └──────┬──────┘                               │
              ▼                                      │
       ┌─────────────────┐                           │
       │ Context Compact │                           │
       └──────┬──────────┘                           │
              ▼                                      │
       ┌─────────────────┐                           │
       │ Load Memories   │                           │
       │ select relevant │                           │
       └──────┬──────────┘                           │
              ▼                                      │
       ┌─────────────────┐                           │
       │ Inject Memories │                           │
       │current user turn│                           │
       └──────┬──────────┘                           │
              ▼                                      │
       ┌─────────────┐                               │
       │     LLM     │                               │
       └──────┬──────┘                               │
              ▼                                      │
       ┌─────────────┐                               │
       │ tool_use ?  │                               │
       └──────┬──────┘                               │
          是  │  否                                  │
              │   └──────► 最终回答 ──► Extract ─────┘
              ▼
        Execute Tool
              │
              ▼
         tool_result
              │
              └──────────────────────────────────────┘
```

---

---

## 四、会话

### 12. Session

Agent 使用 `SessionManager` 管理多个逻辑 Session。

每个 Session 独立保存自己的 `Messages` 和运行状态，
因此用户可以在多个会话之间切换，并继续各自之前的对话。

支持：

```text
/session new <name>
/session switch <name>
/session list
```

例如：

```text
/session new weather
/session new weekly
/session switch weather
```

不同 Session 的 `Messages` 相互独立；
长期 Memory 则可以根据相关性在不同 Session 中被召回。

```text
                    ┌─────────────────────┐
                    │   SessionManager    │
                    │                     │
                    │ Current = "weather" │
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
              ▼                ▼                ▼
       ┌────────────┐    ┌────────────┐   ┌────────────┐
       │ default    │    │ weather    │   │ weekly     │
       │ Messages A │    │ Messages B │   │ Messages C │
       └────────────┘    └─────┬──────┘   └────────────┘
                               │
                         Current Session
                               │
                               ▼
              ┌──────────────────────────────────────┐
              │                                      │
              ▼                                      │
       ┌─────────────────┐                           │
       │ messages[]      │                           │
       │= weather.Messages                           │
       └──────┬──────────┘                           │
              ▼                                      │
       ┌─────────────────┐                           │
       │ Memory + Compact│                           │
       └──────┬──────────┘                           │
              ▼                                      │
       ┌─────────────┐                               │
       │     LLM     │                               │
       └──────┬──────┘                               │
              ▼                                      │
       ┌─────────────┐                               │
       │ tool_use ?  │                               │
       └──────┬──────┘                               │
          是  │  否                                  │
              │   └──────────────► 最终回答          │
              ▼                                      │
        Execute Tool                                 │
              │                                      │
              ▼                                      │
         tool_result                                 │
              │                                      │
              └──────────────────────────────────────┘
```

Agent Loop 结束时，将本轮更新后的 `messages` 写回当前 Session。

<!-- 六、测试

主要测试以下场景：

普通问题直接回答

Calculator Tool Calling

Web Search

Todo Write

连续多 Tool Loop

纯对话追问

基于 Tool Result 的追问

Multi-Session 隔离与切换

Context Compact

Tool Error Recovery

MAX_AGENT_ROUNDS

Memory Recall

七、AI Prompt 与问题解决记录

开发过程中使用 AI 辅助理解、设计和调试 Agent Runtime，主要涉及：

Agent Loop 与 Tool Use / Tool Result 协议

Tool Registry 与 Handler 设计

Permission 与 Hooks

Todo 与 Subagent

Skills 与 System Prompt

Error Recovery

Context Compact

Memory 的召回、注入与提取

Multi-Session

Client Tool 与 Server Tool

MAX_AGENT_ROUNDS

Per-Session Log

建议将较长的 Prompt 和问题解决过程单独记录到：

docs/ai-development.md

README 只保留总体设计、运行方式和关键实现说明。 -->

