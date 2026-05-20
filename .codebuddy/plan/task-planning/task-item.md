# 实施计划

- [ ] 1. 定义 Plan-and-Solve 核心数据结构
   - 新建 `model/plan.go` 文件，定义 `Plan`、`SubTask`、`StrategyResult` 等结构体
   - `SubTask` 包含 ID、Description、Dependencies、Status、Result 字段
   - `Plan` 包含 TaskID、Goal、SubTasks、Version、CreatedAt 字段
   - `StrategyResult` 包含 Mode（react/plan_and_solve）、Confidence、Reason 字段
   - 定义子任务状态常量：pending、running、completed、failed
   - _需求：1.4、5.1、5.2、7.1_

- [ ] 2. 实现正则匹配层
   - 新建 `agent/strategy_regex.go` 文件
   - 实现 `regexMatch(input string) (string, bool)` 函数
   - 编写 Plan-and-Solve 模式的正则规则：匹配"先…再…然后…"、"步骤"、"流程"、"一步步"、"生成"等关键词且字数 < 50
   - 编写 ReAct 模式的正则规则：匹配"查一下"、"看看"等关键词，或字数 > 200
   - 将正则规则定义为包级常量/变量，方便后续扩展
   - _需求：2.1、2.2、2.3、2.4、2.5_

- [ ] 3. 实现 LLM 决策层
   - 新建 `agent/strategy_llm.go` 文件
   - 实现 `llmDecision(agent *Agent, input string) (*StrategyResult, error)` 函数
   - 内置 Task Router 提示词模板，调用 `Agent.SingleAsk()` 方法
   - 解析 LLM 返回的 JSON 响应，提取 mode、confidence、reason
   - JSON 解析失败或 LLM 调用失败时返回默认 ReAct 策略
   - 记录决策日志（mode、confidence、reason）
   - _需求：3.1、3.2、3.3、3.4、3.5_

- [ ] 4. 实现策略路由总控函数
   - 新建 `agent/strategy.go` 文件
   - 实现 `selectStrategy(agent *Agent, input string) (*StrategyResult, error)` 函数
   - 按优先级调用：空输入检查 → 正则匹配层 → LLM 决策层
   - 任何环节出错时默认回退到 ReAct 模式
   - _需求：1.1、1.2、1.3、1.6_

- [ ] 5. 实现 Plan-and-Solve 执行引擎
   - 新建 `agent/planner.go` 文件
   - 实现 `generatePlan(agent *Agent, goal string, history []model.AgentMessage) (*Plan, error)` 函数，调用 LLM 生成结构化执行计划
   - 实现 `buildDAG(plan *Plan) ([][]string, error)` 函数，将子任务构建为 DAG 并进行拓扑排序，检测循环依赖
   - 实现 `executePlan(agent *Agent, plan *Plan, history []model.AgentMessage) (string, error)` 函数，按拓扑序逐个执行子任务
   - 每个子任务通过 `Agent.Ask()` 执行（复用 ReAct 能力），执行结果注入后续子任务上下文
   - 所有子任务完成后汇总结果生成最终答案
   - _需求：5.1、5.2、5.3、5.4、5.5、5.6_

- [ ] 6. 实现动态调整（Re-planning）
   - 在 `agent/planner.go` 中新增 `replan(agent *Agent, plan *Plan, failedTask *SubTask, reason string) (*Plan, error)` 函数
   - 将当前计划状态和失败原因传给 LLM，生成调整后的新计划
   - 新计划递增 version 字段，记录变更原因
   - Re-planning 次数上限为 3 次，超过则终止执行并返回错误
   - 在 `executePlan` 中集成 Re-planning 逻辑：子任务失败时触发
   - _需求：6.1、6.2、6.3、6.4、6.5_

- [ ] 7. 实现双路径并行执行与结果对比
   - 新建 `agent/dual_path.go` 文件
   - 实现 `executeDualPath(agent *Agent, input string, history []model.AgentMessage) (string, error)` 函数
   - 使用 goroutine + WaitGroup 同时启动 ReAct 和 Plan-and-Solve 两条路径
   - 设置 60s 超时，超时后取已完成路径的结果
   - 两条路径均完成后，调用 LLM 使用 Result Evaluator 提示词进行结果对比
   - 一条路径失败时直接返回另一条路径结果；两条均失败则返回错误
   - _需求：4.1、4.2、4.3、4.4、4.5、4.6_

- [ ] 8. 实现进度追踪与状态管理
   - 在 `model/plan.go` 中为 `Plan` 添加进度计算方法 `Progress() string`（返回"已完成 x/y 步骤"）
   - 在 `executePlan` 执行过程中，每个子任务状态变更时更新 Plan 中对应 SubTask 的 Status
   - 在 Session 层集成进度推送：Plan-and-Solve 执行中通过 `SendToChannel` 推送进度通知
   - _需求：7.1、7.2、7.3、7.4_

- [ ] 9. 集成策略路由到 Session 主循环
   - 修改 `session/session.go` 的 `Start()` 方法
   - 在 `Agent.Ask()` 调用前，先调用 `selectStrategy()` 判断策略
   - 根据策略结果分流：confidence > 0.5 走单路径（ReAct 或 Plan-and-Solve）；confidence ≤ 0.5 走双路径
   - Plan-and-Solve 单路径时调用 `executePlan`；ReAct 单路径时保持原有 `Agent.Ask()` 逻辑
   - 双路径时调用 `executeDualPath`
   - _需求：1.1、1.4、1.5_

- [ ] 10. 编写单元测试
   - 新建 `agent/strategy_regex_test.go`，测试正则匹配层各种输入场景（Plan-and-Solve 关键词、ReAct 关键词、超长输入、未匹配输入）
   - 新建 `agent/planner_test.go`，测试 DAG 构建（正常拓扑排序、循环依赖检测）
   - 新建 `agent/strategy_test.go`，测试策略路由总控逻辑（正则命中直接返回、LLM 回退等）
   - _需求：2.1-2.4、5.2、边界情况 5_
