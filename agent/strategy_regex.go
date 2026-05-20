package agent

import (
	"regexp"
	"self-agent/model"
	"unicode/utf8"
)

// Plan-and-Solve 模式的正则规则
var planSolvePatterns = []*regexp.Regexp{
	regexp.MustCompile(`先.+再.+然后`),
	regexp.MustCompile(`步骤`),
	regexp.MustCompile(`流程`),
	regexp.MustCompile(`一步步`),
	regexp.MustCompile(`生成`),
}

// ReAct 模式的正则规则
var reactPatterns = []*regexp.Regexp{
	regexp.MustCompile(`查一下`),
	regexp.MustCompile(`看看`),
}

// regexMatch 通过正则匹配判断策略模式
// 返回值：mode 策略模式，matched 是否命中
func regexMatch(input string) (string, bool) {
	charCount := utf8.RuneCountInString(input)
	// 超长输入直接走 ReAct
	if charCount > 200 {
		return model.StrategyModeReact, true
	}
	// ReAct 关键词匹配
	for _, pattern := range reactPatterns {
		if pattern.MatchString(input) {
			return model.StrategyModeReact, true
		}
	}
	// Plan-and-Solve 关键词匹配（字数 < 50）
	if charCount < 50 {
		for _, pattern := range planSolvePatterns {
			if pattern.MatchString(input) {
				return model.StrategyModePlanSolve, true
			}
		}
	}
	return "", false
}
