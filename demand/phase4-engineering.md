# P3 - 工程化阶段需求

> 本阶段目标：让项目"可生产使用"，从实验性原型进化为可靠、可观测、安全的生产级系统。
> 前置依赖：P0（基础地基）+ P1（核心进化）+ P2（高级智能）全部完成

---

## 需求10：可观测性与日志系统（Observability & Logging）

### 需求背景

当前项目仅使用 `log.Printf` 进行简单日志输出，没有结构化日志、链路追踪、指标监控。在生产环境中，无法快速定位问题、分析性能瓶颈、追踪 Token 消耗成本。一个生产级智能体系统必须具备完善的可观测性体系。

### 需求描述

建立完整的可观测性体系，包括结构化日志、分布式链路追踪、关键指标监控和 Token 成本控制。

### 功能要求

#### 10.1 结构化日志

- 引入结构化日志库（推荐 `zap` 或 `slog`），替换当前所有 `log.Printf`
- 日志字段标准化：时间戳、日志级别、模块名、TraceID、SessionID、消息内容
- 支持日志级别动态调整（无需重启服务）
- 支持日志输出到多目标（控制台、文件、远程收集器）

```go
// logger/logger.go
type Logger struct {
    core     *zap.Logger
    traceID  string
    sessionID string
    module   string
}

func NewLogger(module string) *Logger
func (l *Logger) WithTrace(traceID string) *Logger
func (l *Logger) WithSession(sessionID string) *Logger
func (l *Logger) Info(msg string, fields ...zap.Field)
func (l *Logger) Error(msg string, fields ...zap.Field)
func (l *Logger) Debug(msg string, fields ...zap.Field)
func (l *Logger) Warn(msg string, fields ...zap.Field)
```

#### 10.2 链路追踪（Distributed Tracing）

- 每个请求生成唯一 TraceID，贯穿整个处理链路
- 追踪链路：HTTP 请求 → Gateway → Session → Agent → LLM API 调用 → Tool 执行 → Channel 回复
- 每个环节记录 Span：开始时间、结束时间、耗时、状态、关键参数
- 支持导出为 OpenTelemetry 格式（方便对接 Jaeger/Zipkin）

```go
// tracer/tracer.go
type Span struct {
    TraceID   string        // 链路ID
    SpanID    string        // 节点ID
    ParentID  string        // 父节点ID
    Name      string        // 操作名称（如 "agent.ask", "tool.exec_shell"）
    StartTime time.Time
    EndTime   time.Time
    Duration  time.Duration
    Status    SpanStatus    // OK / ERROR / TIMEOUT
    Tags      map[string]string
}

type Tracer interface {
    StartSpan(name string) *Span
    EndSpan(span *Span)
    GetTraceID() string
}
```

#### 10.3 关键指标监控（Metrics）

- Token 消耗监控：每次 API 调用的 prompt_tokens、completion_tokens、total_tokens
- 成本监控：按模型计价，累计统计 API 调用费用
- 性能指标：请求延迟、Tool 调用耗时、LLM 响应时间
- 业务指标：每日会话数、每会话平均轮次、工具调用频率分布
- 提供 `/metrics` HTTP 端点，兼容 Prometheus 格式

```go
// metrics/metrics.go
type MetricsCollector struct {
    TokenUsage    *TokenMetrics    // Token 消耗
    CostTracker   *CostTracker     // 成本追踪
    LatencyHist   *LatencyMetrics  // 延迟直方图
    BusinessStats *BusinessMetrics // 业务指标
}

type TokenMetrics struct {
    TotalPromptTokens     int64   // 累计 prompt tokens
    TotalCompletionTokens int64   // 累计 completion tokens
    TotalCost             float64 // 累计费用（元）
}
```

#### 10.4 告警机制

- Token 消耗超过预设阈值时发出告警（通过绑定的 Channel 推送）
- 连续 N 次 API 调用失败时告警
- 单次请求耗时超过阈值告警
- 告警去重：相同类型告警在冷却期内不重复发送

### 验收标准

