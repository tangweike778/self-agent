package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// cfg 全局配置
var cfg *Config

// FeishuCfg 飞书配置
type FeishuCfg struct {
	Webhook string `yaml:"default_webhook"`
	Secret  string `yaml:"default_secret"`
	AppID   string `yaml:"app_id"`
	OpenID  string `yaml:"open_id"`
}

// ChannelCfg 渠道配置
type ChannelCfg struct {
	Feishu FeishuCfg `yaml:"feishu"`
}

// Config 应用配置结构体
type Config struct {
	Deepseek struct {
		APIKey    string `yaml:"api_key"`
		MaxTokens int64  `yaml:"max_tokens"`
	} `yaml:"deepseek"`
	Server struct {
		Port      int   `yaml:"port"`
		MaxTokens int64 `yaml:"max_tokens"`
	} `yaml:"server"`
	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logging"`
	Channel ChannelCfg `yaml:"channel"`
}

// GetConfig 获取配置
func GetConfig() *Config {
	if cfg == nil {
		cfg = LoadConfigWithDefaults()
	}
	return cfg
}

// LoadConfig 从YAML文件加载配置
func LoadConfig() (*Config, error) {
	// 读取配置文件
	data, err := os.ReadFile("config/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析YAML配置
	cfg = &Config{}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, fmt.Errorf("解析YAML配置失败: %v", err)
	}

	// 检查必要的配置项
	if cfg.Deepseek.APIKey == "your-deepseek-api-key-here" {
		return nil, fmt.Errorf("请配置有效的Deepseek API密钥")
	}

	return cfg, nil
}

// LoadConfigWithDefaults 加载配置，如果失败则使用默认值
func LoadConfigWithDefaults() *Config {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Printf("配置加载失败，使用默认配置: %v\n", err)
		return &Config{
			Server: struct {
				Port      int   `yaml:"port"`
				MaxTokens int64 `yaml:"max_tokens"`
			}{Port: 19100, MaxTokens: 4096},
			Logging: struct {
				Level  string `yaml:"level"`
				Format string `yaml:"format"`
			}{Level: "info", Format: "text"},
			Channel: ChannelCfg{Feishu: FeishuCfg{
				Webhook: "",
				Secret:  "",
				AppID:   "",
			}},
		}
	}
	return cfg
}

// HasFeishuConfig 检查是否配置了飞书机器人
func (c *Config) HasFeishuConfig() bool {
	return c.Channel.Feishu.Webhook != ""
}

// GetFeishuWebhook 获取飞书webhook地址
func (c *Config) GetFeishuWebhook() string {
	return c.Channel.Feishu.Webhook
}

// GetFeishuSecret 获取飞书secret
func (c *Config) GetFeishuSecret() string {
	return c.Channel.Feishu.Secret
}

// GetAppID 获取飞书app_id
func (c *Config) GetAppID() string {
	return c.Channel.Feishu.AppID
}
