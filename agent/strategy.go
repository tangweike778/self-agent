package agent

import (
	"log"
	"self-agent/model"
	"strings"
)

// SelectStrategy 策略路由总控函数
// 按优先级调用：空输入检查 → 正则匹配层 → LLM 决策层
func SelectStrategy(a *Agent, input string) *model.StrategyResult {
	// 空输入检查
	if strings.TrimSpace(input) == "" {
		return defaultReactResult()
	}
	// 正则匹配层
	if mode, matched := regexMatch(input); matched {
		log.Printf("[策略路由] 正则匹配命中: mode=%s", mode)
		return &model.StrategyResult{
			Mode:       mode,
			Confidence: 1.0,
			Reason:     "正则匹配层命中",
		}
	}
	// LLM 决策层
	result, err := llmDecision(a, input)
	if err != nil {
		log.Printf("[策略路由] LLM决策层出错: %v，回退到ReAct", err)
		return defaultReactResult()
	}
	return result
}
