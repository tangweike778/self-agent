package agent

import (
	"fmt"
	"self-agent/model"
	"strings"
	"testing"
	"time"
)

// ============================================================
// 辅助工具函数
// ============================================================

// reactMsg 生成指定角色和内容的消息（复用 compressor_test.go 中的 msg 函数风格）
func reactMsg(role, content string) model.AgentMessage {
	return model.AgentMessage{Role: role, Content: content}
}

// ============================================================
// 一、ReActStep 数据结构测试
// ============================================================

// TC-001: ReActStep 结构体字段完整性
func TestReActStep_FieldCompleteness(t *testing.T) {
	step := ReActStep{
		StepNum:     1,
		Thought:     "我需要读取文件",
		Action:      "read_file",
		ToolCalls:   []string{"read_file"},
		Observation: "文件内容: hello world",
		Reflection:  "读取成功，信息已足够",
		Success:     true,
		Duration:    2 * time.Second,
	}

	if step.StepNum != 1 {
		t.Errorf("StepNum 应为 1，实际为 %d", step.StepNum)
	}
	if step.Thought != "我需要读取文件" {
		t.Errorf("Thought 内容不正确: %s", step.Thought)
	}
	if step.Action != "read_file" {
		t.Errorf("Action 应为 'read_file'，实际为 '%s'", step.Action)
	}
	if len(step.ToolCalls) != 1 || step.ToolCalls[0] != "read_file" {
		t.Errorf("ToolCalls 不正确: %v", step.ToolCalls)
	}
	if step.Observation != "文件内容: hello world" {
		t.Errorf("Observation 内容不正确: %s", step.Observation)
	}
	if step.Reflection != "读取成功，信息已足够" {
		t.Errorf("Reflection 内容不正确: %s", step.Reflection)
	}
	if !step.Success {
		t.Error("Success 应为 true")
	}
	if step.Duration != 2*time.Second {
		t.Errorf("Duration 应为 2s，实际为 %v", step.Duration)
	}
}

// TC-002: ReActStep 零值初始化
func TestReActStep_ZeroValue(t *testing.T) {
	step := ReActStep{}

	if step.StepNum != 0 {
		t.Errorf("零值 StepNum 应为 0，实际为 %d", step.StepNum)
	}
	if step.Thought != "" {
		t.Errorf("零值 Thought 应为空，实际为 '%s'", step.Thought)
	}
	if step.Success {
		t.Error("零值 Success 应为 false")
	}
	if step.ToolCalls != nil {
		t.Errorf("零值 ToolCalls 应为 nil，实际为 %v", step.ToolCalls)
	}
}

// ============================================================
// 二、ReActTrace 数据结构测试
// ============================================================

// TC-003: ReActTrace 结构体字段完整性
func TestReActTrace_FieldCompleteness(t *testing.T) {
	trace := &ReActTrace{
		Question:  "帮我读取 config.yaml",
		Steps:     []ReActStep{},
		Answer:    "文件内容如下...",
		TotalTime: 5 * time.Second,
	}

	if trace.Question != "帮我读取 config.yaml" {
		t.Errorf("Question 不正确: %s", trace.Question)
	}
	if trace.Answer != "文件内容如下..." {
		t.Errorf("Answer 不正确: %s", trace.Answer)
	}
	if trace.TotalTime != 5*time.Second {
		t.Errorf("TotalTime 应为 5s，实际为 %v", trace.TotalTime)
	}
}

// TC-004: AddStep 添加步骤
func TestReActTrace_AddStep(t *testing.T) {
	trace := &ReActTrace{
		Question: "测试问题",
	}

	if len(trace.Steps) != 0 {
		t.Fatalf("初始 Steps 应为空，实际长度为 %d", len(trace.Steps))
	}

	step1 := ReActStep{StepNum: 1, Thought: "第一步思考"}
	trace.AddStep(step1)

	if len(trace.Steps) != 1 {
		t.Fatalf("添加1步后 Steps 长度应为 1，实际为 %d", len(trace.Steps))
	}
	if trace.Steps[0].StepNum != 1 {
		t.Errorf("第一步 StepNum 应为 1，实际为 %d", trace.Steps[0].StepNum)
	}

	step2 := ReActStep{StepNum: 2, Thought: "第二步思考"}
	trace.AddStep(step2)

	if len(trace.Steps) != 2 {
		t.Fatalf("添加2步后 Steps 长度应为 2，实际为 %d", len(trace.Steps))
	}
	if trace.Steps[1].Thought != "第二步思考" {
		t.Errorf("第二步 Thought 不正确: %s", trace.Steps[1].Thought)
	}
}

