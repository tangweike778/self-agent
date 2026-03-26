package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"self-agent/config"
	"self-agent/model"
	"self-agent/session"
)

// Gateway 网关
type Gateway struct {
	Sessions map[string]*session.Session
	Config   *config.Config
}

// NewGateway 创建网关
func NewGateway() *Gateway {
	cfg, err := config.LoadConfig()
	if err != nil {
		// 如果配置加载失败，使用默认配置
		cfg = config.LoadConfigWithDefaults()
	}

	return &Gateway{
		Sessions: make(map[string]*session.Session),
		Config:   cfg,
	}
}

// ServeHTTP 处理http请求
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		g.handlePostRequest(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePostRequest 处理POST请求（发送消息到AI）
func (g *Gateway) handlePostRequest(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var reqBody model.AIReqBody
	_ = json.Unmarshal(body, &reqBody)

	sessionKey := buildSessionKey(reqBody.ID)
	sessionObj, ok := g.Sessions[sessionKey]
	if !ok {
		http.Error(w, fmt.Sprintf("session %s not found", sessionKey), http.StatusNotFound)
		return
	}

	// 将Content封装为Task，加入到session的Tasks中
	sessionObj.Tasks.AddTask(model.Task{
		Content: reqBody.Content,
	})

	// 如果有绑定的渠道，将消息发送到渠道
	if sessionObj.HasChannel() {
		_ = sessionObj.SendToChannel(fmt.Sprintf("收到新消息: %s", reqBody.Content))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "success",
		"session_id":  reqBody.ID,
		"has_channel": sessionObj.HasChannel(),
	})
}

// Start 启动Gateway服务
func (g *Gateway) Start() {
	// 遍历所有session，启动它们的Agent
	for _, sessionObj := range g.Sessions {
		sessionObj := sessionObj
		go func() {
			sessionObj.Start()
		}()
	}
}

// RegisterSession 注册新的会话
func (g *Gateway) RegisterSession(id string) {
	g.Sessions[buildSessionKey(id)] = session.NewSession(id, g.Config.Deepseek.APIKey)
}

// AutoRegisterSession 自动注册会话
func (g *Gateway) AutoRegisterSession() {
	// 根据配置文件自动创建对应channel的session
	cfg := config.GetConfig()
	if cfg.HasFeishuConfig() {
		// 创建飞书session
		g.RegisterSession("feishu")
	}

}

// Init 初始化Gateway
func (g *Gateway) Init() error {
	// 初始化每一个session，遇到错误则返回错误
	for _, ses := range g.Sessions {
		// 初始化channel
		if err := ses.Init(); err != nil {
			return err
		}
	}
	return nil
}

func buildSessionKey(id string) string {
	return fmt.Sprintf("session:%s", id)
}
