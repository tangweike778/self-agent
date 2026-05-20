package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"self-agent/model"
)

const planGeneratePrompt = `你是一个任务规划专家（Task Planner）。
你的任务是将用户的复杂请求分解为多个可执行的子任务，并明确它们之间的依赖关系。

要求：
1. 每个子任务应该是独立可执行的
2. 子任务数量控制在 2-5 个
3. 明确标注子任务之间的依赖关系（用 ID 引用）
4. 没有依赖的子任务 dependencies 为空数组

请输出严格的 JSON，不要解释。

用户请求：
%s

输出格式：
{
  "sub_tasks": [
    {"id": "1", "description": "子任务描述", "dependencies": []},
    {"id": "2", "description": "子任务描述", "dependencies": ["1"]}
  ]
}`

const planSummaryPrompt = `你是一个结果汇总专家。请根据以下子任务的执行结果，生成一个完整、连贯的最终答案。

用户原始问题：
%s

各子任务执行结果：
%s

请直接输出最终答案，不要解释你的汇总过程。`

// GeneratePlan 调用 LLM 生成结构化执行计划
func GeneratePlan(a *Agent, goal string, history []model.AgentMessage) (*model.Plan, error) {
	prompt := fmt.Sprintf(planGeneratePrompt, goal)
	resp, err := a.SingleAsk(goal, prompt)
	if err != nil {
		return nil, fmt.Errorf("生成计划失败: %v", err)
	}
	var planResp struct {
		SubTasks []model.SubTask `json:"sub_tasks"`
	}
	if err := json.Unmarshal([]byte(resp), &planResp); err != nil {
		return nil, fmt.Errorf("解析计划JSON失败: %v", err)
	}
	if len(planResp.SubTasks) == 0 {
		return nil, fmt.Errorf("生成的计划为空")
	}
	// 初始化子任务状态
	for i := range planResp.SubTasks {
		planResp.SubTasks[i].Status = model.SubTaskStatusPending
	}
	plan := &model.Plan{
		TaskID:   fmt.Sprintf("plan_%d", len(history)),
		Goal:     goal,
		SubTasks: planResp.SubTasks,
		Version:  1,
	}
	log.Printf("[Plan-and-Solve] 生成计划: %d个子任务", len(plan.SubTasks))
	return plan, nil
}

// buildDAG 将子任务构建为 DAG 并进行拓扑排序，返回按层级分组的任务ID
func buildDAG(plan *model.Plan) ([][]string, error) {
	taskMap := make(map[string]*model.SubTask)
	inDegree := make(map[string]int)
	graph := make(map[string][]string)
	for i := range plan.SubTasks {
		task := &plan.SubTasks[i]
		taskMap[task.ID] = task
		inDegree[task.ID] = 0
	}
	// 构建邻接表和入度
	for _, task := range plan.SubTasks {
		for _, dep := range task.Dependencies {
			if _, exists := taskMap[dep]; !exists {
				return nil, fmt.Errorf("子任务 %s 依赖不存在的任务 %s", task.ID, dep)
			}
			graph[dep] = append(graph[dep], task.ID)
			inDegree[task.ID]++
		}
	}
	// 拓扑排序（Kahn算法），按层级分组
	var levels [][]string
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}
	processed := 0
	for len(queue) > 0 {
		levels = append(levels, queue)
		var nextQueue []string
		for _, id := range queue {
			processed++
			for _, next := range graph[id] {
				inDegree[next]--
				if inDegree[next] == 0 {
					nextQueue = append(nextQueue, next)
				}
			}
		}
		queue = nextQueue
	}
	// 检测循环依赖
	if processed != len(plan.SubTasks) {
		return nil, fmt.Errorf("检测到循环依赖，无法执行计划")
	}
	return levels, nil
}

