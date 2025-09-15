package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"
	"ssh-manage/models"
	"ssh-manage/services"
	"ssh-manage/utils"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		if r.Method == "POST" {
			serveUsersPage(w, r)
		} else {
			serveUsersPage(w, r)
		}
	case "/connections":
		serveConnectionsPage(w, r)
	case "/stats":
		serveStatsPage(w, r)
	case "/firewall":
		if r.Method == "POST" {
			serveFirewallPage(w, r)
		} else {
			serveFirewallPage(w, r)
		}
	default:
		http.NotFound(w, r)
	}
}

func serveUsersPage(w http.ResponseWriter, r *http.Request) {
	// 处理表单提交
	if r.Method == "POST" {
		action := r.FormValue("action")
		
		switch action {
		case "add_user":
			// 处理增加用户
			name := r.FormValue("name")
			username := r.FormValue("username")
			password := r.FormValue("password")
			activeStr := r.FormValue("active")
			
			if name != "" && username != "" && password != "" {
				active := activeStr == "true"
				
				user := &models.User{
					Name:     name,
					Username: username,
					Password: password,
					Active:   active,
					Created:  time.Now(),
				}
				
				err := services.AddUser(user)
				if err != nil {
					log.Printf("Failed to add user: %v", err)
				}
			}
			
		case "toggle_active":
			// 处理切换用户激活状态
			userIDStr := r.FormValue("user_id")
			if userIDStr != "" {
				if userID, err := strconv.Atoi(userIDStr); err == nil {
					user := services.GetUserByID(userID)
					if user != nil {
						user.Active = !user.Active
						services.UpdateUser(user)
					}
				}
			}
		}
		
		// 重定向以避免重复提交
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	
	users := services.GetAllUsers()
	
	data := struct {
		Users []*models.User
	}{
		Users: users,
	}
	
	tmpl := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SSH隧道用户管理</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        body { padding: 20px 0; }
        .user-form .form-control, .user-form .form-select {
            width: 100%;
            max-width: 300px;
        }
        .user-form .form-label {
            font-weight: bold;
            display: inline-block;
            margin-right: 10px;
            min-width: 60px;
            width: auto;
        }
        .user-form .form-group {
            display: flex;
            align-items: center;
            margin-bottom: 10px;
            flex-wrap: nowrap;
        }
        @media (min-width: 768px) {
            .user-form .col-md-6 {
                flex: 0 0 auto;
                width: auto;
                margin-right: 20px;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1 class="text-center mb-4">SSH隧道用户管理</h1>
        
        <ul class="nav nav-tabs mb-4">
            <li class="nav-item">
                <a class="nav-link active" href="/">用户管理</a>
            </li>
            <li class="nav-item">
                <a class="nav-link" href="/connections">连接记录</a>
            </li>
            <li class="nav-item">
                <a class="nav-link" href="/stats">统计数据</a>
            </li>
            <li class="nav-item">
                <a class="nav-link" href="/firewall">防火墙规则</a>
            </li>
        </ul>
        
        <!-- 增加用户表单 -->
        <div class="card mb-4">
            <div class="card-header">
                <h5 class="mb-0">增加用户</h5>
            </div>
            <div class="card-body">
                <form method="POST" class="user-form">
                    <div class="row">
                        <div class="col-md-6">
                            <div class="form-group">
                                <label for="name" class="form-label">昵称</label>
                                <input type="text" class="form-control" id="name" name="name" required>
                            </div>
                        </div>
                        <div class="col-md-6">
                            <div class="form-group">
                                <label for="username" class="form-label">用户名</label>
                                <input type="text" class="form-control" id="username" name="username" required>
                            </div>
                        </div>
                        <div class="col-md-6">
                            <div class="form-group">
                                <label for="password" class="form-label">密码</label>
                                <input type="password" class="form-control" id="password" name="password" required>
                            </div>
                        </div>
                        <div class="col-md-6">
                            <div class="form-group">
                                <label for="active" class="form-label">状态</label>
                                <select class="form-select" id="active" name="active">
                                    <option value="true">激活</option>
                                    <option value="false">未激活</option>
                                </select>
                            </div>
                        </div>
                    </div>
                    <div class="col-12 mt-3">
                        <div class="d-flex justify-content-end">
                            <button type="submit" class="btn btn-primary" name="action" value="add_user">添加用户</button>
                        </div>
                    </div>
                </form>
            </div>
        </div>
        
        <div class="card">
            <div class="card-header">
                <h5 class="mb-0">用户列表</h5>
            </div>
            <div class="card-body">
                <div class="table-responsive">
                    <table class="table table-striped table-hover">
                        <thead class="table-dark">
                            <tr>
                                <th>ID</th>
                                <th>昵称</th>
                                <th>用户名</th>
                                <th>创建时间</th>
                                <th>状态</th>
                            </tr>
                        </thead>
                        <tbody>
                            {{range .Users}}
                            <tr>
                                <td>{{.ID}}</td>
                                <td>{{.Name}}</td>
                                <td>{{.Username}}</td>
                                <td>{{.Created.Format "2006-01-02 15:04:05"}}</td>
                                <td>
                                    <form method="POST" style="display: inline;">
                                        <input type="hidden" name="user_id" value="{{.ID}}">
                                        <button type="submit" name="action" value="toggle_active" class="btn btn-sm {{if .Active}}btn-success{{else}}btn-secondary{{end}}">
                                            {{if .Active}}激活{{else}}未激活{{end}}
                                        </button>
                                    </form>
                                </td>
                            </tr>
                            {{end}}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </div>
    
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>
`
	
	t, _ := template.New("users").Parse(tmpl)
	t.Execute(w, data)
}

func serveConnectionsPage(w http.ResponseWriter, r *http.Request) {
	// 获取查询参数中的用户ID和页码
	userIDStr := r.URL.Query().Get("user_id")
	pageStr := r.URL.Query().Get("page")
	
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	
	// 每页显示的记录数
	pageSize := 20
	offset := (page - 1) * pageSize
	
	var targetConnections []*models.TargetConnection
	var sshConnections []*models.Connection
	var totalConnections int
	
	if userIDStr != "" {
		// 如果指定了用户ID，则只显示该用户的目标连接记录
		userID, err := strconv.Atoi(userIDStr)
		if err == nil {
			targetConnections = services.GetTargetConnectionsByUserID(userID)
			sshConnections = services.GetConnectionsByUserID(userID)
			totalConnections = len(targetConnections)
			
			// 应用分页
			start := offset
			end := start + pageSize
			if start < totalConnections {
				if end > totalConnections {
					end = totalConnections
				}
				targetConnections = targetConnections[start:end]
			} else {
				targetConnections = []*models.TargetConnection{}
			}
		} else {
			targetConnections = services.GetAllTargetConnections()
			sshConnections = services.GetAllConnections()
			totalConnections = len(targetConnections)
			
			// 应用分页
			start := offset
			end := start + pageSize
			if start < totalConnections {
				if end > totalConnections {
					end = totalConnections
				}
				targetConnections = targetConnections[start:end]
			} else {
				targetConnections = []*models.TargetConnection{}
			}
		}
	} else {
		// 否则显示所有目标连接记录
		allTargetConnections := services.GetAllTargetConnections()
		sshConnections = services.GetAllConnections()
		totalConnections = len(allTargetConnections)
		
		// 应用分页
		start := offset
		end := start + pageSize
		if start < totalConnections {
			if end > totalConnections {
				end = totalConnections
			}
			targetConnections = allTargetConnections[start:end]
		} else {
			targetConnections = []*models.TargetConnection{}
		}
	}
	
	// 创建SSH连接映射，方便通过ID查找
	sshConnectionMap := make(map[int]*models.Connection)
	for _, conn := range sshConnections {
		sshConnectionMap[conn.ID] = conn
	}
	
	// 获取所有用户用于筛选下拉框
	users := services.GetAllUsers()
	
	// 计算总页数
	totalPages := (totalConnections + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	
	data := struct {
		TargetConnections []*models.TargetConnection
		SSHConnections    map[int]*models.Connection
		Users             []*models.User
		SelectedUserID    string
		CurrentPage       int
		TotalPages        int
		TotalConnections  int
		PageSize          int
	}{
		TargetConnections: targetConnections,
		SSHConnections:    sshConnectionMap,
		Users:             users,
		SelectedUserID:    userIDStr,
		CurrentPage:       page,
		TotalPages:        totalPages,
		TotalConnections:  totalConnections,
		PageSize:          pageSize,
	}
	
	tmpl := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SSH隧道连接记录</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        body { padding: 20px 0; }
        .pagination { justify-content: center; }
        .table th, .table td { 
            white-space: nowrap; 
            text-align: center;
        }
        .table th:nth-child(1), .table td:nth-child(1) { width: 8%; }  /* SSH连接ID */
        .table th:nth-child(2), .table td:nth-child(2) { width: 10%; } /* 用户名 */
        .table th:nth-child(3), .table td:nth-child(3) { width: 12%; } /* 客户端IP */
        .table th:nth-child(4), .table td:nth-child(4) { width: 15%; } /* 目标地址 */
        .table th:nth-child(5), .table td:nth-child(5) { width: 15%; } /* 连接时间 */
        .table th:nth-child(6), .table td:nth-child(6) { width: 15%; } /* 断开时间 */
        .table th:nth-child(7), .table td:nth-child(7) { width: 10%; } /* 上行流量 */
        .table th:nth-child(8), .table td:nth-child(8) { width: 10%; } /* 下行流量 */
    </style>
</head>
<body>
    <div class="container">
        <h1 class="text-center mb-4">SSH隧道连接记录</h1>
        
        <ul class="nav nav-tabs mb-4">
            <li class="nav-item">
                <a class="nav-link" href="/">用户管理</a>
            </li>
            <li class="nav-item">
                <a class="nav-link active" href="/connections">连接记录</a>
            </li>
            <li class="nav-item">
                <a class="nav-link" href="/stats">统计数据</a>
            </li>
            <li class="nav-item">
                <a class="nav-link" href="/firewall">防火墙规则</a>
            </li>
        </ul>
        
        <div class="row mb-3">
            <div class="col-md-4">
                <form method="GET" class="row g-3">
                    <div class="col">
                        <label for="user_id" class="form-label">用户</label>
                        <select class="form-select" id="user_id" name="user_id">
                            <option value="">所有用户</option>
                            {{range .Users}}
                                <option value="{{.ID}}" {{if eq $.SelectedUserID (printf "%d" .ID)}}selected{{end}}>{{.Name}} ({{.Username}})</option>
                            {{end}}
                        </select>
                    </div>
                    <div class="col-auto d-flex align-items-end">
                        <button type="submit" class="btn btn-primary">筛选</button>
                    </div>
                </form>
            </div>
        </div>
        
        <div class="card">
            <div class="card-header">
                <h5 class="mb-0">连接记录列表</h5>
            </div>
            <div class="card-body">
                <div class="table-responsive">
                    <table class="table table-striped table-hover">
                        <thead class="table-dark">
                            <tr>
                                <th>SSH连接ID</th>
                                <th>用户名</th>
                                <th>客户端IP</th>
                                <th>目标地址</th>
                                <th>连接时间</th>
                                <th>断开时间</th>
                                <th>上行流量</th>
                                <th>下行流量</th>
                            </tr>
                        </thead>
                        <tbody>
                            {{range .TargetConnections}}
                            <tr>
                                <td>{{.ConnectionID}}</td>
                                <td>{{(index $.SSHConnections .ConnectionID).Username}}</td>
                                <td>{{(index $.SSHConnections .ConnectionID).IP}}</td>
                                <td>{{.Target}}</td>
                                <td>{{.ConnectedAt.Format "2006-01-02 15:04:05"}}</td>
                                <td>
                                    {{if .DisconnectedAt}}
                                        {{.DisconnectedAt.Format "2006-01-02 15:04:05"}}
                                    {{else}}
                                        未断开
                                    {{end}}
                                </td>
                                <td>{{formatBytes .BytesUp}}</td>
                                <td>{{formatBytes .BytesDown}}</td>
                            </tr>
                            {{else}}
                            <tr>
                                <td colspan="8" class="text-center">暂无连接记录</td>
                            </tr>
                            {{end}}
                        </tbody>
                    </table>
                </div>
                
                <!-- 分页 -->
                {{if gt .TotalPages 1}}
                <nav aria-label="分页导航">
                    <ul class="pagination">
                        <!-- 上一页 -->
                        {{if gt .CurrentPage 1}}
                        <li class="page-item">
                            <a class="page-link" href="?page={{sub .CurrentPage 1}}{{if ne .SelectedUserID ""}}&user_id={{.SelectedUserID}}{{end}}" aria-label="上一页">
                                <span aria-hidden="true">&laquo;</span>
                            </a>
                        </li>
                        {{else}}
                        <li class="page-item disabled">
                            <span class="page-link">&laquo;</span>
                        </li>
                        {{end}}
                        
                        <!-- 页码 -->
                        {{range $i := until .TotalPages}}
                        {{$pageNum := add $i 1}}
                        {{if eq $pageNum $.CurrentPage}}
                        <li class="page-item active" aria-current="page">
                            <span class="page-link">{{$pageNum}}</span>
                        </li>
                        {{else}}
                        <li class="page-item">
                            <a class="page-link" href="?page={{$pageNum}}{{if ne $.SelectedUserID ""}}&user_id={{$.SelectedUserID}}{{end}}">{{$pageNum}}</a>
                        </li>
                        {{end}}
                        {{end}}
                        
                        <!-- 下一页 -->
                        {{if lt .CurrentPage .TotalPages}}
                        <li class="page-item">
                            <a class="page-link" href="?page={{add .CurrentPage 1}}{{if ne .SelectedUserID ""}}&user_id={{.SelectedUserID}}{{end}}" aria-label="下一页">
                                <span aria-hidden="true">&raquo;</span>
                            </a>
                        </li>
                        {{else}}
                        <li class="page-item disabled">
                            <span class="page-link">&raquo;</span>
                        </li>
                        {{end}}
                    </ul>
                </nav>
                <div class="text-center text-muted">
                    共 {{.TotalConnections}} 条记录，第 {{.CurrentPage}} 页，共 {{.TotalPages}} 页
                </div>
                {{end}}
            </div>
        </div>
    </div>
    
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>
`
	
	// 定义模板函数
	funcMap := template.FuncMap{
		"formatBytes": formatBytes,
		"until": func(n int) []int {
			result := make([]int, n)
			for i := range result {
				result[i] = i
			}
			return result
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
	}
	
	t, _ := template.New("connections").Funcs(funcMap).Parse(tmpl)
	t.Execute(w, data)
}

func serveFirewallPage(w http.ResponseWriter, r *http.Request) {
	// 处理表单提交
	if r.Method == "POST" {
		action := r.FormValue("action")
		
		switch action {
		case "add_rule":
			// 添加防火墙规则
			ruleType := r.FormValue("rule_type")
			pattern := r.FormValue("pattern")
			
			if (ruleType == "whitelist" || ruleType == "blacklist") && pattern != "" {
				err := utils.AddFirewallRule(ruleType, pattern)
				if err != nil {
					log.Printf("Failed to add firewall rule: %v", err)
				}
			}
			
		case "delete_rule":
			// 删除防火墙规则
			ruleIDStr := r.FormValue("rule_id")
			if ruleIDStr != "" {
				if ruleID, err := strconv.Atoi(ruleIDStr); err == nil {
					err := utils.DeleteFirewallRule(ruleID)
					if err != nil {
						log.Printf("Failed to delete firewall rule: %v", err)
					}
				}
			}
		}
		
		// 重定向以避免重复提交
		http.Redirect(w, r, "/firewall", http.StatusSeeOther)
		return
	}
	
	// 获取所有防火墙规则
	rules, err := utils.GetFirewallRules()
	if err != nil {
		log.Printf("Failed to get firewall rules: %v", err)
		rules = []*utils.FirewallRule{} // 空列表
	}
	
	data := struct {
		Rules []*utils.FirewallRule
	}{
		Rules: rules,
	}
	
	tmpl := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SSH隧道防火墙规则管理</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        body { padding: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1 class="text-center mb-4">SSH隧道防火墙规则管理</h1>
        
        <ul class="nav nav-tabs mb-4">
            <li class="nav-item">
                <a class="nav-link" href="/">用户管理</a>
            </li>
            <li class="nav-item">
                <a class="nav-link" href="/connections">连接记录</a>
            </li>
            <li class="nav-item">
                <a class="nav-link" href="/stats">统计数据</a>
            </li>
            <li class="nav-item">
                <a class="nav-link active" href="/firewall">防火墙规则</a>
            </li>
        </ul>
        
        <!-- 添加规则表单 -->
        <div class="card mb-4">
            <div class="card-header">
                <h5 class="mb-0">添加防火墙规则</h5>
            </div>
            <div class="card-body">
                <form method="POST" class="row g-3">
                    <div class="col-md-4">
                        <label for="rule_type" class="form-label">规则类型</label>
                        <select class="form-select" id="rule_type" name="rule_type" required>
                            <option value="">请选择规则类型</option>
                            <option value="whitelist">白名单</option>
                            <option value="blacklist">黑名单</option>
                        </select>
                    </div>
                    <div class="col-md-6">
                        <label for="pattern" class="form-label">目标地址模式（正则表达式）</label>
                        <input type="text" class="form-control" id="pattern" name="pattern" placeholder="例如: .*\.google\.com:443" required>
                    </div>
                    <div class="col-md-2">
                        <label class="form-label">&nbsp;</label>
                        <button type="submit" class="btn btn-primary form-control" name="action" value="add_rule">添加规则</button>
                    </div>
                </form>
                <div class="form-text">
                    <p class="mb-1"><strong>使用说明：</strong></p>
                    <ul>
                        <li>如果未设置任何规则，则允许所有流量代理</li>
                        <li>如果设置了白名单（正则），则仅允许匹配白名单的目标地址转发</li>
                        <li>如果设置了黑名单（正则），则不允许匹配黑名单的目标地址转发</li>
                        <li>白名单优先级高于黑名单</li>
                    </ul>
                </div>
            </div>
        </div>
        
        <!-- 规则列表 -->
        <div class="card">
            <div class="card-header">
                <h5 class="mb-0">防火墙规则列表</h5>
            </div>
            <div class="card-body">
                <div class="table-responsive">
                    <table class="table table-striped table-hover">
                        <thead class="table-dark">
                            <tr>
                                <th>ID</th>
                                <th>类型</th>
                                <th>模式</th>
                                <th>操作</th>
                            </tr>
                        </thead>
                        <tbody>
                            {{range .Rules}}
                            <tr>
                                <td>{{.ID}}</td>
                                <td>
                                    {{if eq .Type "whitelist"}}
                                        <span class="badge bg-success">白名单</span>
                                    {{else}}
                                        <span class="badge bg-danger">黑名单</span>
                                    {{end}}
                                </td>
                                <td>{{.Pattern}}</td>
                                <td>
                                    <form method="POST" style="display: inline;">
                                        <input type="hidden" name="rule_id" value="{{.ID}}">
                                        <button type="submit" name="action" value="delete_rule" class="btn btn-sm btn-danger" 
                                            onclick="return confirm('确定要删除这条规则吗？')">删除</button>
                                    </form>
                                </td>
                            </tr>
                            {{else}}
                            <tr>
                                <td colspan="4" class="text-center">暂无防火墙规则</td>
                            </tr>
                            {{end}}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </div>
    
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>
`
	
	t, _ := template.New("firewall").Parse(tmpl)
	t.Execute(w, data)
}

func serveStatsPage(w http.ResponseWriter, r *http.Request) {
	stats := services.GetStatistics()
	connections := services.GetAllConnections()
	users := services.GetAllUsers()
	
	// 准备图表数据
	chartData := prepareChartData(connections)
	userChartData := prepareUserChartData(connections, users)
	
	tmpl := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SSH隧道统计信息</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { padding: 20px 0; }
        .chart-container { position: relative; height: 300px; }
    </style>
</head>
<body>
    <div class="container">
        <h1 class="text-center mb-4">SSH隧道统计信息</h1>
        
        <ul class="nav nav-tabs mb-4">
            <li class="nav-item">
                <a class="nav-link" href="/">用户管理</a>
            </li>
            <li class="nav-item">
                <a class="nav-link" href="/connections">连接记录</a>
            </li>
            <li class="nav-item">
                <a class="nav-link active" href="/stats">统计数据</a>
            </li>
            <li class="nav-item">
                <a class="nav-link" href="/firewall">防火墙规则</a>
            </li>
        </ul>
        
        <div class="row">
            <div class="col-md-6 mb-4">
                <div class="card">
                    <div class="card-header">
                        <h5 class="mb-0">总体统计</h5>
                    </div>
                    <div class="card-body">
                        <div class="row">
                            <div class="col-md-6 mb-3">
                                <div class="card bg-primary text-white">
                                    <div class="card-body text-center">
                                        <h6 class="card-title">总连接数</h6>
                                        <h3>` + fmt.Sprintf("%v", stats["total_connections"]) + `</h3>
                                    </div>
                                </div>
                            </div>
                            <div class="col-md-6 mb-3">
                                <div class="card bg-success text-white">
                                    <div class="card-body text-center">
                                        <h6 class="card-title">活跃用户数</h6>
                                        <h3>` + fmt.Sprintf("%v", stats["active_users"]) + `</h3>
                                    </div>
                                </div>
                            </div>
                            <div class="col-md-6 mb-3">
                                <div class="card bg-info text-white">
                                    <div class="card-body text-center">
                                        <h6 class="card-title">上行流量</h6>
                                        <h3>` + formatBytes(stats["total_traffic_up"].(int64)) + `</h3>
                                    </div>
                                </div>
                            </div>
                            <div class="col-md-6 mb-3">
                                <div class="card bg-warning text-white">
                                    <div class="card-body text-center">
                                        <h6 class="card-title">下行流量</h6>
                                        <h3>` + formatBytes(stats["total_traffic_down"].(int64)) + `</h3>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            
            <div class="col-md-6 mb-4">
                <div class="card">
                    <div class="card-header">
                        <h5 class="mb-0">连接趋势图</h5>
                    </div>
                    <div class="card-body">
                        <div class="chart-container">
                            <canvas id="connectionChart"></canvas>
                        </div>
                    </div>
                </div>
            </div>
            
            <div class="col-md-12 mb-4">
                <div class="card">
                    <div class="card-header">
                        <h5 class="mb-0">各用户连接趋势图</h5>
                    </div>
                    <div class="card-body">
                        <div class="chart-container">
                            <canvas id="userConnectionChart"></canvas>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
    
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js"></script>
    <script>
        // 总体连接趋势图数据
        const chartData = ` + chartData + `;
        
        // 各用户连接趋势图数据
        const userChartData = ` + userChartData + `;
        
        // 创建总体连接趋势图
        const ctx = document.getElementById('connectionChart').getContext('2d');
        const connectionChart = new Chart(ctx, {
            type: 'line',
            data: {
                labels: chartData.labels,
                datasets: [{
                    label: '连接数',
                    data: chartData.data,
                    borderColor: 'rgb(75, 192, 192)',
                    backgroundColor: 'rgba(75, 192, 192, 0.2)',
                    tension: 0.1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        ticks: {
                            precision: 0
                        }
                    }
                }
            }
        });
        
        // 创建各用户连接趋势图
        const userCtx = document.getElementById('userConnectionChart').getContext('2d');
        const userConnectionChart = new Chart(userCtx, {
            type: 'line',
            data: {
                labels: userChartData.labels,
                datasets: userChartData.datasets
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        ticks: {
                            precision: 0
                        }
                    }
                }
            }
        });
    </script>
</body>
</html>
`
	
	t, _ := template.New("stats").Parse(tmpl)
	t.Execute(w, nil)
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)
	
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

type ChartData struct {
	Labels []string `json:"labels"`
	Data   []int    `json:"data"`
}

type UserChartData struct {
	Labels   []string            `json:"labels"`
	Datasets []map[string]interface{} `json:"datasets"`
}

func prepareChartData(connections []*models.Connection) string {
	// 按日期统计连接数
	connectionCountByDate := make(map[string]int)
	
	// 统计最近7天的连接数
	now := time.Now()
	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("01-02") // 只显示月日
		connectionCountByDate[date] = 0
	}
	
	// 遍历连接记录，统计每天的连接数
	for _, conn := range connections {
		date := conn.ConnectedAt.Format("01-02")
		if _, exists := connectionCountByDate[date]; exists {
			connectionCountByDate[date]++
		}
	}
	
	// 按日期排序
	dates := make([]string, 0, len(connectionCountByDate))
	for date := range connectionCountByDate {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	
	// 构建图表数据
	chartData := ChartData{
		Labels: dates,
		Data:   make([]int, len(dates)),
	}
	
	for i, date := range dates {
		chartData.Data[i] = connectionCountByDate[date]
	}
	
	// 转换为JSON
	jsonData, _ := json.Marshal(chartData)
	return string(jsonData)
}

func prepareUserChartData(connections []*models.Connection, users []*models.User) string {
	// 定义一组颜色，确保不同用户有不同的颜色
	colors := []string{
		"255, 99, 132",   // 红色
		"54, 162, 235",   // 蓝色
		"255, 206, 86",   // 黄色
		"75, 192, 192",   // 青色
		"153, 102, 255",  // 紫色
		"255, 159, 64",   // 橙色
		"199, 199, 199",  // 灰色
		"83, 102, 255",   // 靛蓝色
		"255, 99, 255",   // 粉色
		"99, 255, 132",   // 绿色
	}
	
	// 按日期和用户统计连接数
	userConnectionCountByDate := make(map[string]map[int]int)
	
	// 创建用户映射
	userMap := make(map[int]string)
	for _, user := range users {
		userMap[user.ID] = user.Name
	}
	
	// 统计最近7天的连接数
	now := time.Now()
	dates := make([]string, 0)
	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format("01-02") // 只显示月日
		dates = append(dates, date)
		userConnectionCountByDate[date] = make(map[int]int)
		for userID := range userMap {
			userConnectionCountByDate[date][userID] = 0
		}
	}
	
	// 遍历连接记录，统计每天每个用户的连接数
	for _, conn := range connections {
		date := conn.ConnectedAt.Format("01-02")
		if _, exists := userConnectionCountByDate[date]; exists {
			userConnectionCountByDate[date][conn.UserID]++
		}
	}
	
	// 构建图表数据集
	datasets := make([]map[string]interface{}, 0)
	
	// 为每个用户创建数据集
	userIndex := 0
	for userID, userName := range userMap {
		// 为用户选择颜色
		colorIndex := userIndex % len(colors)
		borderColor := fmt.Sprintf("rgb(%s)", colors[colorIndex])
		backgroundColor := fmt.Sprintf("rgba(%s, 0.2)", colors[colorIndex])
		
		data := make([]int, len(dates))
		for i, date := range dates {
			data[i] = userConnectionCountByDate[date][userID]
		}
		
		dataset := map[string]interface{}{
			"label":           userName,
			"data":            data,
			"borderColor":     borderColor,
			"backgroundColor": backgroundColor,
			"tension":         0.1,
		}
		
		datasets = append(datasets, dataset)
		userIndex++
	}
	
	// 构建图表数据
	chartData := UserChartData{
		Labels:   dates,
		Datasets: datasets,
	}
	
	// 转换为JSON
	jsonData, _ := json.Marshal(chartData)
	return string(jsonData)
}

func getRandomColor(alpha ...float64) string {
	// 预定义一组颜色，确保不同用户有不同的颜色
	colors := []string{
		"255, 99, 132",   // 红色
		"54, 162, 235",   // 蓝色
		"255, 206, 86",   // 黄色
		"75, 192, 192",   // 青色
		"153, 102, 255",  // 紫色
		"255, 159, 64",   // 橙色
		"199, 199, 199",  // 灰色
		"83, 102, 255",   // 靛蓝色
		"255, 99, 255",   // 粉色
		"99, 255, 132",   // 绿色
	}
	
	// 使用时间作为种子来选择颜色，确保每次运行时颜色相对固定
	index := time.Now().UnixNano() % int64(len(colors))
	color := colors[index]
	
	if len(alpha) > 0 {
		return fmt.Sprintf("rgba(%s, %f)", color, alpha[0])
	}
	
	return fmt.Sprintf("rgb(%s)", color)
}