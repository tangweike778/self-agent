package agent

import (
	"self-agent/model"
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
