package agent

import (
	"encoding/json"
	"log"
	"self-agent/model"
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
