package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"self-agent/model"
)

const replanPrompt = `你是一个任务规划专家（Task Planner）。
当前执行计划中有子任务失败了，你需要根据当前状态和失败原因，生成一个调整后的新计划。

原始目标：
%s

当前计划状态：
%s

失败的子任务：
- ID: %s
- 描述: %s
- 失败原因: %s

要求：
1. 保留已完成的子任务结果
2. 调整或替换失败的子任务
3. 确保新计划仍然能达成原始目标
4. 子任务数量控制在 2-5 个

请输出严格的 JSON，不要解释。

输出格式：
{
  "sub_tasks": [
    {"id": "1", "description": "子任务描述", "dependencies": [], "status": "completed", "result": "已有结果"},
    {"id": "2", "description": "新的子任务描述", "dependencies": ["1"]}
  ],
  "change_reason": "一句话说明调整原因"
}`

// replan 动态调整计划
func replan(a *Agent, plan *model.Plan, failedTask *model.SubTask, reason string) (*model.Plan, error) {
	// 构建当前计划状态描述
	var statusText string
	for _, t := range plan.SubTasks {
		statusText += fmt.Sprintf("- [%s] %s (状态: %s)\n", t.ID, t.Description, t.Status)
		if t.Result != "" {
			statusText += fmt.Sprintf("  结果: %s\n", t.Result)
		}
	}
	prompt := fmt.Sprintf(replanPrompt, plan.Goal, statusText, failedTask.ID, failedTask.Description, reason)
	resp, err := a.SingleAsk(plan.Goal, prompt)
	if err != nil {
		return nil, fmt.Errorf("Re-planning LLM调用失败: %v", err)
	}
	var replanResp struct {
		SubTasks     []model.SubTask `json:"sub_tasks"`
		ChangeReason string          `json:"change_reason"`
	}
	if err := json.Unmarshal([]byte(resp), &replanResp); err != nil {
		return nil, fmt.Errorf("Re-planning JSON解析失败: %v", err)
	}
	if len(replanResp.SubTasks) == 0 {
		return nil, fmt.Errorf("Re-planning生成的计划为空")
	}
	// 初始化未设置状态的子任务
	for i := range replanResp.SubTasks {
		if replanResp.SubTasks[i].Status == "" {
			replanResp.SubTasks[i].Status = model.SubTaskStatusPending
		}
	}
	newPlan := &model.Plan{
		TaskID:   plan.TaskID,
		Goal:     plan.Goal,
		SubTasks: replanResp.SubTasks,
		Version:  plan.Version + 1,
	}
	log.Printf("[Re-planning] 计划调整为v%d，原因: %s", newPlan.Version, replanResp.ChangeReason)
	return newPlan, nil
}
