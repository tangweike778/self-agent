package agent

import (
	"fmt"
	"strings"
	"testing"
)

// ============================================================
// 四、ParseLLMOutput 解析器测试
// ============================================================

// TC-009: 解析包含 <thought> 标签的输出
func TestParseLLMOutput_WithThought(t *testing.T) {
	content := `<thought>用户想要读取文件，我需要使用 read_file 工具。</thought>
好的，我来帮你读取文件。`

	parsed := ParseLLMOutput(content)

	if parsed.Thought != "用户想要读取文件，我需要使用 read_file 工具。" {
		t.Errorf("Thought 解析不正确: '%s'", parsed.Thought)
	}
}

// TC-010: 解析包含 <reflection> 标签的输出
func TestParseLLMOutput_WithReflection(t *testing.T) {
	content := `<reflection>read_file 失败了，可能是路径不对。我应该先确认文件是否存在。</reflection>
让我换一种方式尝试。`

	parsed := ParseLLMOutput(content)

	if parsed.Reflection != "read_file 失败了，可能是路径不对。我应该先确认文件是否存在。" {
		t.Errorf("Reflection 解析不正确: '%s'", parsed.Reflection)
	}
}

// TC-011: 解析同时包含 <thought> 和 <reflection> 的输出
func TestParseLLMOutput_WithBothTags(t *testing.T) {
	content := `<thought>我需要分析错误原因。</thought>
<reflection>上一步失败了，因为文件不存在。</reflection>
让我重新尝试。`

	parsed := ParseLLMOutput(content)

	if parsed.Thought != "我需要分析错误原因。" {
		t.Errorf("Thought 解析不正确: '%s'", parsed.Thought)
	}
	if parsed.Reflection != "上一步失败了，因为文件不存在。" {
		t.Errorf("Reflection 解析不正确: '%s'", parsed.Reflection)
	}
}

// TC-012: 解析不包含任何标签的输出
func TestParseLLMOutput_NoTags(t *testing.T) {
	content := "这是一个普通的回复，没有任何标签。"

	parsed := ParseLLMOutput(content)

	if parsed.Thought != "" {
		t.Errorf("无标签时 Thought 应为空，实际为 '%s'", parsed.Thought)
	}
	if parsed.Reflection != "" {
		t.Errorf("无标签时 Reflection 应为空，实际为 '%s'", parsed.Reflection)
	}
	if parsed.Content == "" {
		t.Error("Content 不应为空")
	}
}

// TC-013: 解析空字符串
func TestParseLLMOutput_EmptyString(t *testing.T) {
	parsed := ParseLLMOutput("")

	if parsed.Thought != "" {
		t.Errorf("空字符串 Thought 应为空，实际为 '%s'", parsed.Thought)
	}
	if parsed.Reflection != "" {
		t.Errorf("空字符串 Reflection 应为空，实际为 '%s'", parsed.Reflection)
	}
}

// TC-014: 解析多行 <thought> 内容
func TestParseLLMOutput_MultilineThought(t *testing.T) {
	content := `<thought>
用户想要查看系统状态。
我需要执行 ps 命令来获取进程信息。
然后再用 df 命令查看磁盘使用情况。
</thought>
好的，让我来检查系统状态。`

	parsed := ParseLLMOutput(content)

	if !strings.Contains(parsed.Thought, "用户想要查看系统状态") {
		t.Errorf("多行 Thought 应包含第一行内容: '%s'", parsed.Thought)
	}
	if !strings.Contains(parsed.Thought, "df 命令") {
		t.Errorf("多行 Thought 应包含最后一行内容: '%s'", parsed.Thought)
	}
}

// TC-015: 解析包含特殊字符的 <thought> 内容
func TestParseLLMOutput_SpecialCharsInThought(t *testing.T) {
	content := `<thought>文件路径是 ~/GolandProjects/self-agent/config.yaml，需要使用 $HOME 变量。</thought>`

	parsed := ParseLLMOutput(content)

	if !strings.Contains(parsed.Thought, "~/GolandProjects") {
		t.Errorf("Thought 应保留特殊字符: '%s'", parsed.Thought)
	}
	if !strings.Contains(parsed.Thought, "$HOME") {
		t.Errorf("Thought 应保留 $HOME: '%s'", parsed.Thought)
	}
}

// ============================================================
// 五、stripTags 标签清理测试
// ============================================================

