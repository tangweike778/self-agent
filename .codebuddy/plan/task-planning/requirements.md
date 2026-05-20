# 需求文档

## 引言

本功能为 self-agent 项目实现**任务规划与分解（Task Planning / Decomposition）**能力。核心设计思路是：Plan-and-Solve 与已有的 ReAct 执行模型**非互斥**，用户会话进来后首先通过 `selectStrategy` 函数判断走哪一种策略路径。

判断采用**分层决策**机制：正则匹配层快速分流 → LLM 决策层精确判断 → 低置信度时双路径并行执行 + 结果对比。

该功能对标 OpenClaw 级别智能体的任务规划能力，让 Agent 从"单步执行"进化为"先规划再执行"的智能体。

---

## 需求

### 需求 1：策略路由（Strategy Router）

**用户故事：** 作为一名 Agent 系统，我希望能够根据用户输入自动选择最优执行策略（ReAct 或 Plan-and-Solve），以便在简单任务上保持高效、在复杂任务上保证质量。

#### 验收标准

1. WHEN 用户输入到达 Agent THEN 系统 SHALL 调用 `selectStrategy` 函数进行策略判断
2. WHEN 正则匹配层命中明确模式 THEN 系统 SHALL 直接返回对应策略，跳过 LLM 决策层
3. WHEN 正则匹配层未命中 THEN 系统 SHALL 将用户输入传递给 LLM 决策层进行判断
4. WHEN LLM 决策层返回 confidence > 0.5 THEN 系统 SHALL 走单一路径执行
5. WHEN LLM 决策层返回 confidence ≤ 0.5 THEN 系统 SHALL 走双路径并行执行，最终由 LLM 进行结果对比选优
6. IF 策略判断过程出错 THEN 系统 SHALL 默认回退到 ReAct 模式

---

### 需求 2：正则匹配层（Regex Matching Layer）

**用户故事：** 作为一名 Agent 系统，我希望通过正则快速匹配用户输入的特征模式，以便在明确场景下零延迟地完成策略分流，避免不必要的 LLM 调用开销。

#### 验收标准

1. WHEN 用户输入匹配到"先…再…然后…"、"步骤"、"流程"、"一步步"、"生成"等关键词 AND 字数 < 50 THEN 系统 SHALL 判断使用 Plan-and-Solve 模式
2. WHEN 用户输入匹配到"查一下"、"看看"等关键词 THEN 系统 SHALL 判断使用 ReAct 模式
3. WHEN 用户输入字数 > 200 THEN 系统 SHALL 判断使用 ReAct 模式
4. WHEN 用户输入未命中任何正则规则 THEN 系统 SHALL 返回"未匹配"状态，交由 LLM 决策层处理
5. IF 正则规则需要扩展 THEN 系统 SHALL 支持通过配置文件或代码常量方便地新增规则

---

### 需求 3：LLM 决策层（LLM Decision Layer）

**用户故事：** 作为一名 Agent 系统，我希望在正则匹配层无法判断时，利用 LLM 的语义理解能力进行精确的策略决策，以便处理模糊或复杂的用户输入场景。

#### 验收标准

1. WHEN 正则匹配层返回"未匹配" THEN 系统 SHALL 调用 LLM 并使用 Task Router 提示词进行策略判断
2. WHEN LLM 返回结果 THEN 系统 SHALL 解析 JSON 响应，提取 mode、confidence 和 reason 字段
3. WHEN LLM 返回的 JSON 格式不合法 THEN 系统 SHALL 默认回退到 ReAct 模式
4. WHEN LLM 调用超时或失败 THEN 系统 SHALL 默认回退到 ReAct 模式
5. IF LLM 决策层被调用 THEN 系统 SHALL 记录决策日志（包含 mode、confidence、reason）用于后续分析优化

**Task Router 提示词：**
```
你是一个任务调度专家（Task Router）。
你的任务是判断：给定用户请求，应该使用哪种执行模式。

可选模式：
1. **ReAct**：逐步推理 + 行动，适合简单、探索性、交互式任务。
2. **Plan-and-Solve**：先规划再执行，适合复杂、多步骤、可预测任务。

判断标准：
- 如果任务 **步骤少、目标模糊、需要边试边改** → ReAct
- 如果任务 **步骤多、目标明确、可提前规划** → Plan-and-Solve

请输出严格的 JSON，不要解释。

输入：
{user_task}

输出格式：
{
  "mode": "react" | "plan_and_solve",
  "confidence": 0.0~1.0,
  "reason": "一句话说明理由"
}
```

---

### 需求 4：双路径并行执行与结果对比（Dual-Path Execution）

**用户故事：** 作为一名 Agent 系统，我希望在策略判断置信度较低时，同时走两条路径执行并对比结果，以便在不确定场景下仍能给出最优答案。

#### 验收标准

