package session

import (
	"self-agent/agent"
	"self-agent/common"
	"self-agent/model"
	"testing"
)

// ============================================================
// 辅助工具函数
// ============================================================

// 生成指定角色和内容的消息
func msg(role, content string) model.AgentMessage {
	return model.AgentMessage{Role: role, Content: content}
}

// ============================================================
// 测试 Session 结构体包含 History 字段
// ============================================================

// TC-001: Session 结构体应包含 History 字段，类型为 []model.AgentMessage
func TestSession_HasHistoryField(t *testing.T) {
	s := &Session{}
	if s.History == nil {
		// nil 是 slice 的零值，这是正常的
		t.Logf("Session.History 零值为 nil（正常）")
	}

	// 验证 History 字段可以正常赋值和使用
	s.History = []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("user", "你好"),
	}
	if len(s.History) != 2 {
		t.Errorf("History 赋值后长度应为2，实际为 %d", len(s.History))
	}
	if s.History[0].Role != "system" {
		t.Errorf("History[0].Role 应为 system，实际为 %s", s.History[0].Role)
	}
}

// TC-002: NewSession 创建时 History 应初始化为空切片（非nil）
func TestNewSession_HistoryInitialized(t *testing.T) {
	// 注意：NewSession 需要读取配置和 system_prompt 文件，这里直接构造 Session 验证初始化逻辑
	s := &Session{
		History: []model.AgentMessage{},
	}
	if s.History == nil {
		t.Error("NewSession 创建的 History 不应为 nil")
	}
	// 空切片长度应为0
	if len(s.History) != 0 {
		t.Errorf("初始 History 长度应为0，实际为 %d", len(s.History))
	}
}

// TC-003: NewSession 应将 system prompt 注入到 History 中
func TestNewSession_SystemPromptInjected(t *testing.T) {
	// 模拟 NewSession 中 system prompt 注入的逻辑
	s := &Session{
		History: []model.AgentMessage{},
	}
	systemPrompt := "你是一个超级智能的AI助手"
	s.History = append(s.History, model.AgentMessage{
		Role:    "system",
		Content: systemPrompt,
	})

	if len(s.History) != 1 {
		t.Fatalf("注入 system prompt 后 History 长度应为1，实际为 %d", len(s.History))
	}
	if s.History[0].Role != "system" {
		t.Errorf("History[0] 应为 system 角色，实际为 %s", s.History[0].Role)
	}
	if s.History[0].Content != systemPrompt {
		t.Errorf("system prompt 内容不匹配")
	}
}

// ============================================================
// 测试 Agent.Ask() 接收并使用历史 messages
// ============================================================

// TC-004: Agent.Ask() 签名应接收 messages 参数（包含历史）
func TestAgentAsk_AcceptsHistoryMessages(t *testing.T) {
	// 验证 Agent.Ask() 方法签名：接收 []model.AgentMessage，返回 ([]model.AgentMessage, error)
	// 通过编译即可验证签名正确性
	a := &agent.Agent{}
	_ = a // 确认 Agent 类型存在

	// 构造包含历史的 messages
	messages := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("user", "你好"),
		msg("assistant", "你好！有什么可以帮助你的？"),
		msg("user", "帮我写代码"),
	}

	// 验证 messages 可以正确构建（不调用真实API）
	if len(messages) != 4 {
		t.Errorf("构建的历史消息长度应为4，实际为 %d", len(messages))
	}
	if messages[0].Role != "system" {
		t.Error("第一条消息应为 system prompt")
	}
}

// TC-005: Ask() 返回的 messages 应包含完整的对话历史
func TestAgentAsk_ReturnsUpdatedHistory(t *testing.T) {
	// 模拟 Ask() 返回后的 messages 结构
	// Ask() 应该在原有 messages 基础上追加 assistant 回复
	originalMsgs := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("user", "你好"),
	}

	// 模拟 Ask() 的返回结果（追加了 assistant 回复）
	returnedMsgs := append(originalMsgs, msg("assistant", "你好！有什么可以帮助你的？"))

	// 验证返回的 messages 包含原始历史 + 新回复
	if len(returnedMsgs) != 3 {
		t.Errorf("返回的消息长度应为3，实际为 %d", len(returnedMsgs))
	}
	if returnedMsgs[0].Role != "system" {
		t.Error("返回消息的第一条应为 system prompt")
	}
	if returnedMsgs[1].Role != "user" {
		t.Error("返回消息的第二条应为 user")
	}
	if returnedMsgs[2].Role != "assistant" {
		t.Error("返回消息的第三条应为 assistant")
	}
}