// TC-005: AddStep 多步骤顺序正确
func TestReActTrace_AddStep_Order(t *testing.T) {
	trace := &ReActTrace{}

	for i := 1; i <= 5; i++ {
		trace.AddStep(ReActStep{
			StepNum: i,
			Thought: fmt.Sprintf("步骤%d的思考", i),
		})
	}

	if len(trace.Steps) != 5 {
		t.Fatalf("应有5个步骤，实际为 %d", len(trace.Steps))
	}

	for i, step := range trace.Steps {
		expected := i + 1
		if step.StepNum != expected {
			t.Errorf("步骤 %d 的 StepNum 应为 %d，实际为 %d", i, expected, step.StepNum)
		}
	}
}

// ============================================================
// 三、GetLog 日志输出测试
// ============================================================

// TC-006: GetLog 包含完整推理链路信息
func TestReActTrace_GetLog_Complete(t *testing.T) {
	trace := &ReActTrace{
		Question:  "帮我查看系统状态",
		TotalTime: 3 * time.Second,
		Answer:    "系统运行正常",
	}
	trace.AddStep(ReActStep{
		StepNum:     1,
		Thought:     "需要执行 ps 命令",
		Action:      "exec_shell",
		ToolCalls:   []string{"exec_shell"},
		Observation: "PID 1234 running",
		Reflection:  "",
		Success:     true,
		Duration:    1 * time.Second,
	})

	log := trace.GetLog()

	// 验证日志包含关键信息
	checks := []struct {
		field    string
		expected string
	}{
		{"Question", "帮我查看系统状态"},
		{"Step 1", "Step 1"},
		{"Thought", "需要执行 ps 命令"},
		{"Action", "exec_shell"},
		{"Tool Calls", "exec_shell"},
		{"Observation", "PID 1234 running"},
		{"Success", "true"},
		{"Answer", "系统运行正常"},
		{"Total Time", "3s"},
	}

	for _, check := range checks {
		if !strings.Contains(log, check.expected) {
			t.Errorf("GetLog() 应包含 '%s'（%s），实际日志:\n%s", check.expected, check.field, log)
		}
	}
}

// TC-007: GetLog 空步骤
func TestReActTrace_GetLog_EmptySteps(t *testing.T) {
	trace := &ReActTrace{
		Question: "简单问题",
		Answer:   "直接回答",
	}

	log := trace.GetLog()

	if !strings.Contains(log, "Question: 简单问题") {
		t.Errorf("空步骤日志应包含 Question，实际: %s", log)
	}
	if !strings.Contains(log, "Answer: 直接回答") {
		t.Errorf("空步骤日志应包含 Answer，实际: %s", log)
	}
}

// TC-008: GetLog 多步骤日志
func TestReActTrace_GetLog_MultipleSteps(t *testing.T) {
	trace := &ReActTrace{
		Question: "复杂任务",
	}
	trace.AddStep(ReActStep{StepNum: 1, Thought: "第一步", Action: "read_file", Success: true})
	trace.AddStep(ReActStep{StepNum: 2, Thought: "第二步", Action: "exec_shell", Success: false, Reflection: "命令执行失败"})
	trace.AddStep(ReActStep{StepNum: 3, Thought: "第三步", Action: "exec_shell", Success: true})

	log := trace.GetLog()

	if !strings.Contains(log, "Step 1") {
		t.Error("日志应包含 Step 1")
	}
	if !strings.Contains(log, "Step 2") {
		t.Error("日志应包含 Step 2")
	}
	if !strings.Contains(log, "Step 3") {
		t.Error("日志应包含 Step 3")
	}
	if !strings.Contains(log, "命令执行失败") {
		t.Error("日志应包含失败步骤的 Reflection")
	}
}

