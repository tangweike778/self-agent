package agent

import (
	"bytes"
	"fmt"
	"log"
	"self-agent/model"

	"github.com/samber/lo"
)

const (
	DialogStartIdx       = 1
	defaultReserveDialog = 3
)

// Compressor 压缩器
type Compressor struct {
	Agent *Agent
}

// CompressMessages 消息压缩
func (c *Compressor) CompressMessages(messages []model.AgentMessage, maxToken int64) ([]model.AgentMessage, error) {
	// 筛选出三轮对话之前的信息,找到要压缩的位置idx，小于idx的所有信息都要进行压缩
	compressIdx := findCompressEndIds(messages)
	if compressIdx < DialogStartIdx {
		// 会话不足三轮，直接清空上下文，仅保留最后一轮对话
		return getSystemPromptAndLatestDialog(messages)
	}
	availableToken := c.computeAvailableToken(maxToken, messages[compressIdx:])
	if availableToken <= 0 {
		// token不够，直接清空上下文，仅保留最后一轮对话
		return getSystemPromptAndLatestDialog(messages)
	}
	rawMessages := messages[DialogStartIdx:compressIdx]
	// 调用LLM对消息进行压缩
	var msgMerged bytes.Buffer
	for _, message := range rawMessages {
		msgMerged.WriteString(fmt.Sprintf("%s:%s\n", message.Role, message.Content))
	}
	prompt := fmt.Sprintf("请你站在system的视角，帮我提取、总结一下当前的内容，尽量保留决策、结论。保证文本的token不超过%d", availableToken)
	compressedMessage, err := c.Agent.SingleAsk(msgMerged.String(), prompt)
	if err != nil {
		// LLM压缩失败，直接清空上下文，仅保留最后一轮对话
		log.Printf("LLM压缩失败，直接清空上下文，仅保留最后一轮对话，err: %v", err)
		return getSystemPromptAndLatestDialog(messages)
	}
	handledMessage := lo.Concat(messages[0:DialogStartIdx], []model.AgentMessage{{Role: "user", Content: compressedMessage}}, messages[compressIdx:])
	return handledMessage, nil
}

// findCompressEndIds 找到压缩的终点位置，即三轮对话之前的位置or0
func findCompressEndIds(messages []model.AgentMessage) int32 {
	var (
		userIdx     int32
		compressIdx int32
	)
	for i := range messages {
		message := messages[len(messages)-i-1]
		if message.Role == "user" {
			userIdx++
		}
		if userIdx == defaultReserveDialog {
			compressIdx = int32(len(messages) - i - 1)
			break
		}
	}
	return compressIdx
}

func getSystemPromptAndLatestDialog(messages []model.AgentMessage) ([]model.AgentMessage, error) {
	for i := range messages {
		if messages[len(messages)-i-1].Role == "user" {
			return append(messages[:DialogStartIdx], messages[len(messages)-i-1:]...), nil
		}
		// 如果没有user消息，直接返回错误
		if i == len(messages)-1 {
			return nil, fmt.Errorf("no user message")
		}
	}
	return []model.AgentMessage{}, nil
}

// computeAvailableToken 计算可用的token, 保留70%
func (c *Compressor) computeAvailableToken(token int64, messages []model.AgentMessage) int64 {
	return lo.Max([]int64{(token*7)/10 - c.Agent.Estimator.ComputeTokens(messages), 0})
}
