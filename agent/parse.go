package agent

import (
	"regexp"
	"strings"
)

// ParsedMessage 解析LLM输出消息
type ParsedMessage struct {
	Thought    string
	ActionPlan string
	Reflection string
	Content    string
}

// 正则匹配 LLM 输出中的标签
var (
	ThoughtRegex    = regexp.MustCompile(`(?s)<thought>(.*?)</thought>`)
	reflectionRegex = regexp.MustCompile(`(?s)<reflection>(.*?)</reflection>`)
	actionPlanRegex = regexp.MustCompile(`(?s)<action_plan>(.*?)</action_plan>`)
	tagRegex        = regexp.MustCompile(`<.*?>`)
	errorRegex      = regexp.MustCompile(`(?i)(错误|故障|异常|失败|error|exception|fail|unknown|unexpected|致命|崩溃|crash|panic)`)
)

func ParseLLMOutput(content string) *ParsedMessage {
	result := &ParsedMessage{}
	// 提取 <thought>
	if matches := ThoughtRegex.FindStringSubmatch(content); len(matches) > 1 {
		result.Thought = strings.TrimSpace(matches[1])
	}

	// 提取 <reflection>
	if matches := reflectionRegex.FindStringSubmatch(content); len(matches) > 1 {
		result.Reflection = strings.TrimSpace(matches[1])
	}

	// 提取 <action_plan>
	if matches := actionPlanRegex.FindStringSubmatch(content); len(matches) > 1 {
		result.ActionPlan = strings.TrimSpace(matches[1])
	}
	// 去除所有标签，得到纯内容
	result.Content = stripTags(content)
	return result
}

// stripTags 去除所有标签
func stripTags(content string) string {
	return tagRegex.ReplaceAllString(content, "")
}

func isToolError(result string) bool {
	return errorRegex.MatchString(result)
}
