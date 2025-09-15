# SSH 隧道管理系统

一个基于Go语言开发的SSH隧道管理系统，支持用户管理、连接记录、流量统计和防火墙规则等功能。

## 功能特性

- SSH服务器：支持SSH隧道连接
- 用户管理：添加、激活/停用用户
- 连接记录：记录所有SSH连接和目标连接
- 流量统计：实时统计上行和下行流量
- 防火墙规则：支持目标地址白名单和黑名单
- Web管理界面：友好的Web界面进行管理操作
- Web管理界面密码保护：为Web管理界面添加基础认证保护

## 目录结构

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

## 安装与运行

### 环境要求

- Go 1.16 或更高版本
- SQLite3 数据库

### 安装步骤

1. 克隆项目代码：
```bash
git clone <repository-url>
cd ssh-manage
```

2. 初始化Go模块：
```bash
go mod tidy
```

3. 编译程序：
```bash
go build -o ssh-manage .
```

4. 运行程序：
```bash
./ssh-manage
```

### 默认配置

- SSH服务端口：53322
- Web管理界面端口：53380
- 默认管理员账号：admin/admin123

## 使用说明

### 启动后操作

1. 访问Web管理界面：http://localhost:53380
2. 使用默认管理员账号登录 (用户名: admin, 密码: admin123)
3. 可以添加新用户、查看连接记录、设置防火墙规则等

### SSH客户端连接

使用SSH客户端连接到服务器：
```bash
ssh -p 53322 username@server_ip
```

或者建立SSH隧道：
```bash
ssh -p 53322 -L local_port:target_host:target_port username@server_ip
```

### SOCKS5代理使用

SSH隧道可以作为SOCKS5代理使用，命令如下：
```bash
ssh -D 1080 -p 53322 -C -N username@server_ip
```

参数说明：
- `-D 1080`：在本地1080端口创建SOCKS5代理
- `-p 53322`：连接SSH服务器的53322端口
- `-C`：启用数据压缩
- `-N`：不执行远程命令，仅转发端口

配置浏览器或其他应用程序使用SOCKS5代理：
- 代理类型：SOCKS5
- 代理地址：127.0.0.1
- 代理端口：1080

## Web管理界面密码保护

为了提高安全性，Web管理界面现在支持基础认证保护。默认情况下，使用以下凭据登录：

- 用户名：admin
- 密码：admin123

### 自定义认证凭据

可以通过以下方式自定义认证凭据：

1. 通过环境变量设置：
   ```bash
   export WEB_USERNAME=myuser
   export WEB_PASSWORD=mypassword
   ./ssh-manage
   ```

2. 或者在启动时临时设置：
   ```bash
   WEB_USERNAME=myuser WEB_PASSWORD=mypassword ./ssh-manage
   ```

注意：如果将用户名或密码设置为空字符串，则会禁用认证功能。

## 功能模块说明

### 用户管理

- 支持添加新用户
- 可以激活或停用用户
- 用户状态影响SSH连接权限

### 连接记录

- 记录所有SSH连接信息
- 记录每个SSH连接的目标地址连接
- 显示连接时间、断开时间等信息

### 流量统计

- 实时统计每个连接的上行和下行流量
- 在Web界面展示流量统计图表

### 防火墙规则

- 支持白名单模式：仅允许匹配规则的目标地址
- 支持黑名单模式：禁止匹配规则的目标地址
- 白名单优先级高于黑名单
- 使用正则表达式匹配目标地址

## 技术架构

### 后端技术栈

- Go语言
- SQLite数据库
- SSH协议库 (golang.org/x/crypto/ssh)

### 前端技术栈

- Bootstrap 5
- Chart.js (用于数据可视化)

## 开发说明

### 代码结构

项目采用分层架构设计：
- `api/` - SSH服务器实现
- `web/` - Web界面处理
- `services/` - 业务逻辑层
- `utils/` - 工具函数和数据库操作
- `models/` - 数据模型定义
- `config/` - 配置管理

### 数据库设计

使用SQLite数据库存储数据，包含以下表：
- `users` - 用户信息表
- `connections` - SSH连接记录表
- `target_connections` - 目标连接记录表
- `firewall_rules` - 防火墙规则表

## 安全说明

- 所有SSH连接都经过用户认证
- 支持通过防火墙规则限制目标地址访问
- 用户密码以明文形式存储
- Web管理界面支持基础认证保护

## 图片预览

<img width="1700" height="762" alt="PixPin_2025-09-15_16-05-31" src="https://github.com/user-attachments/assets/3bef0ce2-f4a6-471e-bdee-61fd40135ecb" />
<img width="1718" height="795" alt="PixPin_2025-09-15_16-05-18" src="https://github.com/user-attachments/assets/5f31004b-de28-4d51-a0bc-6e43230d0836" />
<img width="1669" height="1182" alt="PixPin_2025-09-15_16-03-55" src="https://github.com/user-attachments/assets/58d8b347-cf86-4a0f-8798-040d44f5e86e" />
<img width="1727" height="936" alt="PixPin_2025-09-15_16-03-36" src="https://github.com/user-attachments/assets/3cab5b35-d633-4b65-bc23-3b3d8b097d34" />



## 许可证

MIT License