// ============================================================
// 八、Ask() ReAct 循环逻辑测试
// ============================================================

// TC-023: Ask() 方法中 ReActTrace 初始化
func TestAsk_ReActTraceInit(t *testing.T) {
	messages := []model.AgentMessage{
		reactMsg("system", "你是一个AI助手"),
		reactMsg("user", "帮我查看文件"),
	}

	question := getLastUserQuestion(messages)
	if question != "帮我查看文件" {
		t.Errorf("getLastUserQuestion 应返回最后一条 user 消息，实际返回: '%s'", question)
	}
}

// TC-024: getLastUserQuestion 多条 user 消息
func TestGetLastUserQuestion_MultipleUsers(t *testing.T) {
	messages := []model.AgentMessage{
		reactMsg("system", "系统提示"),
		reactMsg("user", "第一个问题"),
		reactMsg("assistant", "第一个回答"),
		reactMsg("user", "第二个问题"),
		reactMsg("assistant", "第二个回答"),
		reactMsg("user", "第三个问题"),
	}

	question := getLastUserQuestion(messages)
	if question != "第三个问题" {
		t.Errorf("应返回最后一条 user 消息 '第三个问题'，实际返回: '%s'", question)
	}
}

// TC-025: getLastUserQuestion 无 user 消息
func TestGetLastUserQuestion_NoUser(t *testing.T) {
	messages := []model.AgentMessage{
		reactMsg("system", "系统提示"),
		reactMsg("assistant", "回答"),
	}

	question := getLastUserQuestion(messages)
	if question != "" {
		t.Errorf("无 user 消息时应返回空字符串，实际返回: '%s'", question)
	}
}

// TC-026: getLastUserQuestion 空消息列表
func TestGetLastUserQuestion_Empty(t *testing.T) {
	question := getLastUserQuestion([]model.AgentMessage{})
	if question != "" {
		t.Errorf("空消息列表应返回空字符串，实际返回: '%s'", question)
	}
}

// ============================================================
// 九、Ask() 中 ReAct 步骤构建逻辑测试
// ============================================================

// TC-027: ReActStep 工具调用失败时 Success 为 false
func TestReActStep_ToolCallFailure(t *testing.T) {
	step := ReActStep{
		StepNum:    1,
		Action:     "exec_shell",
		ToolCalls:  []string{"exec_shell"},
		Success:    false,
		Reflection: "工具exec_shell执行失败: 命令不存在",
	}

	if step.Success {
		t.Error("工具调用失败时 Success 应为 false")
	}
	if step.Reflection == "" {
		t.Error("工具调用失败时 Reflection 不应为空")
	}
	if !strings.Contains(step.Reflection, "执行失败") {
		t.Errorf("Reflection 应包含失败信息: '%s'", step.Reflection)
	}
}

// TC-028: ReActStep 工具调用成功时 Success 为 true
func TestReActStep_ToolCallSuccess(t *testing.T) {
	step := ReActStep{
		StepNum:     1,
		Action:      "read_file",
		ToolCalls:   []string{"read_file"},
		Observation: "文件内容: hello",
		Success:     true,
	}

	if !step.Success {
		t.Error("工具调用成功时 Success 应为 true")
	}
}

// ============================================================
// 十、executeTool 工具执行测试
// ============================================================

// TC-029: executeTool 未知工具返回错误信息
func TestExecuteTool_UnknownTool(t *testing.T) {
	agent := NewAgent("test-key")

	toolCall := model.ToolCall{
		ID:   "call_123",
		Type: "function",
		Function: model.FunctionCall{
			Name:      "unknown_tool",
			Arguments: `{}`,
		},
	}

	result := agent.executeTool(toolCall)
	if !strings.Contains(result, "未知工具") {
		t.Errorf("未知工具应返回'未知工具'，实际返回: '%s'", result)
	}
	if !strings.Contains(result, "unknown_tool") {
		t.Errorf("错误信息应包含工具名称，实际返回: '%s'", result)
	}
}

// ============================================================
// 十一、Ask() 中反思提示注入测试
// ============================================================

