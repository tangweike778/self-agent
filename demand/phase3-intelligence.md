# P2 - 高级智能阶段需求

> 本阶段目标：让 Agent "有记忆有个性"，从工具执行者进化为有长期认知、能协作、能自我进化的智能体。
> 前置依赖：P0（基础地基）+ P1（核心进化）全部完成

---

## 需求7：长期记忆系统（Long-term Memory）

### 需求背景

当前 Agent 的记忆仅限于单次对话的上下文窗口（短期记忆）。即使 P0 阶段实现了对话历史持久化，也只是"会话级记忆"。Agent 无法跨会话记住用户的偏好、历史交互中的关键信息、曾经解决过的问题等。OpenClaw 级别的智能体必须具备跨会话的长期记忆能力。

### 需求描述

实现基于向量数据库的长期记忆系统，让 Agent 具备跨会话的持久化语义记忆能力。

### 功能要求

#### 7.1 记忆存储引擎
- 引入向量数据库（推荐 Milvus/Qdrant/Chroma，或轻量级的本地 SQLite + 向量扩展）
- 每条记忆包含：原始文本、Embedding 向量、时间戳、来源会话ID、重要性评分、标签
- 定义 `memory/` 模块，包含 `Memory` 结构体和存储接口

```go
// memory/memory.go
type Memory struct {
    ID          string    // 唯一标识
    Content     string    // 记忆内容
    Embedding   []float64 // 向量表示
    Importance  float64   // 重要性评分 0-1
    SessionID   string    // 来源会话
    Tags        []string  // 标签分类
    CreatedAt   time.Time // 创建时间
    AccessedAt  time.Time // 最近访问时间
    AccessCount int       // 访问次数
}

type MemoryStore interface {
    Save(memory *Memory) error
    Search(query string, topK int) ([]*Memory, error)
    Delete(id string) error
    Decay() error // 记忆衰减
}
```

#### 7.2 记忆提取（Memory Extraction）
- 每次对话结束后，Agent 自动调用 LLM 提取对话中的关键信息
- 提取维度：用户偏好、重要事实、问题解决方案、用户纠正的错误
- 对提取的信息进行去重和合并（避免重复记忆）

#### 7.3 记忆召回（Memory Retrieval）
- 每次新对话开始时，根据用户输入进行语义检索，召回 Top-K 条相关记忆
- 将召回的记忆注入到 system prompt 或作为独立的 context 消息
- 召回策略：语义相似度 × 重要性评分 × 时间衰减因子

#### 7.4 记忆衰减与更新
- 长期未被访问的记忆自动降低重要性评分
- 衰减公式：`importance = importance × decay_rate ^ (days_since_last_access)`
- 重要性低于阈值的记忆标记为归档，不再参与检索
- 当新记忆与旧记忆冲突时，更新旧记忆内容并重置时间戳

#### 7.5 记忆管理 API
- 用户可查看 Agent 的记忆列表
- 用户可手动删除、修改特定记忆
- 用户可主动告诉 Agent "请记住：xxx"

### 技术方案

```
对话结束 → LLM提取关键信息 → Embedding → 存入向量数据库
新对话开始 → 用户输入Embedding → 向量检索Top-K → 注入上下文
定时任务 → 记忆衰减 → 归档/清理
```

### 验收标准

- [ ] Agent 能跨会话记住用户告知的信息（如"我叫小明"，下次对话能主动提及）
- [ ] 记忆检索延迟 < 200ms
- [ ] 支持至少 10000 条记忆的存储和检索
- [ ] 记忆衰减机制正常工作，过期记忆不会被召回
- [ ] 用户可通过 API 管理 Agent 的记忆

### 当前状态

- **状态**：⏳ 未开始
- **进度**：0%

---

## 需求8：多 Agent 协作（Multi-Agent Orchestration）

### 需求背景

单个 Agent 在处理复杂任务时能力有限。OpenClaw 架构的核心优势之一是支持多个不同角色的 Agent 协同工作，类似于一个"AI团队"。每个 Agent 有自己的专长（如规划、编码、测试、审查），通过消息传递和任务委托完成复杂工作。