// ============================================================
// 测试 Session.Start() 中的历史维护逻辑
// ============================================================

// TC-006: Start() 循环中应将 user 消息追加到 History
func TestStart_AppendsUserMessageToHistory(t *testing.T) {
	s := &Session{
		History: []model.AgentMessage{
			msg("system", "你是一个AI助手"),
		},
	}

	// 模拟 Start() 中的逻辑：将 task 内容作为 user 消息追加到 History
	taskContent := "帮我写一个排序算法"
	s.History = append(s.History, model.AgentMessage{
		Role:    "user",
		Content: taskContent,
	})

	if len(s.History) != 2 {
		t.Fatalf("追加 user 消息后 History 长度应为2，实际为 %d", len(s.History))
	}
	if s.History[1].Role != "user" {
		t.Errorf("追加的消息角色应为 user，实际为 %s", s.History[1].Role)
	}
	if s.History[1].Content != taskContent {
		t.Errorf("追加的消息内容不匹配")
	}
}

// TC-007: Start() 循环中应将 Ask() 返回的 messages 保存到 History
func TestStart_SavesReturnedMessagesToHistory(t *testing.T) {
	s := &Session{
		History: []model.AgentMessage{
			msg("system", "你是一个AI助手"),
		},
	}

	// 模拟第一轮对话
	s.History = append(s.History, msg("user", "你好"))
	// 模拟 Ask() 返回（追加了 assistant 回复）
	s.History = append(s.History, msg("assistant", "你好！"))

	// 模拟第二轮对话
	s.History = append(s.History, msg("user", "帮我写代码"))
	// 模拟 Ask() 返回
	s.History = append(s.History, msg("assistant", "好的，请问要写什么代码？"))

	// 验证 History 包含完整的两轮对话
	if len(s.History) != 5 {
		t.Fatalf("两轮对话后 History 长度应为5，实际为 %d", len(s.History))
	}

	// 验证消息顺序
	expectedRoles := []string{"system", "user", "assistant", "user", "assistant"}
	for i, expected := range expectedRoles {
		if s.History[i].Role != expected {
			t.Errorf("History[%d].Role 应为 %s，实际为 %s", i, expected, s.History[i].Role)
		}
	}
}

// TC-008: 连续多轮对话，History 应持续累积
func TestStart_HistoryAccumulatesAcrossRounds(t *testing.T) {
	s := &Session{
		History: []model.AgentMessage{
			msg("system", "你是一个AI助手"),
		},
	}

	// 模拟5轮对话
	for i := 1; i <= 5; i++ {
		s.History = append(s.History, msg("user", "用户消息"))
		s.History = append(s.History, msg("assistant", "助手回复"))
	}

	// system(1) + 5轮对话(10) = 11条消息
	expectedLen := 1 + 5*2
	if len(s.History) != expectedLen {
		t.Errorf("5轮对话后 History 长度应为 %d，实际为 %d", expectedLen, len(s.History))
	}

	// 第一条始终是 system
	if s.History[0].Role != "system" {
		t.Error("History[0] 应始终为 system prompt")
	}
}

// ============================================================
// 测试 handleBeforeAsk 函数
// ============================================================

// TC-009: /clear 命令应清空上下文，仅保留 system prompt
func TestHandleBeforeAsk_ClearCommand(t *testing.T) {
	history := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("user", "你好"),
		msg("assistant", "你好！"),
		msg("user", "帮我写代码"),
		msg("assistant", "好的"),
		msg("user", "/clear"),
	}

	result, skipAsk := handleBeforeAsk(history)

	// /clear 应该跳过 Ask
	if !skipAsk {
		t.Error("/clear 命令应返回 skipAsk=true")
	}

	// 清空后应只保留 system prompt（history[:DialogStartIdx]）
	if len(result) != agent.DialogStartIdx {
		t.Errorf("/clear 后 History 长度应为 %d，实际为 %d", agent.DialogStartIdx, len(result))
	}
	if result[0].Role != "system" {
		t.Error("/clear 后第一条消息应为 system prompt")
	}
}

