# 🚀 第二阶段：P1 核心进化 — 让 Agent "变聪明"

> 本阶段目标：让 Agent 具备推理、规划和实时反馈能力，从"工具调用机器"进化为"自主思考的智能体"。
> 前置依赖：P0 阶段全部完成

---

## 需求 4：实现 ReAct 推理框架（Reasoning + Acting）

### 优先级：P1 | 状态：✅ 已完成

### 背景

当前 Agent 的 tool_calls 循环是"盲目"的 —— LLM 直接决定调用哪个工具，没有显式的推理过程。OpenClaw 级别的智能体核心差异在于 **Thought → Action → Observation → Reflection** 的推理链路，让 Agent 的每一步决策都有理可循。

### 需求描述

1. **Thought（思考）**：在每轮 tool 调用前，要求 LLM 先输出推理过程
   - 通过 system prompt 引导 LLM 使用 `<thought>...</thought>` 标签包裹思考内容
   - 解析 LLM 输出，分离思考内容和工具调用

2. **Action（行动）**：基于思考结果，执行工具调用
   - 保持现有 function calling 机制
   - 新增：LLM 可以输出 `<action_plan>` 标签，说明本轮要做什么以及为什么

3. **Observation（观察）**：工具返回结果后，将结果作为观察信息注入上下文
   - 对工具结果进行格式化，使用 `<observation>` 标签标记
   - 如果工具执行失败，在 observation 中标注失败原因

4. **Reflection（反思）**：每轮结束后触发反思
   - LLM 评估当前进展："我已经知道了什么？还缺什么信息？下一步应该做什么？"
   - 支持 LLM 主动决定"信息已经足够，可以给出最终答案"

### 技术方案

```
新增文件:
  agent/react.go       — ReAct 推理引擎，管理 Thought-Action-Observation 循环
  agent/parser.go      — 解析 LLM 输出中的思考/行动/反思标签
  
修改文件:
  agent/agent.go       — Ask() 方法重构为 ReAct 循环
  prompt/system_prompt.md — 加入 ReAct 格式指令
```

### 核心数据结构

```go
// ReActStep 单轮推理步骤
type ReActStep struct {
    StepNum     int       // 步骤编号
    Thought     string    // 思考内容
    Action      string    // 执行的动作描述
    ToolCalls   []ToolCall // 工具调用列表
    Observation string    // 观察结果
    Reflection  string    // 反思内容
}

// ReActTrace 完整推理链路
type ReActTrace struct {
    Question string       // 原始问题
    Steps    []ReActStep  // 推理步骤链
    Answer   string       // 最终答案
}
```

### 验收标准

- [x] Agent 每轮决策前输出可读的思考过程
- [x] 思考内容可在日志中追踪，支持后续调试
- [x] 工具调用失败时，Agent 能基于反思进行重试或换一种方式
- [x] 推理链路可完整导出，用于分析和优化

---

## 需求 5：任务规划与分解（Task Planning / Decomposition）

### 优先级：P1 | 状态：🔲 未开始

### 背景

当前 `TaskQueue` 只是一个简单的 channel，Agent 一次处理一个 task，无法处理复杂的多步骤任务。OpenClaw 级别的智能体需要能够将复杂任务分解为可执行的子任务，并按依赖关系有序执行。

### 需求描述

1. **任务分解（Decomposition）**
   - Agent 收到复杂任务时，先调用 LLM 制定执行计划
   - 将大任务拆分为多个子任务（SubTask），每个子任务有明确的目标
   - 子任务之间支持依赖关系定义

2. **执行计划（Plan）**
   - 生成有向无关图（DAG）结构的执行计划
   - 无依赖的子任务可以并行执行
   - 有依赖的子任务按拓扑序执行

3. **动态调整（Re-planning）**
   - 执行过程中如果某个子任务失败或发现新信息，Agent 可以动态调整计划
   - 支持新增、删除、修改子任务
   - 每次调整需要记录变更原因

4. **进度追踪**
   - 每个子任务有明确的状态：pending → running → completed / failed
   - 整体任务有完成度百分比
   - 支持向用户报告当前执行进度

### 技术方案

```
新增文件:
  model/plan.go        — Plan、SubTask、DAG 数据结构定义
  agent/planner.go     — 任务规划器，负责调用 LLM 生成/调整计划
  agent/executor.go    — 计划执行器，按 DAG 拓扑序执行子任务

修改文件:
  session/session.go   — Start() 方法重构，支持任务规划流程
  model/TaskQueue.go   — TaskQueue 增加优先级和依赖支持
```

