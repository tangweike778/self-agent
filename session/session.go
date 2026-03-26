package session

import (
	"fmt"
	"log"
	"os"
	"self-agent/agent"
	"self-agent/channel"
	"self-agent/config"
	"self-agent/model"
)

// Session 会话
type Session struct {
	ID      string
	Tasks   *model.TaskQueue
	Agent   *agent.Agent
	Channel *channel.Channel     // 绑定渠道(非必填)
	History []model.AgentMessage // 会话消息（上下文记录）
}

// NewSession 创建会话
func NewSession(id string, apiKey string) *Session {
	sessionObj := &Session{
		ID:      id,
		Tasks:   model.NewTaskQueue(),
		Agent:   agent.NewAgent(apiKey),
		History: []model.AgentMessage{},
	}
	var channelObj *channel.Channel
	cfg := config.GetConfig()
	if id == "feishu" && cfg.HasFeishuConfig() {
		channelObj = channel.NewChannel(id, channel.ChannelTypeFeishu, cfg.Channel, func(id string, content string) {
			sessionObj.Tasks.AddTask(model.Task{
				Content: content,
			})
		})
	}
	sessionObj.SetChannel(channelObj)

	// 系统提示词注入
	// 构建系统提示词
	file, err := os.ReadFile("./prompt/system_prompt.md")
	if err != nil {
		log.Printf("Error reading file: %s", err)
	} else {
		sessionObj.History = append(sessionObj.History, model.AgentMessage{
			Role:    "system",
			Content: string(file),
		})
	}
	return sessionObj
}

// SetChannel 为会话设置渠道
func (s *Session) SetChannel(ch *channel.Channel) {
	s.Channel = ch
}

// SendToChannel 通过绑定的渠道发送消息
func (s *Session) SendToChannel(message string) error {
	if s.Channel == nil {
		return nil // 没有绑定渠道，静默返回
	}
	return s.Channel.SendMessage(message)
}

// HasChannel 检查会话是否绑定了渠道
func (s *Session) HasChannel() bool {
	return s.Channel != nil
}

// Start 开始会话
func (s *Session) Start() {
	for {
		// 构建好会话上下文
		s.History = append(s.History, model.AgentMessage{
			Role:    "user",
			Content: s.Tasks.GetTask().Content,
		})
		handledMsgs, skipAsk := handleBeforeAsk(s.History)
		// 发送给对应的Agent
		if skipAsk {
			s.History = handledMsgs
			continue
		}
		handledMsgs, err := s.Agent.Ask(s.History)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		s.History = handledMsgs
		// 将结果发送到绑定的渠道
		// TODO：这里要是没有绑定的渠道就不应该给它创建session，或者默认渠道就是console
		latestMsg := getLastMsg(s.History)
		if s.HasChannel() {
			if err := s.SendToChannel(latestMsg); err != nil {
				fmt.Println("Send to channel error:", err)
			}
		}
		fmt.Printf("Agent response: %s\n", latestMsg)
	}
}

func handleBeforeAsk(history []model.AgentMessage) ([]model.AgentMessage, bool) {
	// 找到最后一条用户消息
	var latestUserMsg string
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			latestUserMsg = history[i].Content
			break
		}
	}
	if latestUserMsg == "/clear" {
		// 清空上下文信息，跳过ask
		history = history[:agent.DialogStartIdx]
		return history, true
	}
	return history, false
}

func getLastMsg(messages []model.AgentMessage) string {
	return messages[len(messages)-1].Content
}

// Init 初始化会话
func (s *Session) Init() error {
	if err := s.Channel.Init(); err != nil {
		return err
	}
	return nil
}