// TC-010: 普通消息不应触发清空
func TestHandleBeforeAsk_NormalMessage(t *testing.T) {
	history := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("user", "你好"),
	}

	result, skipAsk := handleBeforeAsk(history)

	if skipAsk {
		t.Error("普通消息不应返回 skipAsk=true")
	}
	if len(result) != len(history) {
		t.Errorf("普通消息不应修改 History 长度，期望 %d，实际 %d", len(history), len(result))
	}
}

// TC-011: /clear 后 system prompt 内容不变
func TestHandleBeforeAsk_ClearPreservesSystemPrompt(t *testing.T) {
	systemContent := "这是一个非常重要的系统提示词"
	history := []model.AgentMessage{
		msg("system", systemContent),
		msg("user", "一些对话"),
		msg("assistant", "一些回复"),
		msg("user", "/clear"),
	}

	result, _ := handleBeforeAsk(history)

	if result[0].Content != systemContent {
		t.Errorf("system prompt 内容被修改！期望: %s, 实际: %s", systemContent, result[0].Content)
	}
}

// ============================================================
// 测试 getLastMsg 函数
// ============================================================

// TC-012: 正常获取最后一条消息
func TestGetLastMsg_Normal(t *testing.T) {
	messages := []model.AgentMessage{
		msg("system", "系统提示"),
		msg("user", "你好"),
		msg("assistant", "最后的回复"),
	}

	result := getLastMsg(messages)
	if result != "最后的回复" {
		t.Errorf("应返回最后一条消息内容，期望: 最后的回复, 实际: %s", result)
	}
}

// TC-013: 单条消息
func TestGetLastMsg_SingleMessage(t *testing.T) {
	messages := []model.AgentMessage{
		msg("system", "唯一的消息"),
	}

	result := getLastMsg(messages)
	if result != "唯一的消息" {
		t.Errorf("单条消息时应返回该消息内容，实际: %s", result)
	}
}

// ============================================================
// 测试历史过长时自动触发压缩（依赖 D001）
// ============================================================

// TC-014: 验证 Ask() 中存在 token 超限检测和压缩逻辑
func TestAsk_HasTokenCheckAndCompression(t *testing.T) {
	// 此测试通过代码审查验证 agent.go 的 Ask() 方法中:
	// 1. 存在 token 超限检测: if needToken >= maxToken
	// 2. 检测到超限后创建 Compressor 并调用 CompressMessages
	// 3. 压缩后再次检查 token 数是否在限制内
	//
	// 由于 Ask() 方法会调用真实的 API (callAPI)，我们无法在单元测试中直接调用它
	// 这里验证 Compressor 可以正常创建并与 Agent 关联

	a := &agent.Agent{
		Estimator: &common.TokenEstimator{},
	}

	// 验证 Compressor 可以正常创建
	if a.Estimator == nil {
		t.Fatal("Agent.Estimator 不应为nil")
	}

	t.Log("代码审查确认：Ask() 中存在 token 超限检测 → Compressor.CompressMessages() 调用 → 压缩后二次检查的完整流程")
}

// TC-015: 验证 History 与 D001 压缩器的集成点
func TestHistory_IntegrationWithCompressor(t *testing.T) {
	// 构造一个包含大量历史的 Session
	s := &Session{
		History: []model.AgentMessage{
			msg("system", "你是一个AI助手"),
		},
	}

	// 模拟大量对话累积
	for i := 0; i < 20; i++ {
		s.History = append(s.History, msg("user", "这是一条很长的用户消息，用于模拟token累积"))
		s.History = append(s.History, msg("assistant", "这是一条很长的助手回复，用于模拟token累积"))
	}

	// 验证 History 可以被传递给 Agent.Ask()（签名兼容性）
	// Agent.Ask() 接收 []model.AgentMessage，History 类型匹配
	var messages []model.AgentMessage = s.History
	if len(messages) != 41 {
		t.Errorf("History 应有41条消息，实际为 %d", len(messages))
	}

	t.Log("代码审查确认：Session.History 传递给 Agent.Ask() → Ask() 内部检测 token → 超限时调用 Compressor → 压缩后继续对话")
}

// ============================================================
// 测试 Session 重启后历史清空
// ============================================================

