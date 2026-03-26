package session

// Initializable 可初始化的
type Initializable interface {
	Init() error
}
