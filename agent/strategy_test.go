package agent

import (
	"self-agent/model"
	"strings"
	"testing"
)

func TestSelectStrategy_EmptyInput(t *testing.T) {
	a := &Agent{}
	result := SelectStrategy(a, "")
	if result.Mode != model.StrategyModeReact {
		t.Errorf("empty input should return react, got %s", result.Mode)
	}
}

func TestSelectStrategy_RegexHit(t *testing.T) {
	a := &Agent{}
	// 正则命中 Plan-and-Solve
	result := SelectStrategy(a, "先分析再执行然后汇报")
	if result.Mode != model.StrategyModePlanSolve {
		t.Errorf("expected plan_and_solve, got %s", result.Mode)
	}
	if result.Confidence != 1.0 {
		t.Errorf("regex hit should have confidence 1.0, got %f", result.Confidence)
	}
	// 正则命中 ReAct
	result = SelectStrategy(a, "帮我查一下天气")
	if result.Mode != model.StrategyModeReact {
		t.Errorf("expected react, got %s", result.Mode)
	}
}

func TestRegexMatch_PlanSolveKeywords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMode string
		wantHit  bool
	}{
		{"包含先再然后", "先读取文件再分析然后生成报告", model.StrategyModePlanSolve, true},
		{"先再然后乱序", "然后汇报，我想先分析再执行", model.StrategyModePlanSolve, true},
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
	longInput := strings.Repeat("这是一段很长的文字", 30)
	mode, hit := regexMatch(longInput)
	if !hit {
		t.Error("超长输入应该命中正则规则")
	}
	if mode != model.StrategyModePlanSolve {
		t.Errorf("超长输入应该走PlanSolve，got %s", mode)
	}
}

func TestRegexMatch_NoMatch(t *testing.T) {
	mode, hit := regexMatch("今天天气怎么样")
	if hit {
		t.Error("普通输入不应该命中正则规则")
	}
	if mode != "" {
		t.Errorf("未命中时mode应为空，got %s", mode)
	}
}

func TestRegexMatch_PlanSolveKeywordButLong(t *testing.T) {
	// 新规则：Plan-and-Solve 关键词无字数限制，长文本也应命中
	longInput := "先" + strings.Repeat("做一些事情", 10) + "再" + strings.Repeat("处理一些内容", 10) + "然后完成"
	mode, hit := regexMatch(longInput)
	if !hit {
		t.Error("包含先再然后关键词的长文本应该命中")
	}
	if mode != model.StrategyModePlanSolve {
		t.Errorf("包含先再然后关键词应走PlanSolve，got %s", mode)
	}
}
