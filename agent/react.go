package agent

import (
	"fmt"
	"time"
)

// ReActStep 执行步骤
type ReActStep struct {
	StepNum     int           // 执行步骤
	Thought     string        // 模型的思考
	Action      string        // 模型的执行
	ToolCalls   []string      // 工具调用名称
	Observation string        // 观察结果
	Reflection  string        // 反思结果
	Success     bool          // 本次调用是否成功
	Duration    time.Duration // 执行时间
}

// ReActTrace 执行跟踪
type ReActTrace struct {
	Question  string        // 用户问题
	Steps     []ReActStep   // 执行步骤
	Answer    string        // 最终回答
	TotalTime time.Duration // 总耗时
}

// AddStep 添加步骤
func (r *ReActTrace) AddStep(step ReActStep) {
	r.Steps = append(r.Steps, step)
}

// GetLog 获取日志
func (r *ReActTrace) GetLog() string {
	log := fmt.Sprintf("Question: %s\n", r.Question)
	for _, step := range r.Steps {
		log += fmt.Sprintf("Step %d: %s\n", step.StepNum, step.Thought)
		log += fmt.Sprintf("Action: %s\n", step.Action)
		log += fmt.Sprintf("Tool Calls: %v\n", step.ToolCalls)
		log += fmt.Sprintf("Observation: %s\n", step.Observation)
		log += fmt.Sprintf("Reflection: %s\n", step.Reflection)
		log += fmt.Sprintf("Success: %t\n", step.Success)
		log += fmt.Sprintf("Duration: %s\n", step.Duration)
	}
	log += fmt.Sprintf("Answer: %s\n", r.Answer)
	log += fmt.Sprintf("Total Time: %s\n", r.TotalTime)
	return log
}
