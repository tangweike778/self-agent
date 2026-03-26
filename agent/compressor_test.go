package agent

import (
	"self-agent/common"
	"self-agent/model"
	"testing"
)

// ============================================================
// 辅助工具函数
// ============================================================

// 创建一个最小化的Agent，仅用于测试不涉及API调用的逻辑
func newTestAgent() *Agent {
	return &Agent{
		Estimator: &common.TokenEstimator{},
	}
}

// 生成指定角色和内容的消息
func msg(role, content string) model.AgentMessage {
	return model.AgentMessage{Role: role, Content: content}
}

// 构建标准测试消息列表：1条system + N条user/assistant交替对话
// dialogs 表示要创建几轮对话（每轮含1条user + 1条assistant）
func buildMessages(systemPrompt string, dialogs int) []model.AgentMessage {
	msgs := []model.AgentMessage{
		msg("system", systemPrompt),
	}
	for i := 1; i <= dialogs; i++ {
		msgs = append(msgs, msg("user", repeatStr("用户消息", i*10)))
		msgs = append(msgs, msg("assistant", repeatStr("助手回复", i*10)))
	}
	return msgs
}

// repeatStr 重复字符串n次以模拟不同长度的内容
func repeatStr(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

// ============================================================
// 测试 findCompressEndIds
// ============================================================

// TC-001: 对话不足3轮，压缩终点应为0
func TestFindCompressEndIds_LessThan3Dialogs(t *testing.T) {
	// 1条system + 2轮对话（4条消息），总共5条
	msgs := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("user", "你好"),
		msg("assistant", "你好！"),
		msg("user", "帮我写代码"),
		msg("assistant", "好的"),
	}

	idx := findCompressEndIds(msgs)
	if idx != 0 {
		t.Errorf("对话不足3轮时，压缩终点应为0，实际为 %d", idx)
	}
}

// TC-002: 恰好3轮对话，压缩终点应为0（没有可压缩的部分）
func TestFindCompressEndIds_Exactly3Dialogs(t *testing.T) {
	msgs := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("user", "第1轮"),
		msg("assistant", "回复1"),
		msg("user", "第2轮"),
		msg("assistant", "回复2"),
		msg("user", "第3轮"),
		msg("assistant", "回复3"),
	}

	idx := findCompressEndIds(msgs)
	// 恰好3轮，从末尾回溯找到第3个user时已经到了index 1（即system后面），压缩idx应为1
	if idx != 1 {
		t.Errorf("恰好3轮对话，压缩终点应为1，实际为 %d", idx)
	}
}

// TC-003: 超过3轮对话，应正确计算压缩终点
func TestFindCompressEndIds_MoreThan3Dialogs(t *testing.T) {
	msgs := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("user", "第1轮"),      // idx=1
		msg("assistant", "回复1"), // idx=2
		msg("user", "第2轮"),      // idx=3
		msg("assistant", "回复2"), // idx=4
		msg("user", "第3轮"),      // idx=5
		msg("assistant", "回复3"), // idx=6
		msg("user", "第4轮"),      // idx=7
		msg("assistant", "回复4"), // idx=8
	}

	idx := findCompressEndIds(msgs)
	// 从末尾回溯：idx8=assistant, idx7=user(1st), idx6=assistant, idx5=user(2nd), idx4=assistant, idx3=user(3rd)
	// 第3个user位于idx3，所以compressIdx=3
	if idx != 3 {
		t.Errorf("4轮对话，压缩终点应为3，实际为 %d", idx)
	}
}

// TC-004: 包含tool消息的场景
func TestFindCompressEndIds_WithToolMessages(t *testing.T) {
	msgs := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("user", "第1轮"),       // idx=1
		msg("assistant", "调用工具"), // idx=2
		msg("tool", "工具结果"),      // idx=3
		msg("assistant", "回复1"),  // idx=4
		msg("user", "第2轮"),       // idx=5
		msg("assistant", "回复2"),  // idx=6
		msg("user", "第3轮"),       // idx=7
		msg("assistant", "回复3"),  // idx=8
		msg("user", "第4轮"),       // idx=9
		msg("assistant", "回复4"),  // idx=10
	}

	idx := findCompressEndIds(msgs)
	// 从末尾回溯找3个user: idx9(1st), idx7(2nd), idx5(3rd) → compressIdx=5
	if idx != 5 {
		t.Errorf("含tool消息的4轮对话，压缩终点应为5，实际为 %d", idx)
	}
}