// ExecutePlan 按拓扑序执行子任务
func ExecutePlan(a *Agent, plan *model.Plan, history []model.AgentMessage, progressFn func(string)) (string, error) {
	levels, err := buildDAG(plan)
	if err != nil {
		return "", err
	}
	replanCount := 0
	maxReplan := 3
	for _, level := range levels {
		for _, taskID := range level {
			task := plan.GetSubTask(taskID)
			if task == nil {
				continue
			}
			// 更新状态为 running
			plan.UpdateSubTaskStatus(taskID, model.SubTaskStatusRunning)
			if progressFn != nil {
				progressFn(plan.Progress())
			}
			// 构建子任务上下文：注入已完成任务的结果
			taskHistory := buildSubTaskContext(plan, task, history)
			// 执行子任务（复用 ReAct 能力）
			resultMsgs, err := a.Ask(taskHistory)
			if err != nil {
				plan.UpdateSubTaskStatus(taskID, model.SubTaskStatusFailed)
				// 触发 Re-planning
				if replanCount >= maxReplan {
					return "", fmt.Errorf("子任务 %s 执行失败且Re-planning次数超过上限: %v", taskID, err)
				}
				newPlan, replanErr := replan(a, plan, task, err.Error())
				if replanErr != nil {
					return "", fmt.Errorf("Re-planning失败: %v", replanErr)
				}
				replanCount++
				plan = newPlan
				// 重新构建 DAG 并执行
				return ExecutePlan(a, plan, history, progressFn)
			}
			// 提取最终结果
			result := getLastMsg(resultMsgs)
			plan.UpdateSubTaskResult(taskID, result)
			plan.UpdateSubTaskStatus(taskID, model.SubTaskStatusCompleted)
			if progressFn != nil {
				progressFn(plan.Progress())
			}
			log.Printf("[Plan-and-Solve] 子任务 %s 完成", taskID)
		}
	}
	// 汇总结果生成最终答案
	return summarizeResults(a, plan)
}

// buildSubTaskContext 构建子任务的执行上下文
func buildSubTaskContext(plan *model.Plan, task *model.SubTask, history []model.AgentMessage) []model.AgentMessage {
	// 复制系统提示词
	var taskHistory []model.AgentMessage
	if len(history) > 0 && history[0].Role == "system" {
		taskHistory = append(taskHistory, history[0])
	}
	// 注入已完成依赖任务的结果
	var context string
	for _, depID := range task.Dependencies {
		depTask := plan.GetSubTask(depID)
		if depTask != nil && depTask.Status == model.SubTaskStatusCompleted {
			context += fmt.Sprintf("【前置任务 %s 的结果】：%s\n\n", depTask.Description, depTask.Result)
		}
	}
	// 构建用户消息
	userMsg := task.Description
	if context != "" {
		userMsg = fmt.Sprintf("背景信息：\n%s\n当前任务：%s", context, task.Description)
	}
	taskHistory = append(taskHistory, model.AgentMessage{
		Role:    "user",
		Content: userMsg,
	})
	return taskHistory
}

// summarizeResults 汇总所有子任务结果生成最终答案
func summarizeResults(a *Agent, plan *model.Plan) (string, error) {
	var resultsText string
	for _, task := range plan.SubTasks {
		resultsText += fmt.Sprintf("子任务[%s]: %s\n结果: %s\n\n", task.ID, task.Description, task.Result)
	}
	prompt := fmt.Sprintf(planSummaryPrompt, plan.Goal, resultsText)
	answer, err := a.SingleAsk(plan.Goal, prompt)
	if err != nil {
		// 汇总失败时，直接拼接结果
		log.Printf("[Plan-and-Solve] 汇总失败: %v，直接拼接结果", err)
		return resultsText, nil
	}
	return answer, nil
}

// getLastMsg 获取消息列表中最后一条消息的内容（planner内部使用）
func getLastMsg(messages []model.AgentMessage) string {
	if len(messages) == 0 {
		return ""
	}
	return messages[len(messages)-1].Content
}
