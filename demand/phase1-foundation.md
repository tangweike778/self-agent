# Phase 1 - P0 基础地基

> 本阶段目标：让 Agent "能用起来"，填补当前已有骨架但缺失实现的关键能力。
> 本阶段是后续所有进化的前置依赖，必须优先完成。

---

## D001 上下文压缩器（Context Compressor）

### 需求背景

当前 `agent/compressor.go` 是空实现（直接 return 原文），`agent/agent.go` 的 `Ask()` 方法中检测到 token 超限后 **什么都没做**（空 if 块）。这意味着 Agent 在多轮 tool 调用后，上下文会无限膨胀直到超过模型限制，导致 API 调用失败。

### 现状代码定位

| 文件 | 位置 | 问题 |
|------|------|------|
| `agent/compressor.go` | 全文 | `TextCompress` 空实现，直接返回原文 |
| `agent/agent.go` | `Ask()` 中 `if computeTokens(messages) >= getSystemMaxToken()` | 空 if 块，检测到超限但无任何处理 |
| `agent/agent.go` | `SingleAsk()` | 已有单轮摘要能力，可复用于压缩 |

### 需求描述

实现真正的上下文压缩机制，当对话 token 数逼近模型上限时，自动对历史消息进行摘要压缩，保证 Agent 能持续运行而不会因 token 超限崩溃。

### 压缩策略

```
┌─────────────────────────────────────────────┐
│  messages 列表                               │
│                                             │
│  [0] system prompt        ──→ 永不压缩       │
│  [1] 早期 user 消息       ──→ 压缩为摘要     │
│  [2] 早期 assistant 消息  ──→ 压缩为摘要     │
│  [3] 早期 tool 调用结果   ──→ 压缩为摘要     │
│  ...                                        │
│  [N-5] 近期消息           ──→ 保留原文       │
│  [N-4] 近期消息           ──→ 保留原文       │
│  [N-3] 近期消息           ──→ 保留原文       │
│  [N-2] 近期消息           ──→ 保留原文       │
│  [N-1] 近期消息           ──→ 保留原文       │
│  [N]   最新消息           ──→ 保留原文       │
└─────────────────────────────────────────────┘
```

- **system prompt**（messages[0]）：永远保留原文，不参与压缩
- **最近 N 轮对话**（建议 N=3 轮，即最近 6 条 user/assistant 消息）：保留原文
- **更早的历史消息**：调用 `SingleAsk()` 进行摘要，合并为一条 `user` 类型的摘要消息
- 压缩后的总 token 数应控制在 `maxTokens * 0.7` 以内（预留 30% 给模型回复）

### 技术方案

1. 在 `Compressor` 中新增方法 `CompressMessages(messages []model.AgentMessage, maxTokens int64) []model.AgentMessage`
2. 该方法接收完整 messages 列表和 token 上限，返回压缩后的 messages
3. 内部使用 `SingleAsk()` 对早期消息做摘要
4. 在 `agent.go` 的空 if 块中调用此方法

### 验收标准

- [x] `Compressor.CompressMessages()` 方法实现完成
- [x] system prompt 在压缩过程中永远不被修改
- [x] 最近 3 轮对话保留原文
- [x] 更早的对话被摘要为一条精简消息
- [x] 压缩后总 token 数 ≤ maxTokens * 0.7
- [x] `agent.go` 中 token 超限时自动触发压缩
- [x] 压缩过程有日志输出（压缩前后 token 数）

### 状态：🟢 已完成（2026-03-26）

### 完成备注

- 测试报告：`test/D001-context-compressor-test-report.md`
- 测试结果：20/20 全部通过
- 额外改进：`getSystemPromptAndLatestDialog` 增加 error 返回值、`SingleAsk` 增加 3 次重试机制

---

## D002 对话历史持久化（Conversation History）

### 需求背景

当前 `Agent.Ask()` 每次调用都从零构建 messages（仅包含 system prompt + 当前 user 问题），历史对话完全丢失。Session 虽然有 `TaskQueue`，但只是一个任务通道，没有存储对话历史。这意味着用户每次提问都是"全新对话"，Agent 无法记住之前说过什么。

### 现状代码定位

| 文件 | 位置 | 问题 |
|------|------|------|
| `agent/agent.go` | `Ask()` 方法开头 | 每次都新建 messages，无历史 |
| `session/session.go` | `Session` 结构体 | 无 ConversationHistory 字段 |
| `session/session.go` | `Start()` 循环 | 只取 task → 问 Agent → 发渠道，无历史维护 |

### 需求描述

在 Session 层面维护完整的对话历史，每次调用 Agent 时将历史对话带入上下文，让 Agent 具备跨轮次的对话记忆能力。

### 技术方案

1. 在 `Session` 结构体中新增 `History []model.AgentMessage` 字段
2. 修改 `Agent.Ask()` 签名，接收历史 messages 参数：`Ask(question string, history []model.AgentMessage) (string, []model.AgentMessage, error)`
3. 返回值中新增更新后的 messages 列表，由 Session 保存
4. Session 的 `Start()` 循环中维护历史：
   - 取出 task → 连同 history 调用 Agent.Ask()
   - 保存返回的 messages 到 history
   - 下一轮继续使用
