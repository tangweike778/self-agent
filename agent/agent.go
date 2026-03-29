package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"self-agent/common"
	"self-agent/config"
	"self-agent/model"
	"self-agent/skill"
	"time"

	retry "github.com/avast/retry-go"
	"github.com/samber/lo"
)

// Agent AI代理
type Agent struct {
	APIKey     string
	BaseURL    string
	Tools      []model.ToolDefinition
	Estimator  *common.TokenEstimator
	ToolByName map[string]skill.Skill
}

// NewAgent 创建新的Agent实例
func NewAgent(apiKey string) *Agent {
	a := &Agent{
		APIKey:     apiKey,
		BaseURL:    "https://api.deepseek.com/v1/chat/completions",
		Estimator:  &common.TokenEstimator{},
		ToolByName: make(map[string]skill.Skill),
	}
	// 注册可用的工具
	for _, tool := range skill.AllSkills() {
		a.registerTool(tool)
	}
	return a
}

// registerTool 注册工具
func (a *Agent) registerTool(tool skill.Skill) {
	a.Tools = append(a.Tools, tool.Description())
	a.ToolByName[tool.Name()] = tool
}

// SingleAsk 向Deepseek API提问（单轮对话）
func (a *Agent) SingleAsk(question string, prompt string) (string, error) {
	messages := []model.AgentMessage{
		{
			Role:    "user",
			Content: question,
		},
		{
			Role:    "system",
			Content: prompt,
		},
	}
	var (
		resp *model.AgentResponseWithTools
		err  error
	)
	retry.Do(
		func() error {
			log.Printf("提问: %s", question)
			resp, err = a.callAPI(messages)
			if err != nil {
				return err
			}
			return nil
		},
		retry.Attempts(3),
		retry.Delay(time.Second),
		retry.OnRetry(func(n uint, err error) {
			log.Printf("提问失败，重试第%d次，err: %v", n, err)
		}),
	)
	log.Printf("回答: %s", resp.Choices[0].Message.Content)
	return resp.Choices[0].Message.Content, nil
}

// Ask 向Deepseek API提问（支持function calling多轮对话）
func (a *Agent) Ask(messages []model.AgentMessage) ([]model.AgentMessage, error) {
	trace := &ReActTrace{
		Question: getLastUserQuestion(messages),
	}
	traceStart := time.Now()
	// 多轮tool调用循环，最多10轮防止无限循环
	for i := 0; i < 10; i++ {
		step := ReActStep{
			StepNum: i + 1,
		}
		stepStart := time.Now()
		// 优先检查messages所需token，如果超过则进行压缩
		maxToken := getSystemMaxToken()
		needToken := a.Estimator.ComputeTokens(messages)
		log.Printf("当前消息所需token: %d, 最大token: %d", needToken, maxToken)
		if needToken >= maxToken {
			var err error
			messages, err = compress(messages, a, maxToken)
			if err != nil {
				return []model.AgentMessage{
					{
						Role:    "assistant",
						Content: fmt.Sprintf("%s", err),
					},
				}, nil
			}
		}
		resp, err := a.callAPI(messages)
		if err != nil {
			return nil, err
		}
		if len(resp.Choices) == 0 {
			return nil, fmt.Errorf("API返回空响应")
		}
		choice := resp.Choices[0]
		// === 新增：解析LLM输出 ===
		parsed := ParseLLMOutput(choice.Message.Content)
		step.Thought = parsed.Thought
		// 如果LLM没有调用工具，说明已经得出最终回复
		if choice.FinishReason != "tool_calls" || len(choice.Message.ToolCalls) == 0 {
			messages = append(messages, model.AgentMessage{
				Role:    "assistant",
				Content: choice.Message.Content,
			})
			trace.Answer = choice.Message.Content
			trace.TotalTime = time.Since(traceStart)
			log.Printf("[ReAct] 推理完成，共 %d 步，耗时 %v", len(trace.Steps), trace.TotalTime)
			log.Printf("complete trace: %s", trace.GetLog())
			return messages, nil
		}
		// LLM请求调用工具，将assistant消息加入上下文
		messages = append(messages, model.AgentMessage{
			Role:      "assistant",
			Content:   choice.Message.Content,
			ToolCalls: choice.Message.ToolCalls,
		})
		// 执行每个工具调用，将结果作为tool消息加入上下文
		for _, toolCall := range choice.Message.ToolCalls {
			step.ToolCalls = append(step.ToolCalls, toolCall.Function.Name)
			step.Action = toolCall.Function.Name
			toolResult := a.executeTool(toolCall)
			step.Observation = toolResult
			messages = append(messages, model.AgentMessage{
				Role:       "tool",
				Content:    toolResult,
				ToolCallID: toolCall.ID,
			})
			if isToolError(toolResult) {
				step.Success = false
				// 工具执行失败，写入reflection
				step.Reflection = fmt.Sprintf("工具%s执行失败: %s", toolCall.Function.Name, toolResult)
				reflectionPrompt := fmt.Sprintf(
					"上一步工具调用失败了。请反思：\n1. 失败原因是什么？\n2. 是否需要换一种方式？\n3. 下一步应该怎么做？\n请在 <reflection> 标签中输出你的反思。",
				)
				messages = append(messages, model.AgentMessage{
					Role:    "user",
					Content: reflectionPrompt,
				})
			} else {
				step.Success = true
			}
		}
		step.Duration = time.Since(stepStart)
		trace.AddStep(step)
		log.Printf("第%d轮tool调用完成，继续请求LLM...", i+1)
	}
	return messages, fmt.Errorf("工具调用轮次超过上限")
}

func getLastUserQuestion(messages []model.AgentMessage) string {
	if msg, _, ok := lo.FindLastIndexOf(messages, func(msg model.AgentMessage) bool {
		return msg.Role == "user"
	}); ok {
		return msg.Content
	}
	return ""
}

func compress(messages []model.AgentMessage, a *Agent, maxToken int64) ([]model.AgentMessage, error) {
	compressor := &Compressor{a}
	compressMessages, err := compressor.CompressMessages(messages, maxToken)
	if err != nil {
		return nil, fmt.Errorf("压缩消息失败: %v", err)
	}
	tokenAfterCompresss := a.Estimator.ComputeTokens(compressMessages)
	if tokenAfterCompresss > maxToken {
		return nil, fmt.Errorf("token超过限制，无法继续对话")
	}
	log.Printf("压缩后剩余token: %d", a.Estimator.ComputeTokens(compressMessages))
	messages = compressMessages
	return messages, nil
}

func getSystemMaxToken() int64 {
	return lo.Min([]int64{
		config.GetConfig().Deepseek.MaxTokens,
		config.GetConfig().Server.MaxTokens,
	})
}

// callAPI 调用Deepseek API
func (a *Agent) callAPI(messages []model.AgentMessage) (*model.AgentResponseWithTools, error) {
	requestBody := model.AgentRequestWithTools{
		Messages:  messages,
		Model:     "deepseek-chat",
		MaxTokens: 4096,
		Tools:     a.Tools,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求体失败: %v", err)
	}

	req, err := http.NewRequest("POST", a.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API请求失败: %s, 响应: %s", resp.Status, string(body))
	}

	var apiResp model.AgentResponseWithTools
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &apiResp, nil
}

// executeTool 根据工具调用请求执行对应的工具
func (a *Agent) executeTool(toolCall model.ToolCall) string {
	if tool, ok := a.ToolByName[toolCall.Function.Name]; ok {
		return tool.Execute(toolCall.Function.Arguments)
	}
	return fmt.Sprintf("未知工具: %s", toolCall.Function.Name)
}