### 需求描述

实现多 Agent 编排系统，支持创建不同角色的 Agent，Agent 之间可以通信和委托任务。

### 功能要求

#### 8.1 Agent 角色定义
- 支持通过配置文件或代码定义 Agent 角色
- 每个角色有独立的 system prompt、工具集、行为策略

```go
// agent/role.go
type AgentRole struct {
    Name        string             // 角色名称
    Description string             // 角色描述
    SystemPrompt string            // 专属系统提示词
    Tools       []model.ToolDefinition // 可用工具集
    Model       string             // 使用的模型（可不同于其他Agent）
    MaxTokens   int                // 最大token限制
}
```

预定义角色：
- **Planner**（规划者）：负责任务分析和计划制定，不直接执行
- **Executor**（执行者）：负责具体任务的执行，拥有工具调用权限
- **Reviewer**（审查者）：负责对执行结果进行质量审查和反馈
- **Researcher**（研究者）：负责信息收集和知识检索

#### 8.2 Agent 间通信机制
- 定义统一的 Agent 间消息协议

```go
// agent/message.go
type AgentToAgentMessage struct {
    FromAgent   string      // 发送方Agent ID
    ToAgent     string      // 接收方Agent ID
    Type        MessageType // 消息类型：request/response/delegate/feedback
    TaskID      string      // 关联的任务ID
    Content     string      // 消息内容
    Metadata    map[string]interface{} // 扩展元数据
    Timestamp   time.Time
}

type MessageType string
const (
    MsgTypeRequest  MessageType = "request"   // 请求
    MsgTypeResponse MessageType = "response"  // 响应
    MsgTypeDelegate MessageType = "delegate"  // 任务委托
    MsgTypeFeedback MessageType = "feedback"  // 反馈
)
```

#### 8.3 编排器（Orchestrator）
- 实现一个 Orchestrator 模块，负责 Agent 的创建、调度和通信路由

```go
// orchestrator/orchestrator.go
type Orchestrator struct {
    Agents     map[string]*agent.Agent  // 所有Agent实例
    MessageBus chan AgentToAgentMessage  // 消息总线
    TaskGraph  *TaskDAG                 // 任务依赖图
}

func (o *Orchestrator) Dispatch(task *model.Task) error     // 分发任务
func (o *Orchestrator) Route(msg AgentToAgentMessage) error  // 路由消息
func (o *Orchestrator) Monitor() *OrchestratorStatus         // 监控状态
```

#### 8.4 任务委托（Delegation）
- Agent A 在执行过程中可以将子任务委托给 Agent B
- 委托包含：任务描述、期望输出格式、超时时间
- 被委托的 Agent 执行完成后，结果自动回传给委托方

#### 8.5 协作模式
支持多种预定义的协作模式：
- **流水线模式**（Pipeline）：A → B → C，顺序执行
- **分散-聚合模式**（Scatter-Gather）：A 将任务分发给 B/C/D，汇总结果
- **辩论模式**（Debate）：多个 Agent 对同一问题给出不同观点，由仲裁者决策
- **监督模式**（Supervisor）：一个 Agent 监督其他 Agent 的执行，提供反馈和纠正

### 验收标准

- [ ] 可配置创建至少 4 种不同角色的 Agent
- [ ] Agent 之间可通过消息总线通信，消息延迟 < 50ms
- [ ] 支持至少 2 种协作模式（流水线 + 分散聚合）
- [ ] 任务委托机制正常工作，委托结果可正确回传
- [ ] 提供编排器状态监控接口

### 当前状态

- **状态**：⏳ 未开始
- **进度**：0%

---

## 需求9：自我进化能力（Self-Evolution）

### 需求背景

传统 Agent 的能力在部署后就固化了，无法根据使用过程中的反馈进行自我优化。OpenClaw 理念中，智能体应具备"自我进化"的能力——能够优化自己的提示词、学习新技能、从错误中吸取教训。

### 需求描述

