# AI编程指南

本指南旨在帮助AI助手更好地理解和维护此SSH隧道管理系统项目。

## 项目概述

这是一个基于Go语言开发的SSH隧道管理系统，提供用户管理、连接记录、流量统计和防火墙规则等功能。项目采用模块化设计，具有良好的可扩展性和可维护性。

## 技术栈

### 后端技术
- **语言**: Go (Golang)
- **数据库**: SQLite
- **SSH库**: golang.org/x/crypto/ssh
- **Web框架**: 标准库net/http

### 前端技术
- **HTML/CSS/JavaScript**: 原生Web技术
- **Bootstrap**: 5.3.0版本，用于UI组件和响应式设计
- **Chart.js**: 用于数据可视化

## 项目架构

项目采用分层架构设计，各层职责明确：

```
ssh-manage/
├── api/           # SSH服务器相关代码
├── config/        # 配置文件
├── data/          # 数据文件（数据库、密钥等）
├── models/        # 数据模型
├── services/      # 业务逻辑层
├── utils/         # 工具函数
├── web/           # Web界面相关代码
├── main.go        # 程序入口
├── go.mod         # Go模块文件
└── README.md      # 项目说明文档
```

### 各模块功能说明

#### api包
SSH服务器核心实现，处理SSH连接、认证、通道管理等。

关键组件：
- `StartSSHServer`: 启动SSH服务器
- `handleConnection`: 处理SSH连接
- `handleDirectTCPIPChannel`: 处理TCP/IP隧道连接
- `updateTargetTraffic`: 更新流量统计

#### config包
应用配置管理。

关键组件：
- `Config`: 配置结构体
- `Load`: 加载配置

#### models包
数据模型定义。

关键组件：
- `User`: 用户模型
- `Connection`: SSH连接模型
- `TargetConnection`: 目标连接模型
- `FirewallRule`: 防火墙规则模型

#### services包
业务逻辑层，处理应用的核心业务逻辑。

关键组件：
- `AuthenticateUser`: 用户认证
- `GetAllUsers`: 获取所有用户
- `GetStatistics`: 获取统计信息

#### utils包
工具函数，包括数据库操作和防火墙功能。

关键组件：
- `InitDB`: 初始化数据库
- `GetUserByUsername`: 根据用户名获取用户
- `IsAddressAllowed`: 检查地址是否被防火墙允许

#### web包
Web界面处理。

关键组件：
- `Handler`: Web请求处理函数
- `serveUsersPage`: 用户管理页面
- `serveConnectionsPage`: 连接记录页面
- `serveStatsPage`: 统计数据页面
- `serveFirewallPage`: 防火墙规则页面

## 数据库设计

使用SQLite数据库存储数据，包含以下表：

### users表
存储用户信息
- id: 用户ID (主键)
- name: 昵称
- username: 用户名 (唯一)
- password: 密码
- created: 创建时间
- active: 是否激活

### connections表
存储SSH连接记录
- id: 连接ID (主键)
- user_id: 用户ID (外键)
- username: 用户名
- ip: 客户端IP地址
- connected_at: 连接时间
- disconnected_at: 断开时间
- session_id: 会话ID (唯一)

### target_connections表
存储目标连接记录
- id: 目标连接ID (主键)
- connection_id: SSH连接ID (外键)
- target: 目标地址
- connected_at: 连接时间
- disconnected_at: 断开时间
- bytes_up: 上行流量
- bytes_down: 下行流量

### firewall_rules表
存储防火墙规则
- id: 规则ID (主键)
- type: 规则类型 (whitelist/blacklist)
- pattern: 正则表达式模式
- active: 是否激活

## 核心功能实现

### SSH服务器
SSH服务器基于golang.org/x/crypto/ssh库实现，支持密码认证和TCP/IP隧道。

工作流程：
1. 启动SSH服务器并监听指定端口
2. 接受客户端连接并进行密码认证
3. 认证成功后记录连接信息到数据库
4. 处理客户端请求的通道类型
5. 对于direct-tcpip通道，建立到目标地址的连接并转发数据
6. 实时统计并定期更新流量信息

### 用户管理
通过Web界面管理用户，支持添加用户、激活/停用用户。

### 连接记录
记录所有SSH连接和目标连接信息，支持按用户筛选和分页查看。

### 流量统计
实时统计每个连接的上行和下行流量，并在Web界面以图表形式展示。

### 防火墙规则
支持白名单和黑名单模式，使用正则表达式匹配目标地址。

工作原理：
1. 如果没有规则，允许所有流量
2. 如果有白名单规则，仅允许匹配白名单的流量
3. 如果只有黑名单规则，拒绝匹配黑名单的流量
4. 白名单优先级高于黑名单

## 编程规范

### 命名规范
- 使用驼峰命名法
- 函数名使用动词开头
- 变量名具有描述性
- 常量名全部大写

### 注释规范
- 每个函数都需要注释说明功能、参数和返回值
- 复杂逻辑需要添加行内注释
- 结构体字段需要注释说明用途

### 错误处理
- 所有错误都需要处理
- 错误信息需要记录到日志
- 对于Web请求，需要返回适当的HTTP状态码

### 并发安全
- 使用互斥锁保护共享资源
- 避免数据竞争
- 合理使用goroutine

## 常见问题和解决方案

### 数据库初始化失败
问题：数据库迁移时出现"no such column"错误。
解决方案：在迁移数据前检查字段是否存在。

### SSH连接崩溃
问题：密码错误时服务崩溃。
解决方案：正确处理认证失败的情况，返回错误而不是继续执行。

### Web界面布局问题
问题：表单元素布局不合理。
解决方案：使用CSS Flexbox和Bootstrap栅格系统优化布局。

## 扩展建议

1. **安全性增强**：
   - 密码加密存储
   - 添加双因素认证
   - 实现更细粒度的权限控制

2. **功能扩展**：
   - 添加连接限制（并发连接数、带宽限制等）
   - 支持公钥认证
   - 添加日志审计功能

3. **性能优化**：
   - 添加数据库索引
   - 实现连接池
   - 优化流量统计更新频率

4. **用户体验**：
   - 添加多语言支持
   - 实现响应式设计优化
   - 添加数据导出功能

## 贡献指南

1. Fork项目
2. 创建功能分支
3. 提交代码更改
4. 发起Pull Request

## 许可证

MIT License