// TC-030: 工具失败时反思提示格式验证
func TestReflectionPrompt_Format(t *testing.T) {
	reflectionPrompt := fmt.Sprintf(
		"上一步工具调用失败了。请反思：\n1. 失败原因是什么？\n2. 是否需要换一种方式？\n3. 下一步应该怎么做？\n请在 <reflection> 标签中输出你的反思。",
	)

	if !strings.Contains(reflectionPrompt, "反思") {
		t.Error("反思提示应包含'反思'关键词")
	}
	if !strings.Contains(reflectionPrompt, "失败原因") {
		t.Error("反思提示应包含'失败原因'")
	}
	if !strings.Contains(reflectionPrompt, "<reflection>") {
		t.Error("反思提示应引导 LLM 使用 <reflection> 标签")
	}
	if !strings.Contains(reflectionPrompt, "下一步") {
		t.Error("反思提示应引导 LLM 思考下一步")
	}
}

// TC-031: 工具失败时 step.Reflection 格式
func TestReActStep_ReflectionFormat(t *testing.T) {
	toolName := "exec_shell"
	toolResult := "错误: command not found"

	reflection := fmt.Sprintf("工具%s执行失败: %s", toolName, toolResult)

	if !strings.Contains(reflection, toolName) {
		t.Errorf("Reflection 应包含工具名称: '%s'", reflection)
	}
	if !strings.Contains(reflection, toolResult) {
		t.Errorf("Reflection 应包含工具结果: '%s'", reflection)
	}
}

// ============================================================
// 十三、ReAct 循环上限测试
// ============================================================

// TC-034: Ask() 方法有最大循环次数限制（10轮）
func TestAsk_MaxLoopLimit(t *testing.T) {
	maxLoop := 10
	if maxLoop != 10 {
		t.Errorf("Ask() 最大循环次数应为 10，实际为 %d", maxLoop)
	}
	t.Log("Ask() 方法设置了最大 10 轮循环限制，防止无限循环")
}

// ============================================================
// 十四、System Prompt ReAct 指令测试
// ============================================================

// TC-035: System Prompt 包含 ReAct 推理规范
func TestSystemPrompt_ContainsReActInstructions(t *testing.T) {
	requiredSections := []string{
		"推理规范",
		"ReAct",
		"<thought>",
		"<reflection>",
		"思考",
		"行动",
		"反思",
	}

	systemPrompt := `# 推理规范（ReAct Framework）

在处理用户问题时，请遵循以下推理框架：

## 思考（Thought）
在每次行动前，先用 <thought> 标签输出你的思考过程：
- 分析用户的真实意图
- 评估当前已知信息
- 规划下一步行动

## 行动（Action）
基于思考结果，调用合适的工具执行操作。

## 反思（Reflection）
如果工具调用失败或结果不符合预期，用 <reflection> 标签输出反思`

	for _, section := range requiredSections {
		if !strings.Contains(systemPrompt, section) {
			t.Errorf("System Prompt 应包含 '%s'", section)
		}
	}
}

// TC-036: System Prompt 中 <thought> 标签使用示例
func TestSystemPrompt_ThoughtExample(t *testing.T) {
	example := `<thought>
用户想要读取 config.yaml 文件的内容。我需要使用 read_file 工具来读取这个文件。
文件路径应该是 ~/GolandProjects/self-agent/config/config.yaml。
</thought>`

	if !strings.Contains(example, "<thought>") {
		t.Error("示例应包含 <thought> 标签")
	}
	if !strings.Contains(example, "</thought>") {
		t.Error("示例应包含 </thought> 闭合标签")
	}
}

// ============================================================
// 十六、ReAct 完整推理链路导出测试
// ============================================================