实现 Agent 的自我进化机制，使其能根据用户反馈和执行历史持续优化自身能力。

### 功能要求

#### 9.1 Prompt 自优化
- Agent 记录每次交互的用户满意度（显式反馈：👍👎，隐式反馈：是否追问/重试）
- 定期分析低满意度交互，调用 LLM 生成 system prompt 的改进建议
- 支持 prompt 版本管理，可回滚到历史版本

```go
// evolution/prompt_optimizer.go
type PromptVersion struct {
    Version     int       // 版本号
    Content     string    // 提示词内容
    Score       float64   // 综合评分
    SampleSize  int       // 评估样本数
    CreatedAt   time.Time
    IsActive    bool      // 是否当前生效
}

type PromptOptimizer struct {
    Versions    []PromptVersion
    FeedbackLog []UserFeedback
}

func (po *PromptOptimizer) Analyze() (*OptimizationSuggestion, error)
func (po *PromptOptimizer) Apply(version int) error
func (po *PromptOptimizer) Rollback(version int) error
```

#### 9.2 Skill 动态学习
- Agent 可以根据需要"学习"新的 Skill（工具）
- 学习方式1：用户提供 Skill 描述和实现代码，Agent 注册新工具
- 学习方式2：Agent 从执行 shell 命令的模式中自动抽象出常用操作，封装为 Skill
- 新 Skill 需要经过安全审查才能激活

```go
// evolution/skill_learner.go
type LearnedSkill struct {
    Name        string
    Description string
    Source      string    // "user_provided" | "auto_extracted"
    Code        string    // 实现代码/命令模板
    UsageCount  int       // 使用次数
    SuccessRate float64   // 成功率
    CreatedAt   time.Time
    IsApproved  bool      // 是否通过审查
}
```

#### 9.3 错误学习与避免
- 记录 Agent 执行过程中的所有错误（工具调用失败、用户纠正、超时等）
- 将错误模式抽象为"经验教训"（Lessons Learned）
- 在后续类似场景中，将相关经验教训注入上下文，避免重蹈覆辙

```go
// evolution/error_learner.go
type Lesson struct {
    ID          string
    Scenario    string    // 触发场景描述
    Mistake     string    // 错误行为描述
    Correction  string    // 正确做法描述
    Embedding   []float64 // 向量表示（用于场景匹配）
    Confidence  float64   // 置信度
    ApplyCount  int       // 被应用次数
}
```

#### 9.4 A/B 测试框架
- 支持对 prompt、Skill、策略等进行 A/B 测试
- 自动收集两个版本的效果数据（响应质量、用户满意度、执行时间等）
- 达到统计显著性后自动选择胜出方案

### 验收标准

- [ ] system prompt 支持版本管理，至少保留最近 10 个版本
- [ ] Agent 能根据用户反馈生成 prompt 优化建议
- [ ] 支持用户手动注册新 Skill，注册后立即可用
- [ ] 错误学习机制能记录失败场景，并在类似场景中发出提醒
- [ ] A/B 测试框架可同时运行 2 个版本的 prompt 并对比效果

### 当前状态

- **状态**：⏳ 未开始
- **进度**：0%

---

## 阶段完成标准

P2 阶段全部完成的标志：
1. ✅ Agent 具备跨会话长期记忆，能记住用户偏好和历史交互
2. ✅ 支持至少 4 种角色的 Agent，可协作完成复杂任务
3. ✅ Agent 能根据反馈自动优化 prompt 并学习新 Skill
4. ✅ 所有功能有完善的单元测试和集成测试

## 与 OpenClaw 的对标

| OpenClaw 能力 | 本阶段需求 | 对标程度 |
|---------------|-----------|---------|
| Persistent Memory | 需求7-长期记忆系统 | 🟢 完全对标 |
| Multi-Agent System | 需求8-多Agent协作 | 🟢 完全对标 |
| Self-Improvement | 需求9-自我进化 | 🟡 基础对标（OpenClaw 有更强的元学习能力） |
| Agent Personality | 需求8.1-角色定义 | 🟢 完全对标 |
