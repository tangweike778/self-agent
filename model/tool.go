package model

// ToolDefinition 工具定义，用于Deepseek function calling
type ToolDefinition struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition 函数定义
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall LLM返回的工具调用请求
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall 函数调用详情
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// AgentRequestWithTools 带有工具定义的Agent请求
type AgentRequestWithTools struct {
	Messages  []AgentMessage   `json:"messages"`
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	Tools     []ToolDefinition `json:"tools,omitempty"`
}

// AgentResponseChoice 响应中的选项
type AgentResponseChoice struct {
	Message      AgentMessageWithToolCalls `json:"message"`
	FinishReason string                    `json:"finish_reason"`
}

// AgentMessageWithToolCalls 带有工具调用的消息
type AgentMessageWithToolCalls struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// AgentResponseWithTools 带有工具调用的API响应
type AgentResponseWithTools struct {
	Choices []AgentResponseChoice `json:"choices"`
}