- [ ] 所有模块使用结构化日志替代 `log.Printf`，日志包含 TraceID 和 SessionID
- [ ] 每个请求有完整的链路追踪，可查看各环节耗时
- [ ] `/metrics` 端点正常输出 Prometheus 格式指标
- [ ] Token 消耗和费用可实时查看
- [ ] 告警机制正常工作，超阈值时通过 Channel 推送通知

### 当前状态

- **状态**：⏳ 未开始
- **进度**：0%

---

## 需求11：多模型支持（Multi-Model Support）

### 需求背景

当前 Agent 硬绑定 Deepseek 单一模型（`agent.go` 中 BaseURL 和 Model 写死为 `deepseek-chat`）。在实际使用中，不同任务适合不同模型：复杂推理用强模型（Claude/GPT-4o），简单执行用快模型（DeepSeek），摘要压缩用性价比模型。此外，单一模型厂商的 API 可能出现故障，需要 failover 机制。

### 需求描述

实现模型抽象层，支持多模型注册、按策略路由、自动故障转移。

### 功能要求

#### 11.1 模型抽象接口

- 定义统一的 LLM 调用接口，屏蔽不同厂商 API 差异
- 每个模型实现统一接口即可接入

```go
// llm/interface.go
type LLMProvider interface {
    // Chat 发起对话（支持 function calling）
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    // StreamChat 流式对话
    StreamChat(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error)
    // Name 模型名称标识
    Name() string
    // MaxTokens 模型最大上下文长度
    MaxTokens() int
    // CostPerToken 每 token 费用（用于成本计算）
    CostPerToken() (promptCost float64, completionCost float64)
}

type ChatRequest struct {
    Messages  []model.AgentMessage
    Tools     []model.ToolDefinition
    MaxTokens int
    Temperature float64
    TopP      float64
}

type ChatResponse struct {
    Message      model.AgentMessageWithToolCalls
    FinishReason string
    Usage        TokenUsage
}

type TokenUsage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}
```

#### 11.2 多模型注册与配置

- 通过配置文件注册多个模型，每个模型有独立的 API Key、BaseURL、参数配置
- 支持的模型厂商：Deepseek、OpenAI（GPT-4o/GPT-4o-mini）、Anthropic（Claude）、本地模型（Ollama）

```yaml
# config.yaml 新增配置
models:
  - name: "deepseek-chat"
    provider: "deepseek"
    api_key: "sk-xxx"
    base_url: "https://api.deepseek.com/v1"
    max_tokens: 120000
    cost_per_1k_prompt: 0.001
    cost_per_1k_completion: 0.002

  - name: "gpt-4o"
    provider: "openai"
    api_key: "sk-xxx"
    base_url: "https://api.openai.com/v1"
    max_tokens: 128000
    cost_per_1k_prompt: 0.005
    cost_per_1k_completion: 0.015

  - name: "claude-sonnet"
    provider: "anthropic"
    api_key: "sk-xxx"
    base_url: "https://api.anthropic.com/v1"
    max_tokens: 200000
    cost_per_1k_prompt: 0.003
    cost_per_1k_completion: 0.015

  - name: "local-qwen"
    provider: "ollama"
    base_url: "http://localhost:11434"
    max_tokens: 32000
```

#### 11.3 模型路由策略

- 支持按任务类型自动选择模型
- 预定义路由规则：

```go
// llm/router.go
type ModelRouter struct {
    Providers map[string]LLMProvider
    Rules     []RoutingRule
    Default   string // 默认模型
}

type RoutingRule struct {
    TaskType  string // "planning" | "execution" | "summarization" | "review"
    Model     string // 目标模型名称
    Priority  int    // 优先级
    Condition func(req *ChatRequest) bool // 自定义匹配条件
}
```

路由策略：
- **规划类任务**（planning）→ 使用强模型（GPT-4o / Claude）
- **简单执行**（execution）→ 使用快速模型（Deepseek / 本地模型）
- **摘要压缩**（summarization）→ 使用性价比模型（Deepseek / GPT-4o-mini）
- **代码审查**（review）→ 使用强模型（Claude）

#### 11.4 故障转移（Failover）