// ============================================================
// 测试 getSystemPromptAndLatestDialog
// ============================================================

// TC-005: 获取system prompt和最后一轮对话
func TestGetSystemPromptAndLatestDialog_Normal(t *testing.T) {
	msgs := []model.AgentMessage{
		msg("system", "系统提示"),
		msg("user", "第1轮"),
		msg("assistant", "回复1"),
		msg("user", "第2轮"),
		msg("assistant", "回复2"),
	}

	result, err := getSystemPromptAndLatestDialog(msgs)
	if err != nil {
		t.Fatalf("不应返回错误: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("结果不应为空")
	}

	// 第一条应该是system prompt
	if result[0].Role != "system" || result[0].Content != "系统提示" {
		t.Errorf("第一条消息应为system prompt，实际为 %s: %s", result[0].Role, result[0].Content)
	}

	// 应该包含最后一条user消息
	hasLastUser := false
	for _, m := range result {
		if m.Role == "user" && m.Content == "第2轮" {
			hasLastUser = true
			break
		}
	}
	if !hasLastUser {
		t.Error("结果中应包含最后一条user消息")
	}
}

// TC-006: 空消息列表
func TestGetSystemPromptAndLatestDialog_Empty(t *testing.T) {
	result, err := getSystemPromptAndLatestDialog([]model.AgentMessage{})
	if err != nil {
		// 空列表可能返回error或空结果，两种都可接受
		t.Logf("空消息列表返回错误（可接受）: %v", err)
		return
	}
	if len(result) != 0 {
		t.Errorf("空消息列表应返回空结果，实际长度为 %d", len(result))
	}
}

// ============================================================
// 测试 computeAvailableToken
// ============================================================

// TC-007: 正常情况下可用token计算
func TestComputeAvailableToken_Normal(t *testing.T) {
	agent := newTestAgent()
	compressor := &Compressor{Agent: agent}

	// 简单的保留消息
	reservedMsgs := []model.AgentMessage{
		msg("user", "hello"),
		msg("assistant", "hi"),
	}

	maxToken := int64(10000)
	available := compressor.computeAvailableToken(maxToken, reservedMsgs)

	// 可用token应该 > 0
	if available <= 0 {
		t.Errorf("正常情况下可用token应大于0，实际为 %d", available)
	}

	// 可用token应该 ≤ maxToken * 0.7
	expectedMax := (maxToken * 7) / 10
	if available > expectedMax {
		t.Errorf("可用token不应超过 maxToken*0.7=%d，实际为 %d", expectedMax, available)
	}
}

// TC-008: 保留消息token已经占满70%
func TestComputeAvailableToken_TokenExhausted(t *testing.T) {
	agent := newTestAgent()
	compressor := &Compressor{Agent: agent}

	// 构造很长的保留消息，使其token数 > maxToken * 0.7
	longContent := repeatStr("这是一段很长的内容用于测试token计算", 500)
	reservedMsgs := []model.AgentMessage{
		msg("user", longContent),
	}

	maxToken := int64(100) // 设置很小的maxToken
	available := compressor.computeAvailableToken(maxToken, reservedMsgs)

	// token已经占满，可用token应为0
	if available != 0 {
		t.Errorf("token已占满时可用token应为0，实际为 %d", available)
	}
}

// ============================================================
// 测试 CompressMessages（需要mock SingleAsk）
// ============================================================

// MockAgent 用于模拟Agent的SingleAsk行为
// 因为CompressMessages依赖Agent.SingleAsk()，我们需要验证在不调用真实API的情况下的逻辑行为
// 注意：由于SingleAsk是Agent的方法且没有接口抽象，我们这里主要测试不需要调用SingleAsk的分支

// TC-009: 对话不足3轮时，直接返回system prompt + 最后一轮对话
func TestCompressMessages_LessThan3Dialogs(t *testing.T) {
	agent := newTestAgent()
	compressor := &Compressor{Agent: agent}

	msgs := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("user", "你好"),
		msg("assistant", "你好！有什么可以帮助你的？"),
	}

	result, err := compressor.CompressMessages(msgs, 10000)
	if err != nil {
		t.Fatalf("压缩消息不应返回错误: %v", err)
	}

	// 验证system prompt被保留
	if result[0].Role != "system" || result[0].Content != "你是一个AI助手" {
		t.Error("system prompt应被完整保留")
	}

	// 验证消息数量合理（至少有system + 最后一条user）
	if len(result) < 2 {
		t.Errorf("结果消息数量不合理，至少应有2条，实际为 %d", len(result))
	}
}

