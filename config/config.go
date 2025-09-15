package config

import (
	"path/filepath"
	"os"
)

// Config 应用配置结构体
type Config struct {
	SSHPort  string // SSH服务端口
	WebPort  string // Web服务端口
	DBPath   string // 数据库文件路径
	LogPath  string // 日志文件路径
}

// Load 加载应用配置
// 返回: *Config - 配置信息
func Load() *Config {
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	
	return &Config{
		SSHPort: "53322",                              // 默认SSH端口
		WebPort: "8080",                               // 默认Web端口
		DBPath:  filepath.Join(wd, "data", "ssh_manage.db"), // 数据库路径
		LogPath: filepath.Join(wd, "logs", "ssh_manage.log"), // 日志路径
	}
}