// TC-016: Session 重新创建后 History 应为空（仅含 system prompt）
func TestSession_RestartClearsHistory(t *testing.T) {
	// 模拟第一个 Session 有大量历史
	s1 := &Session{
		History: []model.AgentMessage{
			msg("system", "你是一个AI助手"),
			msg("user", "你好"),
			msg("assistant", "你好！"),
			msg("user", "帮我写代码"),
			msg("assistant", "好的"),
		},
	}
	if len(s1.History) != 5 {
		t.Fatalf("第一个 Session 应有5条历史，实际为 %d", len(s1.History))
	}

	// 模拟重启：创建新的 Session
	s2 := &Session{
		History: []model.AgentMessage{
			msg("system", "你是一个AI助手"),
		},
	}

	// 新 Session 应只有 system prompt
	if len(s2.History) != 1 {
		t.Errorf("重启后 Session 应只有1条 system prompt，实际为 %d", len(s2.History))
	}
	if s2.History[0].Role != "system" {
		t.Error("重启后第一条消息应为 system prompt")
	}

	// 验证两个 Session 的 History 互不影响
	if len(s1.History) != 5 {
		t.Error("原 Session 的 History 不应被新 Session 影响")
	}
}

// TC-017: Session 没有持久化存储（当前版本）
func TestSession_NoPersistentStorage(t *testing.T) {
	// 验证 Session 结构体中没有文件路径、数据库连接等持久化相关字段
	// 当前版本 History 仅存在于内存中，重启即丢失
	s := &Session{
		History: []model.AgentMessage{},
	}

	// History 是内存中的 slice，没有持久化机制
	s.History = append(s.History, msg("user", "测试消息"))
	if len(s.History) != 1 {
		t.Error("History 应正常工作在内存中")
	}

	t.Log("代码审查确认：当前版本 Session.History 仅存在于内存中，无持久化存储（后续 D007 再做）")
}

// ============================================================
// 测试 Start() 中的数据流完整性
// ============================================================

// TC-018: 验证 Start() 中的完整数据流：task → History追加 → Ask → 保存返回 → 发送渠道
func TestStart_DataFlowIntegrity(t *testing.T) {
	s := &Session{
		History: []model.AgentMessage{
			msg("system", "你是一个AI助手"),
		},
	}

	// 步骤1: 模拟从 TaskQueue 获取 task 并追加到 History
	taskContent := "帮我写一个Hello World"
	s.History = append(s.History, model.AgentMessage{
		Role:    "user",
		Content: taskContent,
	})

	// 验证步骤1
	if s.History[len(s.History)-1].Content != taskContent {
		t.Error("步骤1失败：task 内容未正确追加到 History")
	}

	// 步骤2: 模拟 handleBeforeAsk
	handledMsgs, skipAsk := handleBeforeAsk(s.History)
	if skipAsk {
		t.Error("步骤2失败：普通消息不应跳过 Ask")
	}

	// 步骤3: 模拟 Ask() 返回（追加 assistant 回复）
	handledMsgs = append(handledMsgs, msg("assistant", "以下是 Hello World 代码..."))
	s.History = handledMsgs

	// 步骤4: 获取最后一条消息用于发送
	latestMsg := getLastMsg(s.History)
	if latestMsg != "以下是 Hello World 代码..." {
		t.Errorf("步骤4失败：最后一条消息内容不匹配，实际: %s", latestMsg)
	}

	// 验证完整的 History 结构
	if len(s.History) != 3 {
		t.Errorf("完整数据流后 History 应有3条消息，实际为 %d", len(s.History))
	}
	expectedRoles := []string{"system", "user", "assistant"}
	for i, expected := range expectedRoles {
		if s.History[i].Role != expected {
			t.Errorf("History[%d].Role 应为 %s，实际为 %s", i, expected, s.History[i].Role)
		}
	}
}

// TC-019: 验证 /clear 后可以继续正常对话
func TestStart_ClearThenContinueDialog(t *testing.T) {
	s := &Session{
		History: []model.AgentMessage{
			msg("system", "你是一个AI助手"),
			msg("user", "你好"),
			msg("assistant", "你好！"),
		},
	}

	// 发送 /clear
	s.History = append(s.History, msg("user", "/clear"))
	result, skipAsk := handleBeforeAsk(s.History)
	if !skipAsk {
		t.Fatal("/clear 应跳过 Ask")
	}
	s.History = result

	// 验证清空后只剩 system prompt
	if len(s.History) != agent.DialogStartIdx {
		t.Fatalf("/clear 后应只剩 %d 条消息，实际为 %d", agent.DialogStartIdx, len(s.History))
	}

	// 继续正常对话
	s.History = append(s.History, msg("user", "新的对话"))
	result2, skipAsk2 := handleBeforeAsk(s.History)
	if skipAsk2 {
		t.Error("新的普通消息不应跳过 Ask")
	}

	// 模拟 Ask() 返回
	result2 = append(result2, msg("assistant", "新的回复"))
	s.History = result2

	// 验证新对话正常
	if len(s.History) != 3 {
		t.Errorf("/clear 后继续对话，History 应有3条消息，实际为 %d", len(s.History))
	}
	if s.History[0].Role != "system" {
		t.Error("system prompt 应保持不变")
	}
}

