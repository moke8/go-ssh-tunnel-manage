package models

import "time"

// User 用户模型
type User struct {
	ID       int       `json:"id"`       // 用户ID
	Name     string    `json:"name"`     // 昵称
	Username string    `json:"username"` // 用户名
	Password string    `json:"password"` // 密码
	Created  time.Time // 创建时间
	Active   bool      // 是否激活
}

// Connection 连接记录模型
type Connection struct {
	ID             int        // 连接ID
	UserID         int        // 用户ID
	Username       string     // 用户名
	IP             string     // 客户端IP地址
	ConnectedAt    time.Time  // 连接时间
	DisconnectedAt *time.Time // 断开时间
	SessionID      string     // 会话ID
}

// TargetConnection 目标连接记录模型
type TargetConnection struct {
	ID             int        // 目标连接ID
	ConnectionID   int        // SSH连接ID
	Target         string     // 目标地址
	ConnectedAt    time.Time  // 连接时间
	DisconnectedAt *time.Time // 断开时间
	BytesUp        int64      // 上行流量（字节）
	BytesDown      int64      // 下行流量（字节）
}

// FirewallRule 防火墙规则模型
type FirewallRule struct {
	ID      int    // 规则ID
	Type    string // 规则类型："whitelist"（白名单）或"blacklist"（黑名单）
	Pattern string // 正则表达式模式
	Active  bool   // 是否激活
}