### 核心数据结构

```go
// SubTask 子任务
type SubTask struct {
    ID           string   `json:"id"`
    Description  string   `json:"description"`
    Dependencies []string `json:"dependencies"` // 依赖的子任务ID列表
    Status       string   `json:"status"`       // pending/running/completed/failed
    Result       string   `json:"result"`
}

// Plan 执行计划
type Plan struct {
    TaskID    string    `json:"task_id"`
    Goal      string    `json:"goal"`       // 总目标
    SubTasks  []SubTask `json:"sub_tasks"`
    Version   int       `json:"version"`    // 计划版本（每次re-plan递增）
    CreatedAt time.Time `json:"created_at"`
}
```

### 验收标准

- [ ] 复杂任务能被自动分解为 2-5 个子任务
- [ ] 子任务按依赖关系正确排序执行
- [ ] 子任务失败时能触发 re-planning
- [ ] 可以向用户展示当前执行进度
- [ ] 简单任务不触发规划流程，直接执行（避免过度规划）

---

## 需求 6：流式响应（Streaming Response）

### 优先级：P1 | 状态：🔲 未开始

### 背景

当前 `callAPI` 是同步阻塞式调用，用户发送问题后需要等待 LLM 全部生成完毕才能看到结果。对于复杂的多轮推理任务，等待时间可能长达数十秒甚至数分钟，用户体验极差。OpenClaw 级别的智能体需要实时反馈思考过程和执行状态。

### 需求描述

1. **LLM 流式调用**
   - 支持 Deepseek API 的 SSE（Server-Sent Events）流式输出
   - 逐 token 接收 LLM 的响应，实时处理

2. **Gateway 流式推送**
   - HTTP 接口支持 SSE 协议，将 Agent 的思考和执行过程实时推送给客户端
   - 推送事件类型包括：
     - `thinking` — Agent 正在思考
     - `tool_calling` — Agent 正在调用工具
     - `tool_result` — 工具执行结果
     - `answering` — Agent 正在生成最终答案
     - `done` — 完成

3. **Channel 渠道适配**
   - 飞书渠道：先发送"🤔 正在思考..."占位消息，后续更新为最终结果
   - 如果最终结果过长，拆分为多条消息发送
   - 支持发送执行进度卡片（如"已完成 3/5 步骤"）

4. **超时与中断**
   - 流式请求支持客户端主动中断（取消请求）
   - 设置总超时时间，超时后优雅终止

### 技术方案

```
新增文件:
  agent/stream.go      — 流式调用 Deepseek API 的实现
  gateway/sse.go       — SSE 推送服务端实现
  model/event.go       — 流式事件定义

修改文件:
  agent/agent.go       — callAPI 增加流式调用选项
  gateway/gateway.go   — 新增 SSE 端点 GET /ai/stream
  channel/channel.go   — sendToFeishu 支持消息更新
```

### 核心数据结构

```go
// StreamEvent 流式事件
type StreamEvent struct {
    Type    string      `json:"type"`    // thinking/tool_calling/tool_result/answering/done
    Content string      `json:"content"` // 事件内容
    Step    int         `json:"step"`    // 当前步骤
    Total   int         `json:"total"`   // 总步骤数（如果已知）
}
```

### 验收标准

- [ ] LLM 响应支持流式接收，首 token 延迟 < 500ms
- [ ] HTTP SSE 端点能实时推送 Agent 执行过程
- [ ] 飞书渠道能先发送占位消息再更新为最终结果
- [ ] 客户端断开连接后，后端能正确释放资源
- [ ] 总超时机制生效，不会出现无限等待

---

## 阶段验收总结

完成本阶段后，Agent 应具备以下新能力：

| 能力 | 对标 OpenClaw | 描述 |
|------|-------------|------|
| 自主推理 | ✅ ReAct 框架 | 每步决策有思考→行动→观察→反思 |
| 任务规划 | ✅ Task Planning | 复杂任务自动分解为子任务 DAG |
| 实时反馈 | ✅ Streaming | 思考过程和执行状态实时推送 |

**预估工期**：10-15 天
**前置依赖**：P0 阶段全部完成（上下文压缩、对话历史、工具集）