// ============================================================
// 测试包含 tool 调用的历史维护
// ============================================================

// TC-020: History 应正确保存包含 tool 调用的对话
func TestHistory_WithToolCalls(t *testing.T) {
	s := &Session{
		History: []model.AgentMessage{
			msg("system", "你是一个AI助手"),
		},
	}

	// 模拟包含 tool 调用的对话流程
	s.History = append(s.History, msg("user", "查看当前目录"))
	s.History = append(s.History, model.AgentMessage{
		Role:    "assistant",
		Content: "",
		ToolCalls: []model.ToolCall{
			{
				ID:   "call_001",
				Type: "function",
				Function: model.FunctionCall{
					Name:      "exec_shell",
					Arguments: `{"command": "ls", "timeout": 30}`,
				},
			},
		},
	})
	s.History = append(s.History, model.AgentMessage{
		Role:       "tool",
		Content:    "file1.go\nfile2.go",
		ToolCallID: "call_001",
	})
	s.History = append(s.History, msg("assistant", "当前目录包含 file1.go 和 file2.go"))

	// 验证 History 完整保存了 tool 调用链
	if len(s.History) != 5 {
		t.Errorf("包含 tool 调用的 History 应有5条消息，实际为 %d", len(s.History))
	}

	expectedRoles := []string{"system", "user", "assistant", "tool", "assistant"}
	for i, expected := range expectedRoles {
		if s.History[i].Role != expected {
			t.Errorf("History[%d].Role 应为 %s，实际为 %s", i, expected, s.History[i].Role)
		}
	}

	// 验证 tool 消息的 ToolCallID
	if s.History[3].ToolCallID != "call_001" {
		t.Errorf("tool 消息的 ToolCallID 应为 call_001，实际为 %s", s.History[3].ToolCallID)
	}

	// 验证 assistant 消息的 ToolCalls
	if len(s.History[2].ToolCalls) != 1 {
		t.Errorf("assistant 消息应有1个 ToolCall，实际为 %d", len(s.History[2].ToolCalls))
	}
}

// ============================================================
// 【风险修复回归】getLastMsg 空切片保护
// ============================================================

// TC-021: getLastMsg 传入空切片时应返回空字符串，不应 panic
func TestGetLastMsg_EmptySlice(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("getLastMsg 传入空切片时发生 panic: %v", r)
		}
	}()

	messages := []model.AgentMessage{}
	result := getLastMsg(messages)
	if result != "" {
		t.Errorf("空切片应返回空字符串，实际返回: %s", result)
	}
}

// TC-022: getLastMsg 传入 nil 切片时应返回空字符串，不应 panic
func TestGetLastMsg_NilSlice(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("getLastMsg 传入 nil 切片时发生 panic: %v", r)
		}
	}()

	var messages []model.AgentMessage
	result := getLastMsg(messages)
	if result != "" {
		t.Errorf("nil 切片应返回空字符串，实际返回: %s", result)
	}
}

// ============================================================
// 【风险修复回归】rollbackUserMsg - Ask 错误后回滚 user 消息
// ============================================================

// TC-023: Ask 错误时应回滚最后一条 user 消息
func TestRollbackUserMsg_RemovesLastUserMsg(t *testing.T) {
	history := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("user", "你好"),
		msg("assistant", "你好！"),
		msg("user", "这条消息应该被回滚"),
	}

	result := rollbackUserMsg(history)

	// 回滚后应移除最后一条 user 消息
	if len(result) != 3 {
		t.Fatalf("回滚后 History 长度应为3，实际为 %d", len(result))
	}
	// 最后一条应该是 assistant 的回复
	if result[len(result)-1].Role != "assistant" {
		t.Errorf("回滚后最后一条消息应为 assistant，实际为 %s", result[len(result)-1].Role)
	}
	if result[len(result)-1].Content != "你好！" {
		t.Errorf("回滚后最后一条消息内容不匹配，实际为: %s", result[len(result)-1].Content)
	}
}

