package model

import "fmt"

// 子任务状态常量
const (
	SubTaskStatusPending   = "pending"
	SubTaskStatusRunning   = "running"
	SubTaskStatusCompleted = "completed"
	SubTaskStatusFailed    = "failed"
)

// 策略模式常量
const (
	StrategyModeReact     = "react"
	StrategyModePlanSolve = "plan_and_solve"
)

// SubTask 子任务
type SubTask struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	Status       string   `json:"status"`
	Result       string   `json:"result"`
}

// Plan 执行计划
type Plan struct {
	TaskID   string    `json:"task_id"`
	Goal     string    `json:"goal"`
	SubTasks []SubTask `json:"sub_tasks"`
	Version  int       `json:"version"`
}

// StrategyResult 策略判断结果
type StrategyResult struct {
	Mode       string  `json:"mode"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

// Progress 返回当前计划的进度信息
func (p *Plan) Progress() string {
	completed := 0
	for _, t := range p.SubTasks {
		if t.Status == SubTaskStatusCompleted {
			completed++
		}
	}
	return fmt.Sprintf("已完成 %d/%d 步骤", completed, len(p.SubTasks))
}

// UpdateSubTaskStatus 更新子任务状态
func (p *Plan) UpdateSubTaskStatus(taskID string, status string) {
	for i := range p.SubTasks {
		if p.SubTasks[i].ID == taskID {
			p.SubTasks[i].Status = status
			return
		}
	}
}

// UpdateSubTaskResult 更新子任务结果
func (p *Plan) UpdateSubTaskResult(taskID string, result string) {
	for i := range p.SubTasks {
		if p.SubTasks[i].ID == taskID {
			p.SubTasks[i].Result = result
			return
		}
	}
}

// GetSubTask 获取子任务
func (p *Plan) GetSubTask(taskID string) *SubTask {
	for i := range p.SubTasks {
		if p.SubTasks[i].ID == taskID {
			return &p.SubTasks[i]
		}
	}
	return nil
}
