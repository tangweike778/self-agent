package model

// TaskQueue 任务队列
type TaskQueue struct {
	task chan Task
}

// Task 任务
type Task struct {
	Content string
}

// NewTaskQueue 创建新的任务队列
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		task: make(chan Task),
	}
}

// AddTask 添加任务
func (t *TaskQueue) AddTask(task Task) {
	t.task <- task
}

// GetTask 获取任务
func (t *TaskQueue) GetTask() *Task {
	task := <-t.task
	return &task
}

// Close 关闭任务队列
func (t *TaskQueue) Close() {
	close(t.task)
}
