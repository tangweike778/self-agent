package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"self-agent/model"
)

// ReadFileSkill 读取文件技能
type ReadFileSkill struct{}

// Name 技能名称
func (rf *ReadFileSkill) Name() string {
	return "read_file"
}

// Description 技能描述
func (rf *ReadFileSkill) Description() model.ToolDefinition {
	return model.ToolDefinition{
		Type: "function",
		Function: model.FunctionDefinition{
			Name:        "read_file",
			Description: "读取指定文件内容，可以指定从第几行开始读取，往后读取多少行",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "要读取的文件路径，给绝对路径，比如：/home/user/test.txt",
					},
					"start": map[string]interface{}{
						"type":        "integer",
						"description": "开始读取的行号，从1开始，默认为1",
					},
					"line_count": map[string]interface{}{
						"type":        "integer",
						"description": "要读取的行数，默认为0，表示读取到文件结尾",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

// Execute 执行技能
func (rf *ReadFileSkill) Execute(argsJSON string) string {
	var args struct {
		Path      string `json:"path"`
		Start     int    `json:"start"`
		LineCount int    `json:"line_count"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("解析工具参数失败: %v", err)
	}
	if args.Path == "" {
		return "文件路径不能为空"
	}
	_, err := os.Stat(args.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("文件不存在: %s", args.Path)
		}
		return fmt.Sprintf("文件读取失败: %v", err)
	}
	if args.Start <= 0 {
		args.Start = 1
	}
	// 获取脚本的绝对路径（相对于可执行文件或项目根目录）
	scriptPath := "skill/bash/read_lines.sh"
	cmd := exec.Command("bash", scriptPath, args.Path, fmt.Sprintf("%d", args.Start), fmt.Sprintf("%d", args.LineCount))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("执行脚本失败: %v", err)
	}
	return string(output)
}