- 当主模型 API 调用失败时，自动切换到备用模型
- 失败重试策略：最多重试 2 次，每次间隔指数递增
- 模型健康检查：定期 ping，标记不可用的模型
- 故障恢复：模型恢复后自动切回

```go
// llm/failover.go
type FailoverManager struct {
    HealthStatus map[string]bool          // 模型健康状态
    RetryConfig  RetryConfig              // 重试配置
    FallbackChain map[string][]string     // 故障转移链：model -> [fallback1, fallback2]
}

type RetryConfig struct {
    MaxRetries    int           // 最大重试次数
    InitialDelay  time.Duration // 初始延迟
    MaxDelay      time.Duration // 最大延迟
    BackoffFactor float64       // 退避因子
}
```

### 验收标准

- [ ] 至少接入 2 种不同厂商的模型（Deepseek + OpenAI 或 Claude）
- [ ] 统一接口可正常调用不同模型，返回格式一致
- [ ] 模型路由策略生效，不同任务类型使用不同模型
- [ ] 主模型故障时自动切换到备用模型，切换延迟 < 3s
- [ ] 配置文件支持热加载新模型，无需重启服务

### 当前状态

- **状态**：⏳ 未开始
- **进度**：0%

---

## 需求12：安全与权限控制（Security & Access Control）

### 需求背景

当前 `exec_shell` 工具可以在服务器上执行任意 shell 命令，没有任何沙箱隔离和权限控制，这在生产环境中是极其危险的。此外，HTTP API 没有任何认证鉴权机制，任何人都可以调用。随着工具集的丰富（P0-D003），安全问题会愈发严峻。

### 需求描述

建立完整的安全体系，包括工具权限控制、命令沙箱隔离、API 认证鉴权和敏感信息保护。

### 功能要求

#### 12.1 工具权限控制

- 定义工具权限等级：`safe`（安全）、`moderate`（需确认）、`dangerous`（危险）
- 不同 Session/用户可配置不同的工具权限白名单
- 危险操作需要二次确认（通过 Channel 回调用户确认）

```go
// security/permission.go
type PermissionLevel string
const (
    PermSafe      PermissionLevel = "safe"      // 只读操作，如 ls/cat/grep
    PermModerate  PermissionLevel = "moderate"  // 修改操作，如 write_file/mkdir
    PermDangerous PermissionLevel = "dangerous" // 高危操作，如 rm/exec 任意命令
)

type ToolPermission struct {
    ToolName    string
    Permission  PermissionLevel
    AllowedArgs map[string][]string // 参数白名单（如限制命令前缀）
}

type PermissionManager struct {
    DefaultLevel   PermissionLevel            // 默认权限等级
    ToolPerms      map[string]*ToolPermission // 工具权限映射
    UserOverrides  map[string]PermissionLevel // 用户级权限覆盖
}

func (pm *PermissionManager) Check(userID string, toolName string, args map[string]interface{}) (bool, string)
```

#### 12.2 命令沙箱隔离

- exec_shell 工具增加命令黑名单机制
- 禁止执行的命令模式：`rm -rf /`、`:(){ :|:& };:`（fork bomb）、`dd if=/dev/zero`、`mkfs` 等
- 限制工作目录：只允许在指定目录下执行命令
- 限制资源：CPU 时间上限、内存使用上限、磁盘写入限制
- 可选：使用 Docker/namespace 做进程级隔离

```go
// security/sandbox.go
type Sandbox struct {
    AllowedDirs    []string          // 允许的工作目录
    BlockedCmds    []string          // 黑名单命令关键词
    BlockedPatterns []*regexp.Regexp // 黑名单命令正则
    MaxCPUTime     time.Duration     // CPU 时间限制
    MaxMemoryMB    int               // 内存限制 (MB)
    MaxOutputBytes int               // 输出大小限制
}

func (s *Sandbox) Validate(command string) (bool, string) // 校验命令是否安全
func (s *Sandbox) Execute(command string, timeout int) *ExecShellResult // 在沙箱中执行
```

#### 12.3 API 认证鉴权

- HTTP API 增加认证机制，支持以下方式：
  - API Key 认证（Header: `Authorization: Bearer <api-key>`）
  - Session Token 认证（登录后获取 token）
