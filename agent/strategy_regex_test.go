package agent

import (
	"self-agent/model"
	"strings"
	"testing"
)

func TestRegexMatch_PlanSolveKeywords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMode string
		wantHit  bool
	}{
		{"先再然后模式", "先读取文件再分析然后生成报告", model.StrategyModePlanSolve, true},
		{"步骤关键词", "请列出步骤", model.StrategyModePlanSolve, true},
		{"流程关键词", "描述一下流程", model.StrategyModePlanSolve, true},
		{"一步步关键词", "一步步教我", model.StrategyModePlanSolve, true},
		{"生成关键词", "生成一份报告", model.StrategyModePlanSolve, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, hit := regexMatch(tt.input)
			if hit != tt.wantHit {
				t.Errorf("regexMatch(%q) hit = %v, want %v", tt.input, hit, tt.wantHit)
			}
			if mode != tt.wantMode {
				t.Errorf("regexMatch(%q) mode = %v, want %v", tt.input, mode, tt.wantMode)
			}
		})
	}
}

func TestRegexMatch_ReactKeywords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMode string
		wantHit  bool
	}{
		{"查一下关键词", "帮我查一下天气", model.StrategyModeReact, true},
		{"看看关键词", "看看今天有什么新闻", model.StrategyModeReact, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, hit := regexMatch(tt.input)
			if hit != tt.wantHit {
				t.Errorf("regexMatch(%q) hit = %v, want %v", tt.input, hit, tt.wantHit)
			}
			if mode != tt.wantMode {
				t.Errorf("regexMatch(%q) mode = %v, want %v", tt.input, mode, tt.wantMode)
			}
		})
	}
}

func TestRegexMatch_LongInput(t *testing.T) {
	// 超过200字的输入应该走 ReAct
	longInput := strings.Repeat("这是一段很长的文字", 30)
	mode, hit := regexMatch(longInput)
	if !hit {
		t.Error("超长输入应该命中正则规则")
	}
	if mode != model.StrategyModeReact {
		t.Errorf("超长输入应该走ReAct，got %s", mode)
	}
}

func TestRegexMatch_NoMatch(t *testing.T) {
	// 普通输入不命中任何规则
	mode, hit := regexMatch("今天天气怎么样")
	if hit {
		t.Error("普通输入不应该命中正则规则")
	}
	if mode != "" {
		t.Errorf("未命中时mode应为空，got %s", mode)
	}
}

func TestRegexMatch_PlanSolveKeywordButLong(t *testing.T) {
	// 包含Plan-and-Solve关键词但字数超过50，不应命中Plan-and-Solve
	longInput := "先" + strings.Repeat("做一些事情", 10) + "再" + strings.Repeat("处理一些内容", 10) + "然后完成"
	mode, hit := regexMatch(longInput)
	// 字数超过50但不超过200，且不含ReAct关键词，应该不命中
	if hit && mode == model.StrategyModePlanSolve {
		t.Error("字数超过50的Plan-and-Solve关键词不应命中Plan-and-Solve模式")
	}
}
