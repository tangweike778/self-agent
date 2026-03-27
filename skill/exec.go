package skill

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"self-agent/model"
	"strings"
	"time"
)

// ExecShellSkill 执行shell命令
type ExecShellSkill struct{}

// Name 获取技能名称
func (e *ExecShellSkill) Name() string {
	return "exec_shell"
}

// Description 获取技能描述
func (e *ExecShellSkill) Description() model.ToolDefinition {
	return model.ToolDefinition{
		Type: "function",
		Function: model.FunctionDefinition{
			Name:        "exec_shell",
			Description: "在服务器上执行shell命令。可以用来查看文件、运行脚本、查询系统状态等。注意：命令会在服务器上真实执行，请谨慎使用。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "要执行的shell命令",
					},
					"timeout": map[string]interface{}{
						"type":        "integer",
						"description": "命令超时时间（秒），默认30秒，最大120秒",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

// ExecShellResult shell命令执行结果
type ExecShellResult struct {
	Command  string `json:"command"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}

// ExecShell 执行shell命令并返回结果
// 设置超时时间防止命令长时间阻塞
func (e *ExecShellSkill) ExecShell(command string, timeoutSeconds int) *ExecShellResult {
	result := &ExecShellResult{
		Command: command,
	}

	if timeoutSeconds <= 0 {
		timeoutSeconds = 30 // 默认30秒超时
	}

	// 根据操作系统选择shell
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("/bin/sh", "-c", command)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 启动命令
	if err := cmd.Start(); err != nil {
		result.Error = fmt.Sprintf("启动命令失败: %v", err)
		result.ExitCode = -1
		return result
	}

	// 等待命令完成，带超时
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		result.Stdout = strings.TrimSpace(stdout.String())
		result.Stderr = strings.TrimSpace(stderr.String())
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				result.ExitCode = exitErr.ExitCode()
			} else {
				result.ExitCode = -1
				result.Error = fmt.Sprintf("命令执行失败: %v", err)
			}
		} else {
			result.ExitCode = 0
		}
	case <-time.After(time.Duration(timeoutSeconds) * time.Second):
		_ = cmd.Process.Kill()
		result.ExitCode = -1
		result.Error = fmt.Sprintf("命令执行超时（%d秒）", timeoutSeconds)
	}

	return result
}

// FormatResult 将执行结果格式化为适合LLM阅读的字符串
func (r *ExecShellResult) FormatResult() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("命令: %s\n", r.Command))
	sb.WriteString(fmt.Sprintf("退出码: %d\n", r.ExitCode))
	if r.Stdout != "" {
		sb.WriteString(fmt.Sprintf("标准输出:\n%s\n", r.Stdout))
	}
	if r.Stderr != "" {
		sb.WriteString(fmt.Sprintf("标准错误:\n%s\n", r.Stderr))
	}
	if r.Error != "" {
		sb.WriteString(fmt.Sprintf("错误信息: %s\n", r.Error))
	}
	return sb.String()
}

// Execute 执行shell命令工具
func (e *ExecShellSkill) Execute(argsJSON string) string {
	var args struct {
		Command string `json:"command"`
		Timeout int    `json:"timeout"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("解析工具参数失败: %v", err)
	}

	if args.Timeout <= 0 {
		args.Timeout = 30
	}
	if args.Timeout > 120 {
		args.Timeout = 120
	}

	log.Printf("执行shell命令: %s (超时: %ds)", args.Command, args.Timeout)
	result := e.ExecShell(args.Command, args.Timeout)
	return result.FormatResult()
}