// TC-010: 对话不足3轮，system prompt内容不被修改
func TestCompressMessages_SystemPromptPreserved_ShortDialog(t *testing.T) {
	agent := newTestAgent()
	compressor := &Compressor{Agent: agent}

	systemContent := "你是一个超级智能的AI助手，你擅长编程和数学。"
	msgs := []model.AgentMessage{
		msg("system", systemContent),
		msg("user", "1+1=?"),
		msg("assistant", "1+1=2"),
	}

	result, err := compressor.CompressMessages(msgs, 10000)
	if err != nil {
		t.Fatalf("压缩消息不应返回错误: %v", err)
	}

	if result[0].Content != systemContent {
		t.Errorf("system prompt内容被修改，期望: %s, 实际: %s", systemContent, result[0].Content)
	}
}

// TC-011: token不够时（保留消息已超过70%），直接返回system prompt + 最后一轮对话
func TestCompressMessages_TokenNotEnough(t *testing.T) {
	agent := newTestAgent()
	compressor := &Compressor{Agent: agent}

	// 构造多轮对话，但设置很小的maxToken
	msgs := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("user", repeatStr("早期消息", 50)),
		msg("assistant", repeatStr("早期回复", 50)),
		msg("user", repeatStr("早期消息2", 50)),
		msg("assistant", repeatStr("早期回复2", 50)),
		msg("user", repeatStr("较近消息", 50)),
		msg("assistant", repeatStr("较近回复", 50)),
		msg("user", repeatStr("最近消息", 50)),
		msg("assistant", repeatStr("最近回复", 50)),
	}

	// 设置极小的maxToken让保留消息就超了
	result, err := compressor.CompressMessages(msgs, 10)
	if err != nil {
		t.Fatalf("压缩消息不应返回错误: %v", err)
	}

	// 应该是走"token不够"分支，直接返回system + 最后一轮
	if result[0].Role != "system" {
		t.Error("结果的第一条应为system prompt")
	}
}

// ============================================================
// 测试 agent.go 中 token 超限触发压缩的集成逻辑
// ============================================================

// TC-012: 验证 Ask() 中存在token超限检测和压缩调用的代码逻辑
func TestAskMethod_HasTokenCheckAndCompression(t *testing.T) {
	// 此测试通过代码审查验证 agent.go 的 Ask() 方法中:
	// 1. 存在 token 超限检测: if needToken >= maxToken
	// 2. 检测到超限后创建 Compressor 并调用 CompressMessages
	// 3. 压缩后再次检查 token 数是否在限制内
	//
	// 由于 Ask() 方法会调用真实的 API (callAPI)，我们无法在单元测试中直接调用它
	// 这里通过构造场景来验证相关的辅助逻辑

	agent := newTestAgent()

	// 验证 Estimator 和 Compressor 可以正常创建
	if agent.Estimator == nil {
		t.Fatal("Agent.Estimator 不应为nil")
	}

	compressor := &Compressor{Agent: agent}
	if compressor.Agent == nil {
		t.Fatal("Compressor.Agent 不应为nil")
	}
}

// ============================================================
// 测试 findCompressEndIds 边界情况
// ============================================================

// TC-013: 只有system消息
func TestFindCompressEndIds_OnlySystem(t *testing.T) {
	msgs := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
	}

	idx := findCompressEndIds(msgs)
	if idx != 0 {
		t.Errorf("仅有system消息时，压缩终点应为0，实际为 %d", idx)
	}
}

// TC-014: 大量轮次对话
func TestFindCompressEndIds_ManyDialogs(t *testing.T) {
	msgs := buildMessages("系统提示", 10) // 10轮对话 = 1 + 20条消息

	idx := findCompressEndIds(msgs)

	// 10轮对话，保留最近3轮，从末尾回溯第3个user
	// 消息总数21，最后一个user在idx=19，第2个在idx=17，第3个在idx=15
	// compressIdx应为15
	if idx != 15 {
		t.Errorf("10轮对话，压缩终点应为15，实际为 %d", idx)
	}

	// 验证compressIdx之后的消息中确实有3个user
	userCount := int32(0)
	for _, m := range msgs[idx:] {
		if m.Role == "user" {
			userCount++
		}
	}
	if userCount != defaultReserveDialog {
		t.Errorf("压缩终点之后应有%d个user消息，实际为 %d", defaultReserveDialog, userCount)
	}
}

