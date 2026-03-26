# Self-Agent 项目

一个基于Go的AI代理系统，支持会话管理和Deepseek API集成，并提供多渠道消息转发功能。

## 功能特性

- 会话管理：每个会话都有独立的任务队列和AI代理
- Deepseek API集成：支持与Deepseek AI模型进行对话
- HTTP网关：提供RESTful API接口
- YAML配置管理：使用YAML文件管理所有配置项
- **渠道集成**：支持为每个Session绑定消息渠道（如飞书机器人）
- **消息转发**：AI回复可自动转发到绑定的渠道

## 快速开始

### 1. 配置YAML文件

编辑 `config/config.yaml` 文件，配置Deepseek API密钥和飞书机器人：

```yaml
# Self-Agent 配置文件
# 配置Deepseek API和服务器设置

# Deepseek API配置
deepseek:
  api_key: "your-actual-deepseek-api-key-here"  # 替换为您的真实Deepseek API密钥

# 服务器配置
server:
  port: 19100  # 服务端口，默认19100

# 日志配置
logging:
  level: "info"  # 日志级别: debug, info, warn, error
  format: "json"  # 日志格式: text, json

# 飞书机器人配置（可选）
feishu:
  default_webhook: "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-id"  # 飞书机器人webhook地址
  default_secret: "your-secret"    # 飞书机器人secret
```

### 2. 运行服务

```bash
go run main.go
```

服务将在配置的端口启动（默认 `localhost:19100`）。

### 3. 使用示例

#### 发送消息到AI

```bash
curl -X POST http://localhost:19100/ai \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-session",
    "content": "你好，请介绍一下你自己"
  }'
```

#### 为Session绑定飞书机器人

```bash
curl -X PUT http://localhost:19100/ai \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "test-session",
    "channel_type": "feishu",
    "webhook_url": "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-id",
    "secret": "your-secret"
  }'
```

绑定后，所有发送到该Session的消息都会自动转发到飞书机器人。

### 4. 直接使用Session和Channel

您也可以在代码中直接使用Session和Channel：

```go
package main

import (
    "fmt"
    "log"
    "self-agent/channel"
    "self-agent/config"
    "self-agent/session"
)

func main() {
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("配置加载失败: %v", err)
    }

    // 创建Session
    session := session.NewSession("test-session", cfg.Deepseek.APIKey)

    // 绑定飞书机器人
    if cfg.HasFeishuConfig() {
        feishuChannel := channel.NewChannel(
            "feishu-bot",
            channel.ChannelTypeFeishu,
            cfg.GetFeishuWebhook(),
            cfg.GetFeishuSecret(),
        )
        session.SetChannel(feishuChannel)
    }

    // 使用Agent进行问答并转发到Channel
    response, err := session.Agent.Ask("你好，请介绍一下你自己")
    if err != nil {
        log.Printf("Agent问答失败: %v", err)
        return
    }

    // 将回复转发到绑定的渠道
    if session.HasChannel() {
        err := session.SendToChannel(fmt.Sprintf("AI回复: %s", response))
        if err != nil {
            log.Printf("转发到渠道失败: %v", err)
        }
    }

    fmt.Printf("Agent回复: %s\n", response)
}
```

## API接口

### POST /ai - 发送消息到AI

**请求体:**
```json
{
    "id": "session-id",
    "content": "消息内容"
}
```

**响应:**
```json
{
    "status": "success",
    "session_id": "session-id",
    "has_channel": true
}
```

### PUT /ai - 绑定Channel到Session

**请求体:**
```json
{
    "session_id": "session-id",
    "channel_type": "feishu",
    "webhook_url": "飞书机器人webhook地址",
    "secret": "飞书机器人secret"
}
```

**响应:**
```json
{
    "status": "success",
    "session_id": "session-id",
    "channel_id": "channel-id",
    "channel_type": "feishu"
}
```

## 项目结构

```
self-agent/
├── config/           # 配置管理模块
│   ├── config.go     # 配置加载和解析逻辑
│   └── config.yaml   # YAML格式的配置文件
├── agent/           # AI代理模块
│   └── agent.go     # Agent结构体和Deepseek API集成
├── gateway/         # HTTP网关模块
│   └── gateway.go   # Gateway结构体和HTTP处理
├── session/         # 会话管理模块
│   └── session.go   # Session结构体
├── channel/         # 渠道管理模块（新增）
│   └── channel.go   # Channel结构体和飞书机器人集成
├── model/           # 数据模型
│   └── TaskQueue.go # 任务队列模型
├── examples/        # 使用示例（新增）
│   └── channel_example.go # Channel使用示例
└── main.go         # 主程序入口
```

## API密钥获取

要使用Deepseek API，您需要：

1. 访问 [Deepseek官网](https://www.deepseek.com)
2. 注册账号并获取API密钥
3. 将API密钥配置在 `config/config.yaml` 文件中

## 飞书机器人配置

要使用飞书机器人，您需要：

1. 在飞书开放平台创建机器人
2. 获取webhook地址和secret
3. 将配置添加到 `config/config.yaml` 文件中

## 注意事项

- 确保网络连接正常，Agent需要访问Deepseek API
- API调用可能有速率限制，请合理使用
- 生产环境建议使用HTTPS和适当的认证机制
- 配置文件支持热重载，修改配置后重启服务生效
- Channel绑定后，所有Session消息都会自动转发到绑定的渠道