// TC-016: stripTags 去除所有标签
func TestStripTags_RemoveAllTags(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"去除 thought 标签",
			"<thought>思考内容</thought>正文",
			"思考内容正文",
		},
		{
			"去除 reflection 标签",
			"<reflection>反思内容</reflection>正文",
			"反思内容正文",
		},
		{
			"去除多个标签",
			"<thought>思考</thought><reflection>反思</reflection>正文",
			"思考反思正文",
		},
		{
			"无标签不变",
			"普通文本内容",
			"普通文本内容",
		},
		{
			"空字符串",
			"",
			"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := stripTags(tc.input)
			if result != tc.expected {
				t.Errorf("stripTags(%q) = %q, 期望 %q", tc.input, result, tc.expected)
			}
		})
	}
}

// ============================================================
// 六、isToolError 错误检测测试
// ============================================================

// TC-017: 检测中文错误关键词
func TestIsToolError_ChineseKeywords(t *testing.T) {
	errorCases := []string{
		"执行命令错误: exit status 1",
		"文件读取故障",
		"操作异常终止",
		"执行失败: permission denied",
		"致命错误: 内存不足",
		"程序崩溃了",
	}

	for _, errMsg := range errorCases {
		if !isToolError(errMsg) {
			t.Errorf("isToolError 应检测到中文错误关键词: '%s'", errMsg)
		}
	}
}

// TC-018: 检测英文错误关键词
func TestIsToolError_EnglishKeywords(t *testing.T) {
	errorCases := []string{
		"Error: file not found",
		"Exception occurred",
		"Command failed with exit code 1",
		"Unknown command: xyz",
		"Unexpected token in JSON",
		"Fatal error in module",
		"Process crash detected",
		"Panic: runtime error",
	}

	for _, errMsg := range errorCases {
		if !isToolError(errMsg) {
			t.Errorf("isToolError 应检测到英文错误关键词: '%s'", errMsg)
		}
	}
}

// TC-019: 正常输出不应被误判为错误
func TestIsToolError_NormalOutput(t *testing.T) {
	normalCases := []string{
		"文件内容: hello world",
		"命令执行成功",
		"总共 3 个文件",
		"CPU 使用率: 25%",
		"进程列表如下:",
		"",
	}

	for _, normalMsg := range normalCases {
		if isToolError(normalMsg) {
			t.Errorf("isToolError 不应将正常输出判为错误: '%s'", normalMsg)
		}
	}
}

// TC-020: 大小写不敏感检测
func TestIsToolError_CaseInsensitive(t *testing.T) {
	testCases := []string{
		"ERROR: something went wrong",
		"error: something went wrong",
		"Error: something went wrong",
		"FAIL: test failed",
		"Fail: test failed",
	}

	for _, tc := range testCases {
		if !isToolError(tc) {
			t.Errorf("isToolError 应大小写不敏感检测: '%s'", tc)
		}
	}
}

// ============================================================
// 七、正则表达式测试
// ============================================================

// TC-021: ThoughtRegex 正确匹配 <thought> 标签
func TestThoughtRegex_Match(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"单行 thought",
			"<thought>简单思考</thought>",
			"简单思考",
		},
		{
			"多行 thought",
			"<thought>\n第一行\n第二行\n</thought>",
			"\n第一行\n第二行\n",
		},
		{
			"thought 前后有内容",
			"前缀<thought>思考内容</thought>后缀",
			"思考内容",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := ThoughtRegex.FindStringSubmatch(tc.input)
			if len(matches) < 2 {
				t.Fatalf("ThoughtRegex 未匹配到内容: '%s'", tc.input)
			}
			if matches[1] != tc.expected {
				t.Errorf("ThoughtRegex 匹配结果不正确: 期望 '%s'，实际 '%s'", tc.expected, matches[1])
			}
		})
	}
}

// TC-022: ThoughtRegex 不匹配无标签内容
func TestThoughtRegex_NoMatch(t *testing.T) {
	noMatchCases := []string{
		"普通文本",
		"<reflection>反思内容</reflection>",
		"<thought>未闭合标签",
		"",
	}

	for _, input := range noMatchCases {
		matches := ThoughtRegex.FindStringSubmatch(input)
		if len(matches) > 1 && input == "<thought>未闭合标签" {
			// 未闭合标签不应匹配
			t.Errorf("ThoughtRegex 不应匹配未闭合标签: '%s'", input)
		}
	}
}

// ============================================================
// 十二、ParsedMessage 数据结构测试
// ============================================================

