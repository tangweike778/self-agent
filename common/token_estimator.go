package common

import (
	"self-agent/model"
	"strings"
	"unicode"
)

// 不同编码方式的token估算
const (
	// OpenAI GPT模型的中文token比率（近似）
	ChineseCharPerToken = 0.8 // 每个汉字约1.25个token
	EnglishCharPerToken = 4.0 // 每个英文单词约4个字符
	PunctuationPerToken = 5.0 // 每个标点约5个字符
)

// TokenEstimator 基于字符数的token估算器
type TokenEstimator struct{}

// EstimateTokens 估算文本的总token数
func (te *TokenEstimator) EstimateTokens(text string) int64 {
	chineseCount, englishCount, punctuationCount := analyzeText(text)

	// 使用近似公式
	tokens := int64(float64(chineseCount)/ChineseCharPerToken) +
		int64(float64(englishCount)/EnglishCharPerToken) +
		int64(float64(punctuationCount)/PunctuationPerToken) +
		1 // 基础token

	return tokens
}

// EstimateTokensGPT 针对GPT模型的更精确估算
func (te *TokenEstimator) EstimateTokensGPT(text string) int {
	// 简单估算：汉字每个约1.5个token，英文每个单词约1.3个token
	chineseCount, _, _ := analyzeText(text)

	// 统计英文单词数
	englishWords := 0
	fields := strings.Fields(text)
	for _, field := range fields {
		if isEnglishWord(field) {
			englishWords++
		}
	}

	tokens := int(float64(chineseCount)*1.5) + int(float64(englishWords)*1.3) + 3
	return tokens
}

// analyzeText 分析文本中的字符类型
func analyzeText(text string) (chineseCount, englishCount, punctuationCount int) {
	for _, r := range text {
		switch {
		case unicode.Is(unicode.Han, r): // 中文字符
			chineseCount++
		case unicode.IsLetter(r) && (r < 0x80): // 英文字母
			englishCount++
		case unicode.IsPunct(r) || unicode.IsSpace(r): // 标点和空格
			punctuationCount++
		}
	}
	return
}

func isEnglishWord(s string) bool {
	for _, r := range s {
		if r > 0x7F { // 非ASCII字符
			return false
		}
	}
	return true
}

// ComputeTokens 计算token数
func (te *TokenEstimator) ComputeTokens(messages []model.AgentMessage) int64 {
	// 获取每个message的content
	builder := strings.Builder{}
	for _, message := range messages {
		builder.WriteString(message.Content)
	}
	tokens := te.EstimateTokens(builder.String())
	return tokens
}
