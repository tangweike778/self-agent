package model

// AgentMessage AI代理消息
type AgentMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// AgentRequest AI代理请求
type AgentRequest struct {
	Messages  []AgentMessage `json:"messages"`
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
}
