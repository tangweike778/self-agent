package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"self-agent/config"
	"self-agent/model"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkevent "github.com/larksuite/oapi-sdk-go/v3/event"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

// ChannelType 渠道类型
type ChannelType string

const (
	ChannelTypeFeishu   ChannelType = "feishu"   // 飞书机器人
	ChannelTypeWechat   ChannelType = "wechat"   // 微信机器人
	ChannelTypeDingtalk ChannelType = "dingtalk" // 钉钉机器人
)

// Channel 渠道
type Channel struct {
	ID         string      `json:"id"`
	Type       ChannelType `json:"type"`
	AppID      string      `json:"app_id"`
	WebhookURL string      `json:"webhook_url"`
	Secret     string      `json:"secret,omitempty"`
	IsActive   bool        `json:"is_active"`
	CreatedAt  time.Time   `json:"created_at"`
	LastUsedAt time.Time   `json:"last_used_at"`

	// SessionCallback 会话回调
	SessionCallback func(string, string)
}

// FeishuMessage 飞书消息结构
type FeishuMessage struct {
	MsgType string `json:"msg_type"`
	Content struct {
		Text string `json:"text"`
	} `json:"content"`
}

// NewChannel 创建渠道
func NewChannel(id string, channelType ChannelType, cfg config.ChannelCfg, sessionCallback func(string, string)) *Channel {
	var (
		appID      string
		webhookURL string
		secret     string
	)
	switch channelType {
	case ChannelTypeFeishu:
		appID = cfg.Feishu.AppID
		webhookURL = cfg.Feishu.Webhook
		secret = cfg.Feishu.Secret
	}
	return &Channel{
		ID:              id,
		AppID:           appID,
		Type:            channelType,
		WebhookURL:      webhookURL,
		Secret:          secret,
		IsActive:        true,
		CreatedAt:       time.Now(),
		LastUsedAt:      time.Now(),
		SessionCallback: sessionCallback,
	}
}

// Init 初始化
func (c *Channel) Init() error {
	if c.Type == ChannelTypeFeishu {
		go func() {
			// 注册事件回调，OnP2MessageReceiveV1 为接收消息 v2.0；OnCustomizedEvent 内的 message 为接收消息 v1.0。
			eventHandler := dispatcher.NewEventDispatcher("", "").
				OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
					if event == nil {
						return nil
					}
					var eventData model.FeishuEventData
					_ = json.Unmarshal(event.Body, &eventData)
					// 发送消息给所属session
					c.SessionCallback(c.ID, eventData.Event.Message.Content)
					return nil
				}).
				OnCustomizedEvent("这里填入你要自定义订阅的 event 的 key，例如 out_approval", func(ctx context.Context, event *larkevent.EventReq) error {
					fmt.Printf("[ OnCustomizedEvent access ], type: message, data: %s\n", string(event.Body))
					return nil
				})
			// 创建Client
			cli := larkws.NewClient(c.AppID, c.Secret,
				larkws.WithEventHandler(eventHandler),
				larkws.WithLogLevel(larkcore.LogLevelDebug),
			)
			// 启动客户端
			err := cli.Start(context.Background())
			if err != nil {
				panic(fmt.Errorf("failed to start larkws client: %v", err))
			}
		}()
	}
	return nil
}

// SendMessage 发送消息到渠道
func (c *Channel) SendMessage(message string) error {
	if !c.IsActive {
		return fmt.Errorf("channel %s is not active", c.ID)
	}

	switch c.Type {
	case ChannelTypeFeishu:
		return c.sendToFeishu(message)
	default:
		return fmt.Errorf("unsupported channel type: %s", c.Type)
	}
}

// sendToFeishu 发送消息到飞书机器人
func (c *Channel) sendToFeishu(message string) error {
	log.Printf("send to feishu: %s", message)
	// 创建 Client
	client := lark.NewClient(c.AppID, c.Secret)
	// 创建请求对象
	openID := config.GetConfig().Channel.Feishu.OpenID
	// 构造合法的 JSON content
	contentMap := map[string]interface{}{"config": map[string]interface{}{
		"wide_screen_mode": true, // 开启宽屏，显示效果更好
	},
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": "贾维斯", // 卡片标题
			},
			"template": "blue", // 标题栏颜色：red(告警), blue(信息), green(成功)
		},
		"elements": []map[string]interface{}{
			{
				"tag":     "markdown", // 或 "lark_md"，两者等效
				"content": message,
			},
		}}
	contentBytes, err := json.Marshal(contentMap)
	if err != nil {
		return fmt.Errorf("failed to marshal message content: %v", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(`open_id`).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(openID).
			MsgType(`interactive`).
			Content(string(contentBytes)).
			Build()).
		Build()

	// 发起请求
	resp, err := client.Im.V1.Message.Create(context.Background(), req)

	// 处理错误
	if err != nil {
		fmt.Println(err)
		return err
	}

	// 服务端错误处理
	if !resp.Success() {
		fmt.Printf("logId: %s, error response: \n%s", resp.RequestId(), larkcore.Prettify(resp.CodeError))
		return fmt.Errorf("failed to send message: %s", resp.CodeError)
	}

	// 业务处理
	fmt.Println(larkcore.Prettify(resp))
	c.LastUsedAt = time.Now()
	return nil
}

// BindToSession 将渠道绑定到会话
func (c *Channel) BindToSession(sessionID string) string {
	return fmt.Sprintf("channel:%s:session:%s", c.ID, sessionID)
}

// Activate 激活渠道
func (c *Channel) Activate() {
	c.IsActive = true
	c.LastUsedAt = time.Now()
}

// Deactivate 停用渠道
func (c *Channel) Deactivate() {
	c.IsActive = false
}
