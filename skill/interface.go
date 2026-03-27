package skill

import "self-agent/model"

// Skill 技能
type Skill interface {
	// Name 工具名称（唯一标识）
	Name() string
	// Description 返回 LLM 可理解的工具定义
	Description() model.ToolDefinition
	// Execute 执行工具，接收 JSON 参数字符串，返回结果字符串
	Execute(argsJSON string) string
}