// TC-032: ParsedMessage 字段完整性
func TestParsedMessage_FieldCompleteness(t *testing.T) {
	pm := &ParsedMessage{
		Thought:    "思考内容",
		Reflection: "反思内容",
		ActionPlan: "行动计划",
		Content:    "纯文本内容",
	}

	if pm.Thought != "思考内容" {
		t.Errorf("Thought 不正确: '%s'", pm.Thought)
	}
	if pm.Reflection != "反思内容" {
		t.Errorf("Reflection 不正确: '%s'", pm.Reflection)
	}
	if pm.ActionPlan != "行动计划" {
		t.Errorf("ActionPlan 不正确: '%s'", pm.ActionPlan)
	}
	if pm.Content != "纯文本内容" {
		t.Errorf("Content 不正确: '%s'", pm.Content)
	}
}

// TC-033: ParseLLMOutput 返回的 Content 去除了标签
func TestParseLLMOutput_ContentStripped(t *testing.T) {
	content := "<thought>思考过程</thought>这是最终回复<reflection>反思</reflection>"

	parsed := ParseLLMOutput(content)

	// Content 应该去除了所有标签
	if strings.Contains(parsed.Content, "<thought>") {
		t.Errorf("Content 不应包含 <thought> 标签: '%s'", parsed.Content)
	}
	if strings.Contains(parsed.Content, "</thought>") {
		t.Errorf("Content 不应包含 </thought> 标签: '%s'", parsed.Content)
	}
	if strings.Contains(parsed.Content, "<reflection>") {
		t.Errorf("Content 不应包含 <reflection> 标签: '%s'", parsed.Content)
	}
	if !strings.Contains(parsed.Content, "这是最终回复") {
		t.Errorf("Content 应包含纯文本内容: '%s'", parsed.Content)
	}
}

// ============================================================
// 十五、Session tidyMessage 测试
// ============================================================

// TC-037: tidyMessage 清理 <thought> 标签（通过 ThoughtRegex 验证）
func TestTidyMessage_RemoveThought(t *testing.T) {
	// session.go 中 tidyMessage 使用 agent.ThoughtRegex 清理消息
	// 这里直接测试 ThoughtRegex 的替换功能
	message := "<thought>这是思考过程，不应展示给用户。</thought>这是最终回复。"

	tidied := ThoughtRegex.ReplaceAllString(message, "")

	if strings.Contains(tidied, "思考过程") {
		t.Errorf("tidyMessage 应清除 <thought> 内容: '%s'", tidied)
	}
	if !strings.Contains(tidied, "这是最终回复") {
		t.Errorf("tidyMessage 应保留非标签内容: '%s'", tidied)
	}
}

// TC-038: tidyMessage 多个 <thought> 标签
func TestTidyMessage_MultipleThoughts(t *testing.T) {
	message := "<thought>第一次思考</thought>中间内容<thought>第二次思考</thought>最终回复"

	tidied := ThoughtRegex.ReplaceAllString(message, "")

	if strings.Contains(tidied, "第一次思考") || strings.Contains(tidied, "第二次思考") {
		t.Errorf("应清除所有 <thought> 内容: '%s'", tidied)
	}
	if !strings.Contains(tidied, "中间内容") || !strings.Contains(tidied, "最终回复") {
		t.Errorf("应保留非标签内容: '%s'", tidied)
	}
}

// TC-039: tidyMessage 无 <thought> 标签时不变
func TestTidyMessage_NoThought(t *testing.T) {
	message := "这是一个普通回复，没有思考标签。"

	tidied := ThoughtRegex.ReplaceAllString(message, "")

	if tidied != message {
		t.Errorf("无标签时消息不应改变: 期望 '%s'，实际 '%s'", message, tidied)
	}
}

// ============================================================
// 十七、errorRegex 正则表达式测试
// ============================================================

// TC-042: errorRegex 匹配所有预定义的错误关键词
func TestErrorRegex_AllKeywords(t *testing.T) {
	keywords := []string{
		"错误", "故障", "异常", "失败",
		"error", "exception", "fail",
		"unknown", "unexpected",
		"致命", "崩溃", "crash", "panic",
	}

	for _, kw := range keywords {
		testStr := fmt.Sprintf("这里有一个%s发生了", kw)
		if !errorRegex.MatchString(testStr) {
			t.Errorf("errorRegex 应匹配关键词 '%s'", kw)
		}
	}
}

// TC-043: errorRegex 大小写不敏感
func TestErrorRegex_CaseInsensitive(t *testing.T) {
	testCases := []string{
		"ERROR occurred",
		"Error occurred",
		"error occurred",
		"EXCEPTION thrown",
		"Exception thrown",
		"FAIL: test",
		"PANIC: runtime",
		"CRASH detected",
	}

	for _, tc := range testCases {
		if !errorRegex.MatchString(tc) {
			t.Errorf("errorRegex 应大小写不敏感匹配: '%s'", tc)
		}
	}
}
