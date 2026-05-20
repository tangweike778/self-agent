package agent

import (
	"encoding/json"
	"log"
	"regexp"
	"self-agent/model"
	"strings"
	"unicode/utf8"
)

const taskRouterPrompt = `你是一个任务调度专家（Task Router）。
你的任务是判断：给定用户请求，应该使用哪种执行模式。

可选模式：
1. **ReAct**：逐步推理 + 行动，适合简单、探索性、交互式任务。
2. **Plan-and-Solve**：先规划再执行，适合复杂、多步骤、可预测任务。

判断标准：
- 如果任务 **步骤少、目标模糊、需要边试边改** → ReAct
- 如果任务 **步骤多、目标明确、可提前规划** → Plan-and-Solve

请输出严格的 JSON，不要解释。

输出格式：
{
  "mode": "react" | "plan_and_solve",
  "confidence": 0.0~1.0,
  "reason": "一句话说明理由"
}`

// Plan-and-Solve 模式的正则规则
var planSolvePatterns = []*regexp.Regexp{
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

// SelectStrategy 策略路由总控函数
// 按优先级调用：空输入检查 → 正则匹配层 → LLM 决策层
func SelectStrategy(a *Agent, input string) *model.StrategyResult {
	if strings.TrimSpace(input) == "" {
		return defaultReactResult()
	}
	if mode, matched := regexMatch(input); matched {
		log.Printf("[策略路由] 正则匹配命中: mode=%s", mode)
		return &model.StrategyResult{
			Mode:       mode,
			Confidence: 1.0,
			Reason:     "正则匹配层命中",
		}
	}
	result, err := llmDecision(a, input)
	if err != nil {
		log.Printf("[策略路由] LLM决策层出错: %v，回退到ReAct", err)
		return defaultReactResult()
	}
	return result
}

// containsPlanKeywords 判断输入是否同时包含"先"、"再"、"然后"三个关键词
func containsPlanKeywords(input string) bool {
	return strings.Contains(input, "先") && strings.Contains(input, "再") && strings.Contains(input, "然后")
}

// regexMatch 通过正则匹配判断策略模式
func regexMatch(input string) (string, bool) {
	charCount := utf8.RuneCountInString(input)
	// 超长输入直接走 Plan-and-Solve
	if charCount > 200 {
		return model.StrategyModePlanSolve, true
	}
	// ReAct 关键词匹配（字数 < 50）
	if charCount < 50 {
		for _, pattern := range reactPatterns {
			if pattern.MatchString(input) {
				return model.StrategyModeReact, true
			}
		}
	}
	// Plan-and-Solve 关键词匹配（无字数限制）
	if containsPlanKeywords(input) {
		return model.StrategyModePlanSolve, true
	}
	for _, pattern := range planSolvePatterns {
		if pattern.MatchString(input) {
			return model.StrategyModePlanSolve, true
		}
	}
	return "", false
}

// llmDecision 通过 LLM 判断策略模式
func llmDecision(a *Agent, input string) (*model.StrategyResult, error) {
	resp, err := a.SingleAsk(input, taskRouterPrompt)
	if err != nil {
		log.Printf("[策略路由] LLM决策调用失败: %v，回退到ReAct", err)
		return defaultReactResult(), nil
	}
	var result model.StrategyResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		log.Printf("[策略路由] LLM返回JSON解析失败: %v，回退到ReAct", err)
		return defaultReactResult(), nil
	}
	log.Printf("[策略路由] LLM决策: mode=%s, confidence=%.2f, reason=%s", result.Mode, result.Confidence, result.Reason)
	return &result, nil
}

// defaultReactResult 返回默认的 ReAct 策略结果
func defaultReactResult() *model.StrategyResult {
	return &model.StrategyResult{
		Mode:       model.StrategyModeReact,
		Confidence: 1.0,
		Reason:     "默认回退到ReAct模式",
	}
}
