package session

import (
	"fmt"
	"log"
	"os"
	"self-agent/agent"
	"self-agent/channel"
	"self-agent/config"
	"self-agent/model"

	"github.com/samber/lo"
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
	return s.Channel.SendMessage(tidyMessage(message))
}

// tidyMessage 清理消息(将<thought>包裹的消息屏蔽)
func tidyMessage(message string) string {
	return agent.ThoughtRegex.ReplaceAllString(message, "")
}

// HasChannel 检查会话是否绑定了渠道
func (s *Session) HasChannel() bool {
	return s.Channel != nil
}

// Start 开始会话
func (s *Session) Start() {
	for {
		// 构建好会话上下文
		userInput := s.Tasks.GetTask().Content
		s.History = append(s.History, model.AgentMessage{
			Role:    "user",
			Content: userInput,
		})
		handledMsgs, skipAsk := handleBeforeAsk(s.History)
		// 发送给对应的Agent
		if skipAsk {
			s.History = handledMsgs
			continue
		}
		// 策略路由判断
		strategy := agent.SelectStrategy(s.Agent, userInput)
		log.Printf("[Session] 策略判断: mode=%s, confidence=%.2f", strategy.Mode, strategy.Confidence)
		var response string
		var err error
		switch {
		case strategy.Confidence <= 0.5:
			// 低置信度：双路径并行执行
			response, err = agent.ExecuteDualPath(s.Agent, userInput, s.History)
		case strategy.Mode == model.StrategyModePlanSolve:
			// Plan-and-Solve 模式
			plan, planErr := agent.GeneratePlan(s.Agent, userInput, s.History)
			if planErr != nil {
				err = planErr
				break
			}
			progressFn := func(progress string) {
				if s.HasChannel() {
					_ = s.SendToChannel(progress)
				}
				log.Printf("[Plan-and-Solve] %s", progress)
			}
			response, err = agent.ExecutePlan(s.Agent, plan, s.History, progressFn)
		default:
			// ReAct 模式（默认）
			handledMsgs, err = s.Agent.Ask(s.History)
			if err == nil {
				s.History = handledMsgs
				response = getLastMsg(s.History)
			}
		}
		if err != nil {
			s.History = rollbackUserMsg(s.History)
			fmt.Println("Error:", err)
			continue
		}
		// 非 ReAct 模式需要手动将结果追加到历史
		if strategy.Mode != model.StrategyModeReact || strategy.Confidence <= 0.5 {
			s.History = append(s.History, model.AgentMessage{
				Role:    "assistant",
				Content: response,
			})
		}
		// 将结果发送到绑定的渠道
		if s.HasChannel() {
			if err := s.SendToChannel(response); err != nil {
				fmt.Println("Send to channel error:", err)
			}
		}
		fmt.Printf("Agent response: %s\n", response)
	}
}

func rollbackUserMsg(history []model.AgentMessage) []model.AgentMessage {
	// 找到最后一条用户消息
	_, idx, find := lo.FindLastIndexOf(history, func(msg model.AgentMessage) bool {
		return msg.Role == "user"
	})
	if find {
		return history[:idx]
	}
	return history
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
	if len(messages) == 0 {
		return ""
	}
	return messages[len(messages)-1].Content
}

// Init 初始化会话
func (s *Session) Init() error {
	if err := s.Channel.Init(); err != nil {
		return err
	}
	return nil
}
