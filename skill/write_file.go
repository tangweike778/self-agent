package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"self-agent/model"
)

// WriteFileSkill 写入文件技能
type WriteFileSkill struct{}

// Name 技能名称
func (wf *WriteFileSkill) Name() string {
	return "write_file"
}

// Description 技能描述
func (wf *WriteFileSkill) Description() model.ToolDefinition {
	return model.ToolDefinition{
		Type: "function",
		Function: model.FunctionDefinition{
			Name:        "write_file",
			Description: "创建或写入文件。支持覆盖写入和追加写入两种模式。如果文件所在目录不存在，会自动创建目录。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "要写入的文件路径，给绝对路径，比如：/home/user/test.txt",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "要写入的文件内容",
					},
					"mode": map[string]interface{}{
						"type":        "string",
						"description": "写入模式：overwrite（覆盖写入，默认）或 append（追加写入）",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}
}

// Execute 执行技能
func (wf *WriteFileSkill) Execute(argsJSON string) string {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Mode    string `json:"mode"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("解析工具参数失败: %v", err)
	}
	if args.Path == "" {
		return "文件路径不能为空"
	}
	if args.Mode == "" {
		args.Mode = "overwrite"
	}
	if args.Mode != "overwrite" && args.Mode != "append" {
		return fmt.Sprintf("不支持的写入模式: %s，仅支持 overwrite 或 append", args.Mode)
	}

	// 确保目录存在
	dir := filepath.Dir(args.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Sprintf("创建目录失败: %v", err)
	}

	// 根据模式选择文件打开方式
	var flag int
	if args.Mode == "append" {
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	} else {
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	file, err := os.OpenFile(args.Path, flag, 0644)
	if err != nil {
		return fmt.Sprintf("打开文件失败: %v", err)
	}
	defer file.Close()

	n, err := file.WriteString(args.Content)
	if err != nil {
		return fmt.Sprintf("写入文件失败: %v", err)
	}

	// 获取写入后的文件信息
	info, _ := os.Stat(args.Path)
	modeDesc := "覆盖写入"
	if args.Mode == "append" {
		modeDesc = "追加写入"
	}

	return fmt.Sprintf("文件写入成功\n路径: %s\n模式: %s\n本次写入: %d 字节\n文件总大小: %d 字节", args.Path, modeDesc, n, info.Size())
}