5. 结合 D001 的压缩器，当历史过长时自动压缩

### 数据流

```
Session.Start() 循环:
  ┌──────────────────────────────────────────┐
  │  task = TaskQueue.GetTask()              │
  │  response, newHistory = Agent.Ask(       │
  │      task.Content,                       │
  │      session.History                     │
  │  )                                       │
  │  session.History = newHistory             │
  │  Channel.SendMessage(response)           │
  └──────────────────────────────────────────┘
```

### 验收标准

- [x] Session 结构体包含 History 字段
- [x] Agent.Ask() 接收并使用历史 messages
- [x] 连续对话时，Agent 能记住之前的上下文
- [x] 历史过长时自动触发压缩（依赖 D001）
- [x] Session 重启后历史清空（后续 D007 再做持久化存储）

### 状态：🟢 已完成（2026-03-26）

### 依赖：D001（压缩器，否则历史无限增长会崩溃）✅ 已满足

### 完成备注

- 测试报告：`test/D002-conversation-history-test-report.md`
- 测试结果：28/28 全部通过（含 8 个回归测试）
- 额外功能：实现了 `/clear` 命令清空上下文、`rollbackUserMsg` Ask 错误回滚机制、`getLastMsg` 空切片保护
- 设计优化：Ask() 签名简化为 `Ask(messages) (messages, error)`，由 Session 层负责追加 user 消息和提取回复，职责分离更清晰

---

## D003 丰富工具集（Tool/Skill Expansion）

### 需求背景

当前 Agent 只有 `exec_shell` 一个工具，虽然理论上可以通过 shell 做很多事，但存在以下问题：
- 不够安全：LLM 直接执行任意 shell 命令风险高
- 不够精确：用 `cat` 读文件不如专用 `read_file` 工具（可控制行数、编码等）
- 不够丰富：缺少网络搜索、文件搜索等高阶能力
- 硬编码注册：工具注册在 `registerTools()` 和 `executeTool()` 中都是 switch-case 硬编码

### 现状代码定位

| 文件 | 位置 | 问题 |
|------|------|------|
| `agent/agent.go` | `registerTools()` | 硬编码只注册了 exec_shell |
| `agent/agent.go` | `executeTool()` | switch-case 硬编码分发 |
| `skill/exec.go` | 全文 | 唯一的 skill 实现 |

### 需求描述

#### 3.1 工具注册机制重构

将当前硬编码的工具注册改为 **接口化 + 自动注册** 机制：

```go
// skill/interface.go
type Skill interface {
    // Name 工具名称（唯一标识）
    Name() string
    // Description 返回 LLM 可理解的工具定义
    Description() model.ToolDefinition
    // Execute 执行工具，接收 JSON 参数字符串，返回结果字符串
    Execute(argsJSON string) string
}
```

- 所有 Skill 实现此接口
- `Agent` 通过 `[]Skill` 管理工具，无需 switch-case
- 支持动态注册/注销工具

#### 3.2 新增核心工具

| 工具名 | 功能 | 文件 | 优先级 |
|--------|------|------|--------|
| `read_file` | 读取指定文件内容，支持行范围 | `skill/read_file.go` | 高 |
| `write_file` | 创建/写入文件 | `skill/write_file.go` | 高 |
| `search_file` | 在目录中搜索文件名（模糊匹配） | `skill/search_file.go` | 中 |
| `grep_search` | 在文件中搜索内容（正则匹配） | `skill/grep_search.go` | 中 |
| `list_dir` | 列出目录内容 | `skill/list_dir.go` | 中 |

#### 3.3 工具参数规范

每个工具的 Parameters 定义应包含：
- `type`: "object"
- `properties`: 每个参数的类型、描述
- `required`: 必填参数列表

### 验收标准

- [ ] `Skill` 接口定义完成
- [ ] `exec_shell` 重构为实现 `Skill` 接口
- [ ] `Agent` 使用接口化方式管理和调度工具，消除 switch-case
- [ ] 至少新增 `read_file`、`write_file` 两个核心工具
- [ ] 工具执行结果格式统一，便于 LLM 理解
- [ ] 新增工具只需实现接口 + 注册，无需修改 Agent 代码

### 状态：🔴 未开始

---

## 📋 Phase 1 整体检查清单

- [x] D001 上下文压缩器实现并集成
- [x] D002 对话历史在 Session 中维护
- [ ] D003 工具接口化 + 至少 2 个新工具
- [ ] 所有变更有对应的日志输出
- [ ] 主流程 main.go → Gateway → Session → Agent 链路正常运行
- [ ] 飞书渠道收发消息正常

---

## 📅 更新日志

| 日期 | 更新内容 |
|------|---------|
| 2026-03-25 | 创建 Phase 1 需求文档，定义 D001/D002/D003 三个需求 |
| 2026-03-26 | D001 上下文压缩器验收通过，全部 7 项验收标准达标，20/20 测试通过 |
| 2026-03-26 | D002 对话历史持久化验收通过，全部 5 项验收标准达标，28/28 测试通过 |