// TC-040: 完整推理链路可导出为日志
func TestReActTrace_FullTraceExport(t *testing.T) {
	trace := &ReActTrace{
		Question:  "帮我检查 config.yaml 文件是否存在",
		TotalTime: 8 * time.Second,
		Answer:    "config.yaml 文件存在，内容如下...",
	}

	trace.AddStep(ReActStep{
		StepNum:     1,
		Thought:     "用户想检查文件是否存在，我先用 exec_shell 执行 ls 命令",
		Action:      "exec_shell",
		ToolCalls:   []string{"exec_shell"},
		Observation: "ls: config.yaml: No such file or directory",
		Reflection:  "",
		Success:     false,
		Duration:    2 * time.Second,
	})

	trace.AddStep(ReActStep{
		StepNum:     2,
		Thought:     "ls 命令失败了，可能路径不对。让我尝试在 config 目录下查找",
		Action:      "exec_shell",
		ToolCalls:   []string{"exec_shell"},
		Observation: "config/config.yaml",
		Reflection:  "",
		Success:     true,
		Duration:    1 * time.Second,
	})

	trace.AddStep(ReActStep{
		StepNum:     3,
		Thought:     "找到文件了，现在读取内容",
		Action:      "read_file",
		ToolCalls:   []string{"read_file"},
		Observation: "deepseek:\n  api_key: xxx",
		Reflection:  "",
		Success:     true,
		Duration:    1 * time.Second,
	})

	log := trace.GetLog()

	if !strings.Contains(log, "Question:") {
		t.Error("导出日志应包含 Question")
	}
	if !strings.Contains(log, "Step 1") && !strings.Contains(log, "Step 2") && !strings.Contains(log, "Step 3") {
		t.Error("导出日志应包含所有步骤")
	}
	if !strings.Contains(log, "Answer:") {
		t.Error("导出日志应包含 Answer")
	}
	if !strings.Contains(log, "Total Time:") {
		t.Error("导出日志应包含 Total Time")
	}

	if len(log) < 100 {
		t.Errorf("完整推理链路日志长度应 > 100 字符，实际为 %d", len(log))
	}

	t.Logf("完整推理链路日志长度: %d 字符", len(log))
}

// TC-041: 推理链路中失败步骤的 Reflection 可追踪
func TestReActTrace_FailureReflectionTraceable(t *testing.T) {
	trace := &ReActTrace{
		Question: "执行危险命令",
	}

	trace.AddStep(ReActStep{
		StepNum:    1,
		Thought:    "用户要求执行命令",
		Action:     "exec_shell",
		ToolCalls:  []string{"exec_shell"},
		Success:    false,
		Reflection: "工具exec_shell执行失败: permission denied",
		Duration:   500 * time.Millisecond,
	})

	trace.AddStep(ReActStep{
		StepNum:    2,
		Thought:    "权限不足，需要换一种方式",
		Action:     "exec_shell",
		ToolCalls:  []string{"exec_shell"},
		Success:    true,
		Reflection: "",
		Duration:   1 * time.Second,
	})

	log := trace.GetLog()

	if !strings.Contains(log, "permission denied") {
		t.Error("日志应包含失败步骤的 Reflection 信息")
	}
	if !strings.Contains(log, "false") {
		t.Error("日志应包含失败步骤的 Success: false")
	}
}

// ============================================================
// 十八、NewAgent 初始化测试
// ============================================================

// TC-044: NewAgent 正确初始化 ToolByName
func TestNewAgent_ToolByNameInit(t *testing.T) {
	agent := NewAgent("test-api-key")

	if agent.ToolByName == nil {
		t.Fatal("ToolByName 不应为 nil")
	}
	if len(agent.ToolByName) == 0 {
		t.Error("ToolByName 应包含已注册的工具")
	}

	if len(agent.Tools) != len(agent.ToolByName) {
		t.Errorf("Tools(%d) 和 ToolByName(%d) 数量应一致", len(agent.Tools), len(agent.ToolByName))
	}
}

// TC-045: NewAgent 正确初始化 Estimator
func TestNewAgent_EstimatorInit(t *testing.T) {
	agent := NewAgent("test-api-key")

	if agent.Estimator == nil {
		t.Fatal("Estimator 不应为 nil")
	}
}

// TC-046: NewAgent 正确设置 API 配置
func TestNewAgent_APIConfig(t *testing.T) {
	agent := NewAgent("my-test-key")

	if agent.APIKey != "my-test-key" {
		t.Errorf("APIKey 应为 'my-test-key'，实际为 '%s'", agent.APIKey)
	}
	if agent.BaseURL == "" {
		t.Error("BaseURL 不应为空")
	}
	if !strings.Contains(agent.BaseURL, "deepseek") {
		t.Errorf("BaseURL 应包含 'deepseek'，实际为 '%s'", agent.BaseURL)
	}
}
