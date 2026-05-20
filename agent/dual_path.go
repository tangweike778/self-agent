package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"self-agent/model"
	"sync"
	"time"
)

const resultEvaluatorPrompt = `你是一个专业的结果评审专家（Result Evaluator）。
现在有同一个用户问题，Agent 通过不同路径得到了多个结果。
请你从以下维度进行对比，并给出最终推荐结果。

用户问题：
%s

候选结果：
%s

评审维度：
1. 准确性（是否回答问题）
2. 完整性（是否覆盖关键点）
3. 逻辑性（推理是否清晰）
4. 可执行性（是否可直接使用）
5. 冗余度（是否啰嗦）

输出格式（严格 JSON）：
{
  "best_result_id": "result_1" | "result_2",
  "ranking": [
    {"id": "result_2", "score": 9},
    {"id": "result_1", "score": 7}
  ],
  "comparison_summary": "简要对比说明",
  "final_answer": "推荐给用户的最终答案"
}`

const dualPathTimeout = 60 * time.Second

type pathResult struct {
	ID     string
	Result string
	Err    error
}

// ExecuteDualPath 双路径并行执行与结果对比
func ExecuteDualPath(a *Agent, input string, history []model.AgentMessage) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dualPathTimeout)
	defer cancel()
	var wg sync.WaitGroup
	results := make(chan pathResult, 2)
	// 启动 ReAct 路径
	wg.Add(1)
	go func() {
		defer wg.Done()
		reactHistory := make([]model.AgentMessage, len(history))
		copy(reactHistory, history)
		reactHistory = append(reactHistory, model.AgentMessage{
			Role:    "user",
			Content: input,
		})
		msgs, err := a.Ask(reactHistory)
		select {
		case <-ctx.Done():
			return
		default:
			if err != nil {
				results <- pathResult{ID: "result_react", Err: err}
			} else {
				results <- pathResult{ID: "result_react", Result: getLastMsg(msgs)}
			}
		}
	}()
	// 启动 Plan-and-Solve 路径
	wg.Add(1)
	go func() {
		defer wg.Done()
		plan, err := GeneratePlan(a, input, history)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				results <- pathResult{ID: "result_plan", Err: err}
			}
			return
		}
		answer, err := ExecutePlan(a, plan, history, nil)
		select {
		case <-ctx.Done():
			return
		default:
			if err != nil {
				results <- pathResult{ID: "result_plan", Err: err}
			} else {
				results <- pathResult{ID: "result_plan", Result: answer}
			}
		}
	}()
	// 等待所有路径完成或超时
	go func() {
		wg.Wait()
		close(results)
	}()
	var collected []pathResult
	for r := range results {
		collected = append(collected, r)
	}
	// 筛选成功的结果
	var successResults []pathResult
	for _, r := range collected {
		if r.Err == nil {
			successResults = append(successResults, r)
		}
	}
	// 两条路径均失败
	if len(successResults) == 0 {
		return "", fmt.Errorf("双路径执行均失败")
	}
	// 只有一条路径成功
	if len(successResults) == 1 {
		log.Printf("[双路径] 仅一条路径成功: %s", successResults[0].ID)
		return successResults[0].Result, nil
	}
	// 两条路径均成功，调用 LLM 进行结果对比
	return compareResults(a, input, successResults)
}

// compareResults 调用 LLM 对比两条路径的结果
func compareResults(a *Agent, query string, results []pathResult) (string, error) {
	type candidateResult struct {
		ID     string `json:"id"`
		Result string `json:"result"`
	}
	candidates := make([]candidateResult, len(results))
	for i, r := range results {
		candidates[i] = candidateResult{ID: r.ID, Result: r.Result}
	}
	candidatesJSON, _ := json.Marshal(candidates)
	prompt := fmt.Sprintf(resultEvaluatorPrompt, query, string(candidatesJSON))
	resp, err := a.SingleAsk(query, prompt)
	if err != nil {
		// 对比失败时返回第一个成功结果
		log.Printf("[双路径] 结果对比LLM调用失败: %v，返回第一个结果", err)
		return results[0].Result, nil
	}
	var evalResult struct {
		BestResultID      string `json:"best_result_id"`
		ComparisonSummary string `json:"comparison_summary"`
		FinalAnswer       string `json:"final_answer"`
	}
	if err := json.Unmarshal([]byte(resp), &evalResult); err != nil {
		log.Printf("[双路径] 结果对比JSON解析失败: %v，返回第一个结果", err)
		return results[0].Result, nil
	}
	log.Printf("[双路径] 最优结果: %s, 对比说明: %s", evalResult.BestResultID, evalResult.ComparisonSummary)
	if evalResult.FinalAnswer != "" {
		return evalResult.FinalAnswer, nil
	}
	// 根据 best_result_id 返回对应结果
	for _, r := range results {
		if r.ID == evalResult.BestResultID {
			return r.Result, nil
		}
	}
	return results[0].Result, nil
}