- 权限分级：管理员（所有权限）、普通用户（受限权限）、只读用户
- 请求频率限制（Rate Limiting）：防止 API 滥用

```go
// security/auth.go
type AuthManager struct {
    APIKeys    map[string]*APIKeyInfo  // API Key → 用户信息
    RateLimiter *RateLimiter           // 请求频率限制器
}

type APIKeyInfo struct {
    Key       string
    UserID    string
    Role      UserRole   // admin / user / readonly
    RateLimit int        // 每分钟请求上限
    CreatedAt time.Time
    ExpiresAt time.Time
}

type UserRole string
const (
    RoleAdmin    UserRole = "admin"
    RoleUser     UserRole = "user"
    RoleReadonly UserRole = "readonly"
)
```

#### 12.4 敏感信息保护

- API Key 等敏感配置支持环境变量注入，不在配置文件中明文存储
- 日志输出自动脱敏：API Key、用户 token、密码等字段自动替换为 `***`
- LLM 对话中的敏感信息检测和过滤（如用户不小心发送了密码）
- 审计日志：记录所有工具调用、权限变更、认证事件

```go
// security/sanitizer.go
type Sanitizer struct {
    Patterns []*regexp.Regexp // 敏感信息正则模式
    Keywords []string         // 敏感关键词
}

func (s *Sanitizer) SanitizeLog(text string) string      // 日志脱敏
func (s *Sanitizer) SanitizeOutput(text string) string    // 输出脱敏
func (s *Sanitizer) DetectSensitive(text string) []string // 检测敏感信息
```

#### 12.5 安全配置

```yaml
# config.yaml 新增安全配置
security:
  # 工具权限
  default_permission: "moderate"
  blocked_commands:
    - "rm -rf /"
    - "mkfs"
    - "dd if=/dev"
    - ":(){:|:&};:"
  allowed_dirs:
    - "/Users/weiketang/GolandProjects/self-agent"
    - "/tmp/self-agent"
  
  # API 认证
  auth:
    enabled: true
    api_keys:
      - key: "${SELF_AGENT_ADMIN_KEY}"  # 环境变量注入
        role: "admin"
        rate_limit: 100
      - key: "${SELF_AGENT_USER_KEY}"
        role: "user"
        rate_limit: 30
  
  # 沙箱限制
  sandbox:
    max_cpu_time: 60      # 秒
    max_memory_mb: 512    # MB
    max_output_bytes: 1048576  # 1MB
```

### 验收标准

- [ ] exec_shell 命令黑名单生效，危险命令被拦截并返回提示
- [ ] 工具调用前进行权限校验，无权限时拒绝执行
- [ ] HTTP API 需要有效的 API Key 才能访问，无效 Key 返回 401
- [ ] 请求频率限制生效，超限返回 429
- [ ] 日志中不出现任何明文 API Key 或密码
- [ ] 配置文件支持环境变量注入敏感信息
- [ ] 审计日志完整记录所有安全相关事件

### 当前状态

- **状态**：⏳ 未开始
- **进度**：0%

---

## 阶段完成标准

P3 阶段全部完成的标志：
1. ✅ 所有模块使用结构化日志，具备完整的链路追踪能力
2. ✅ 支持至少 2 种模型厂商，具备自动故障转移能力
3. ✅ 安全体系完善，工具执行在沙箱内，API 有认证鉴权
4. ✅ 具备 Token 消耗和成本的实时监控能力
5. ✅ 所有功能有完善的单元测试和集成测试

## 与 OpenClaw 的对标

| OpenClaw 能力 | 本阶段需求 | 对标程度 |
|---------------|-----------|---------|
| Observability | 需求10-可观测性与日志 | 🟢 完全对标 |
| Model Agnostic | 需求11-多模型支持 | 🟢 完全对标 |
| Secure Execution | 需求12-安全与权限控制 | 🟢 完全对标 |
| Cost Control | 需求10.3-指标监控 | 🟡 基础对标（OpenClaw 有更细粒度的成本优化策略） |
| Multi-Provider Failover | 需求11.4-故障转移 | 🟢 完全对标 |
