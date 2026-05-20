package agent

import (
	"self-agent/model"
	"testing"
)

func TestBuildDAG_Normal(t *testing.T) {
	plan := &model.Plan{
		SubTasks: []model.SubTask{
			{ID: "1", Description: "任务1", Dependencies: []string{}},
			{ID: "2", Description: "任务2", Dependencies: []string{"1"}},
			{ID: "3", Description: "任务3", Dependencies: []string{"1"}},
			{ID: "4", Description: "任务4", Dependencies: []string{"2", "3"}},
		},
	}
	levels, err := buildDAG(plan)
	if err != nil {
		t.Fatalf("buildDAG failed: %v", err)
	}
	if len(levels) != 3 {
		t.Errorf("expected 3 levels, got %d", len(levels))
	}
	// 第一层应该只有任务1
	if len(levels[0]) != 1 || levels[0][0] != "1" {
		t.Errorf("first level should be [1], got %v", levels[0])
	}
	// 第二层应该有任务2和3
	if len(levels[1]) != 2 {
		t.Errorf("second level should have 2 tasks, got %d", len(levels[1]))
	}
	// 第三层应该只有任务4
	if len(levels[2]) != 1 || levels[2][0] != "4" {
		t.Errorf("third level should be [4], got %v", levels[2])
	}
}

func TestBuildDAG_CyclicDependency(t *testing.T) {
	plan := &model.Plan{
		SubTasks: []model.SubTask{
			{ID: "1", Description: "任务1", Dependencies: []string{"3"}},
			{ID: "2", Description: "任务2", Dependencies: []string{"1"}},
			{ID: "3", Description: "任务3", Dependencies: []string{"2"}},
		},
	}
	_, err := buildDAG(plan)
	if err == nil {
		t.Error("expected error for cyclic dependency, got nil")
	}
}

func TestBuildDAG_InvalidDependency(t *testing.T) {
	plan := &model.Plan{
		SubTasks: []model.SubTask{
			{ID: "1", Description: "任务1", Dependencies: []string{"999"}},
		},
	}
	_, err := buildDAG(plan)
	if err == nil {
		t.Error("expected error for invalid dependency, got nil")
	}
}

func TestBuildDAG_NoDependencies(t *testing.T) {
	plan := &model.Plan{
		SubTasks: []model.SubTask{
			{ID: "1", Description: "任务1", Dependencies: []string{}},
			{ID: "2", Description: "任务2", Dependencies: []string{}},
			{ID: "3", Description: "任务3", Dependencies: []string{}},
		},
	}
	levels, err := buildDAG(plan)
	if err != nil {
		t.Fatalf("buildDAG failed: %v", err)
	}
	// 所有任务都在第一层
	if len(levels) != 1 {
		t.Errorf("expected 1 level, got %d", len(levels))
	}
	if len(levels[0]) != 3 {
		t.Errorf("first level should have 3 tasks, got %d", len(levels[0]))
	}
}

func TestPlanProgress(t *testing.T) {
	plan := &model.Plan{
		SubTasks: []model.SubTask{
			{ID: "1", Status: model.SubTaskStatusCompleted},
			{ID: "2", Status: model.SubTaskStatusRunning},
			{ID: "3", Status: model.SubTaskStatusPending},
		},
	}
	progress := plan.Progress()
	expected := "已完成 1/3 步骤"
	if progress != expected {
		t.Errorf("Progress() = %q, want %q", progress, expected)
	}
}
