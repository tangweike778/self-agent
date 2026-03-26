package model

// Header 事件头部信息
type Header struct {
	EventID    string `json:"event_id"`
	Token      string `json:"token"`
	CreateTime string `json:"create_time"`
	EventType  string `json:"event_type"`
	TenantKey  string `json:"tenant_key"`
	AppID      string `json:"app_id"`
}

// SenderID 发送者ID信息
type SenderID struct {
	OpenID  string  `json:"open_id"`
	UnionID string  `json:"union_id"`
	UserID  *string `json:"user_id"`
}

// Sender 发送者信息
type Sender struct {
	SenderID   SenderID `json:"sender_id"`
	SenderType string   `json:"sender_type"`
	TenantKey  string   `json:"tenant_key"`
}

// Message 消息内容
type Message struct {
	ChatID      string `json:"chat_id"`
	ChatType    string `json:"chat_type"`
	Content     string `json:"content"`
	CreateTime  string `json:"create_time"`
	MessageID   string `json:"message_id"`
	MessageType string `json:"message_type"`
	UpdateTime  string `json:"update_time"`
}

// Event 事件主体
type Event struct {
	Message Message `json:"message"`
	Sender  Sender  `json:"sender"`
}

// FeishuEventData 飞书事件数据完整结构
type FeishuEventData struct {
	Schema string `json:"schema"`
	Header Header `json:"header"`
	Event  Event  `json:"event"`
}