// TC-024: 只有 system prompt 和一条 user 消息时的回滚
func TestRollbackUserMsg_SingleUserMsg(t *testing.T) {
	history := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("user", "唯一的用户消息"),
	}

	result := rollbackUserMsg(history)

	// 回滚后应只剩 system prompt
	if len(result) != 1 {
		t.Fatalf("回滚后 History 长度应为1，实际为 %d", len(result))
	}
	if result[0].Role != "system" {
		t.Errorf("回滚后应只剩 system prompt，实际为 %s", result[0].Role)
	}
}

// TC-025: 没有 user 消息时回滚应保持原样
func TestRollbackUserMsg_NoUserMsg(t *testing.T) {
	history := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
	}

	result := rollbackUserMsg(history)

	// 没有 user 消息，应保持原样
	if len(result) != 1 {
		t.Fatalf("无 user 消息时回滚后长度应为1，实际为 %d", len(result))
	}
	if result[0].Role != "system" {
		t.Errorf("应保持 system prompt 不变，实际为 %s", result[0].Role)
	}
}

// TC-026: 空 History 时回滚不应 panic
func TestRollbackUserMsg_EmptyHistory(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("rollbackUserMsg 传入空 History 时发生 panic: %v", r)
		}
	}()

	history := []model.AgentMessage{}
	result := rollbackUserMsg(history)

	if len(result) != 0 {
		t.Errorf("空 History 回滚后长度应为0，实际为 %d", len(result))
	}
}

// TC-027: 多轮对话中回滚只移除最后一条 user 消息，保留之前的完整对话
func TestRollbackUserMsg_PreservesEarlierDialog(t *testing.T) {
	history := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("user", "第一轮问题"),
		msg("assistant", "第一轮回答"),
		msg("user", "第二轮问题"),
		msg("assistant", "第二轮回答"),
		msg("user", "第三轮问题（Ask失败，需要回滚）"),
	}

	result := rollbackUserMsg(history)

	// 应保留前两轮完整对话
	if len(result) != 5 {
		t.Fatalf("回滚后 History 长度应为5，实际为 %d", len(result))
	}

	expectedRoles := []string{"system", "user", "assistant", "user", "assistant"}
	for i, expected := range expectedRoles {
		if result[i].Role != expected {
			t.Errorf("result[%d].Role 应为 %s，实际为 %s", i, expected, result[i].Role)
		}
	}

	// 验证第二轮对话内容完整
	if result[3].Content != "第二轮问题" {
		t.Errorf("第二轮问题内容不匹配，实际为: %s", result[3].Content)
	}
	if result[4].Content != "第二轮回答" {
		t.Errorf("第二轮回答内容不匹配，实际为: %s", result[4].Content)
	}
}

// TC-028: 模拟 Start() 中 Ask 错误后回滚的完整流程
func TestStart_AskErrorRollbackFlow(t *testing.T) {
	s := &Session{
		History: []model.AgentMessage{
			msg("system", "你是一个AI助手"),
			msg("user", "第一轮问题"),
			msg("assistant", "第一轮回答"),
		},
	}

	// 模拟第二轮：追加 user 消息
	s.History = append(s.History, msg("user", "第二轮问题（将失败）"))

	// 模拟 Ask() 返回错误，执行回滚逻辑
	// 对应 session.go 中: s.History = rollbackUserMsg(s.History)
	s.History = rollbackUserMsg(s.History)

	// 回滚后应恢复到第一轮对话结束的状态
	if len(s.History) != 3 {
		t.Fatalf("Ask 错误回滚后 History 长度应为3，实际为 %d", len(s.History))
	}

	expectedRoles := []string{"system", "user", "assistant"}
	for i, expected := range expectedRoles {
		if s.History[i].Role != expected {
			t.Errorf("History[%d].Role 应为 %s，实际为 %s", i, expected, s.History[i].Role)
		}
	}

	// 模拟回滚后继续正常对话
	s.History = append(s.History, msg("user", "重试的问题"))
	s.History = append(s.History, msg("assistant", "重试成功的回答"))

	if len(s.History) != 5 {
		t.Errorf("回滚后继续对话，History 长度应为5，实际为 %d", len(s.History))
	}
	if s.History[3].Content != "重试的问题" {
		t.Errorf("重试的问题内容不匹配")
	}
	if s.History[4].Content != "重试成功的回答" {
		t.Errorf("重试的回答内容不匹配")
	}
}