1. WHEN confidence ≤ 0.5 THEN 系统 SHALL 同时启动 ReAct 和 Plan-and-Solve 两条路径并行执行
2. WHEN 两条路径均执行完成 THEN 系统 SHALL 调用 LLM 使用 Result Evaluator 提示词进行结果对比
3. WHEN 结果对比完成 THEN 系统 SHALL 返回评审推荐的最优结果给用户
4. IF 其中一条路径执行失败 THEN 系统 SHALL 直接返回另一条路径的结果
5. IF 两条路径均失败 THEN 系统 SHALL 返回错误信息给用户
6. WHEN 双路径执行 THEN 系统 SHALL 设置合理的超时时间，避免长时间等待

**Result Evaluator 提示词：**
```
你是一个专业的结果评审专家（Result Evaluator）。
现在有同一个用户问题，Agent 通过不同路径得到了多个结果。
请你从以下维度进行对比，并给出最终推荐结果。

用户问题：
{user_query}

候选结果：
{results_json}

评审维度：
1. 准确性（是否回答问题）
2. 完整性（是否覆盖关键点）
3. 逻辑性（推理是否清晰）
4. 可执行性（是否可直接使用）
5. 冗余度（是否啰嗦）

输出格式（严格 JSON）：
{
  "best_result_id": "result_2",
  "ranking": [
    {"id": "result_2", "score": 9},
    {"id": "result_1", "score": 7}
  ],
  "comparison_summary": "简要对比说明",
  "final_answer": "推荐给用户的最终答案"
}
```

---

### 需求 5：Plan-and-Solve 执行引擎

**用户故事：** 作为一名 Agent 系统，我希望具备将复杂任务分解为子任务并按依赖关系有序执行的能力，以便高效完成多步骤、可预测的复杂任务。

#### 验收标准

1. WHEN 策略路由决定使用 Plan-and-Solve 模式 THEN 系统 SHALL 调用 LLM 生成结构化执行计划（Plan）
2. WHEN Plan 生成完成 THEN 系统 SHALL 将子任务构建为 DAG（有向无环图）结构
3. WHEN 执行计划时 THEN 系统 SHALL 按拓扑排序顺序执行子任务
4. WHEN 同一层级存在多个无依赖子任务 THEN 系统 SHALL 支持串行执行（初版），后续可优化为并行
5. WHEN 子任务执行完成 THEN 系统 SHALL 将结果注入后续依赖子任务的上下文中
6. WHEN 所有子任务完成 THEN 系统 SHALL 汇总结果生成最终答案

---

### 需求 6：动态调整与 Re-planning

**用户故事：** 作为一名 Agent 系统，我希望在子任务执行失败或发现新信息时能动态调整计划，以便应对执行过程中的不确定性。

#### 验收标准

1. WHEN 子任务执行失败 THEN 系统 SHALL 触发 Re-planning 判断
2. WHEN Re-planning 触发 THEN 系统 SHALL 将当前计划状态和失败原因传给 LLM，生成调整后的新计划
3. WHEN 新计划生成 THEN 系统 SHALL 递增 Plan 的 version 字段
4. IF Re-planning 次数超过 3 次 THEN 系统 SHALL 终止执行并返回错误信息
5. WHEN Re-planning 发生 THEN 系统 SHALL 记录变更原因用于追踪

---

### 需求 7：进度追踪与状态管理

**用户故事：** 作为一名用户，我希望能够了解复杂任务的当前执行进度，以便掌握任务完成情况。

#### 验收标准

1. WHEN 子任务状态变更 THEN 系统 SHALL 更新状态为 pending → running → completed / failed
2. WHEN 用户查询进度 THEN 系统 SHALL 返回已完成数/总数的进度信息
3. WHEN Plan-and-Solve 模式执行中 THEN 系统 SHALL 通过已绑定的 Channel 推送进度通知
4. IF 系统绑定了飞书渠道 THEN 系统 SHALL 发送类似"已完成 2/5 步骤"的进度消息

---

## 边界情况与约束

### 边界情况
1. **空输入**：用户输入为空时，不进入策略判断，直接提示用户输入内容
2. **超长输入**：字数 > 200 的输入直接走 ReAct，避免 Plan-and-Solve 对超长输入的规划开销
3. **LLM 返回非法 JSON**：解析失败时回退到 ReAct 模式
4. **双路径超时**：设置合理超时（建议 60s），超时后取已完成路径的结果
5. **子任务循环依赖**：DAG 构建时需检测环，发现环则拒绝执行并报错

### 技术约束
1. 正则匹配层不依赖 LLM，零延迟返回
2. LLM 决策层使用 `SingleAsk` 方法，复用现有 API 调用能力
3. Plan-and-Solve 的子任务执行复用现有 `Agent.Ask()` 方法（每个子任务享有完整 ReAct 能力）
4. 双路径并行执行使用 Go 的 goroutine + WaitGroup 实现
5. Re-planning 次数上限为 3 次，防止死循环

### 成功标准
1. 简单任务（如"查一下天气"）不触发规划流程，响应时间无明显增加
2. 复杂任务（如"先读取文件，再分析内容，然后生成报告"）能被正确分解为 2-5 个子任务
3. 策略路由判断准确率 > 80%（通过日志分析验证）
4. 双路径模式下最终结果质量不低于单路径最优结果