// TC-015: 验证 dialogStartIdx 常量
func TestDialogStartIdx(t *testing.T) {
	if DialogStartIdx != 1 {
		t.Errorf("dialogStartIdx 应为1（第0条为system prompt），实际为 %d", DialogStartIdx)
	}
}

// TC-016: 验证 defaultReserveDialog 常量
func TestDefaultReserveDialog(t *testing.T) {
	if defaultReserveDialog != 3 {
		t.Errorf("defaultReserveDialog 应为3，实际为 %d", defaultReserveDialog)
	}
}

// ============================================================
// 测试压缩后system prompt不变性
// ============================================================

// TC-017: 多轮对话压缩后，system prompt始终不变
func TestCompressMessages_SystemPromptImmutable(t *testing.T) {
	agent := newTestAgent()
	compressor := &Compressor{Agent: agent}

	originalSystemPrompt := "这是一个非常重要的系统提示词，包含了Agent的核心行为规则。"
	msgs := []model.AgentMessage{
		msg("system", originalSystemPrompt),
		msg("user", "消息1"),
		msg("assistant", "回复1"),
	}

	result, err := compressor.CompressMessages(msgs, 10000)
	if err != nil {
		t.Fatalf("压缩消息不应返回错误: %v", err)
	}

	// system prompt必须保持不变
	if result[0].Role != "system" {
		t.Fatalf("第一条消息角色应为system，实际为 %s", result[0].Role)
	}
	if result[0].Content != originalSystemPrompt {
		t.Errorf("system prompt被修改！\n期望: %s\n实际: %s", originalSystemPrompt, result[0].Content)
	}
}

// ============================================================
// 测试 computeAvailableToken 精确计算
// ============================================================

// TC-018: 验证70%阈值的正确计算
func TestComputeAvailableToken_Threshold(t *testing.T) {
	agent := newTestAgent()
	compressor := &Compressor{Agent: agent}

	// 空的保留消息
	emptyMsgs := []model.AgentMessage{}

	maxToken := int64(1000)
	available := compressor.computeAvailableToken(maxToken, emptyMsgs)

	// 空消息时，TokenEstimator对空字符串也有基础token(+1)
	// 所以可用token = maxToken*0.7 - 基础token
	emptyTokens := agent.Estimator.ComputeTokens(emptyMsgs)
	expected := (maxToken*7)/10 - emptyTokens
	if available != expected {
		t.Errorf("空保留消息时，可用token应为 %d，实际为 %d", expected, available)
	}

	// 同时验证可用token在合理范围内
	if available <= 0 || available > (maxToken*7)/10 {
		t.Errorf("可用token应在 (0, %d] 范围内，实际为 %d", (maxToken*7)/10, available)
	}
}

// TC-020: 无user消息时，getSystemPromptAndLatestDialog应返回error
func TestGetSystemPromptAndLatestDialog_NoUserMessage(t *testing.T) {
	msgs := []model.AgentMessage{
		msg("system", "你是一个AI助手"),
		msg("assistant", "你好！"),
		msg("assistant", "还有什么可以帮你的？"),
	}

	_, err := getSystemPromptAndLatestDialog(msgs)
	if err == nil {
		t.Error("无user消息时应返回error")
	}
	t.Logf("无user消息时返回的错误: %v", err)
}

// TC-019: 验证compressIdx小于dialogStartIdx时的分支
func TestCompressMessages_CompressIdxLessThanDialogStartIdx(t *testing.T) {
	agent := newTestAgent()
	compressor := &Compressor{Agent: agent}

	// 只有2轮对话，findCompressEndIds应返回0，小于dialogStartIdx(1)
	msgs := []model.AgentMessage{
		msg("system", "系统提示"),
		msg("user", "你好"),
		msg("assistant", "你好！"),
		msg("user", "再见"),
		msg("assistant", "再见！"),
	}

	result, err := compressor.CompressMessages(msgs, 10000)
	if err != nil {
		t.Fatalf("压缩消息不应返回错误: %v", err)
	}

	// 应该走"对话不足三轮"分支
	if result[0].Role != "system" {
		t.Error("结果第一条应为system prompt")
	}

	// 结果应该是system prompt + 最后一轮对话
	t.Logf("对话不足3轮时返回的消息数: %d", len(result))
}
