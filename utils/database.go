package utils

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"ssh-manage/config"
	"ssh-manage/models"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

// InitDB 初始化数据库
func InitDB() error {
	cfg := config.Load()
	
	// 确保数据目录存在
	dir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// 打开数据库
	var err error
	db, err = sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		return err
	}
	
	// 设置连接池
	db.SetMaxOpenConns(1) // SQLite是文件数据库，限制为1个连接可避免并发问题
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // 永不关闭连接
	
	// 配置SQLite特定选项
	db.Exec("PRAGMA journal_mode = WAL;")
	db.Exec("PRAGMA synchronous = NORMAL;")
	db.Exec("PRAGMA cache_size = 1000000;")
	db.Exec("PRAGMA foreign_keys = ON;")
	
	// 测试连接
	if err := db.Ping(); err != nil {
		return err
	}
	
	// 创建表
	if err := createTables(); err != nil {
		return err
	}
	
	// 迁移表结构（添加新字段）
	if err := migrateTables(); err != nil {
		return err
	}
	
	log.Println("Database initialized successfully")
	return nil
}

// GetDB 获取数据库连接
func GetDB() *sql.DB {
	return db
}

// CloseDB 关闭数据库连接
func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// createTables 创建数据表
func createTables() error {
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// 创建用户表
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		created DATETIME DEFAULT CURRENT_TIMESTAMP,
		active BOOLEAN DEFAULT TRUE
	);`
	
	_, err = tx.Exec(userTable)
	if err != nil {
		return err
	}
	
	// 创建连接记录表（SSH连接）
	connectionTable := `
	CREATE TABLE IF NOT EXISTS connections (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		username TEXT NOT NULL,
		ip TEXT NOT NULL,
		connected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		disconnected_at DATETIME,
		session_id TEXT UNIQUE, -- 添加唯一约束
		FOREIGN KEY (user_id) REFERENCES users(id)
	);`
	
	_, err = tx.Exec(connectionTable)
	if err != nil {
		return err
	}
	
	// 创建目标连接记录表（每个SSH连接可能有多个目标连接）
	targetConnectionTable := `
	CREATE TABLE IF NOT EXISTS target_connections (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		connection_id INTEGER NOT NULL,
		target TEXT NOT NULL,
		connected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		disconnected_at DATETIME,
		bytes_up INTEGER DEFAULT 0,
		bytes_down INTEGER DEFAULT 0,
		FOREIGN KEY (connection_id) REFERENCES connections(id)
	);`
	
	_, err = tx.Exec(targetConnectionTable)
	if err != nil {
		return err
	}
	
	// 提交事务
	return tx.Commit()
}

// migrateTables 迁移表结构以支持新字段
func migrateTables() error {
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// 检查并添加target字段
	if err := addColumnIfNotExists(tx, "connections", "target", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	
	// 检查并添加bytes_up字段
	if err := addColumnIfNotExists(tx, "connections", "bytes_up", "INTEGER DEFAULT 0"); err != nil {
		return err
	}
	
	// 检查并添加bytes_down字段
	if err := addColumnIfNotExists(tx, "connections", "bytes_down", "INTEGER DEFAULT 0"); err != nil {
		return err
	}
	
	// 检查bytes_in字段是否存在，如果存在则将旧的bytes_in数据迁移到bytes_up（如果bytes_up是空的）
	if columnExists(tx, "connections", "bytes_in") {
		_, err = tx.Exec("UPDATE connections SET bytes_up = bytes_in WHERE bytes_up = 0")
		if err != nil {
			return err
		}
	}
	
	// 检查bytes_out字段是否存在，如果存在则将旧的bytes_out数据迁移到bytes_down（如果bytes_down是空的）
	if columnExists(tx, "connections", "bytes_out") {
		_, err = tx.Exec("UPDATE connections SET bytes_down = bytes_out WHERE bytes_down = 0")
		if err != nil {
			return err
		}
	}
	
	// 提交事务
	return tx.Commit()
}

// columnExists 检查表中是否存在指定字段
func columnExists(tx *sql.Tx, table, column string) bool {
	query := `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`
	var count int
	err := tx.QueryRow(query, table, column).Scan(&count)
	return err == nil && count > 0
}

// addColumnIfNotExists 检查字段是否存在，如果不存在则添加
func addColumnIfNotExists(tx *sql.Tx, table, column, definition string) error {
	// 检查字段是否已存在
	query := `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`
	var count int
	err := tx.QueryRow(query, table, column).Scan(&count)
	if err != nil {
		return err
	}
	
	// 如果字段不存在，则添加
	if count == 0 {
		_, err = tx.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition)
		if err != nil {
			return err
		}
		log.Printf("Added column %s to table %s", column, table)
	}
	
	return nil
}

// CreateDefaultUser 创建默认管理员用户
func CreateDefaultUser() error {
	db := GetDB()
	
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", "admin").Scan(&count)
	if err != nil {
		return err
	}
	
	if count == 0 {
		_, err = tx.Exec("INSERT INTO users (name, username, password, active) VALUES (?, ?, ?, ?)",
			"Admin User", "admin", "admin123", true)
		if err != nil {
			return err
		}
		log.Println("Default admin user created")
	}
	
	// 提交事务
	return tx.Commit()
}

// GetUserByUsername 根据用户名获取用户信息
func GetUserByUsername(username string) (*models.User, error) {
	db := GetDB()
	
	var user models.User
	var created string
	err := db.QueryRow("SELECT id, name, username, password, created, active FROM users WHERE username = ?", username).Scan(
		&user.ID, &user.Name, &user.Username, &user.Password, &created, &user.Active)
	
	if err != nil {
		return nil, err
	}
	
	// 解析时间
	user.Created, err = time.Parse("2006-01-02 15:04:05", created)
	if err != nil {
		// 尝试其他时间格式
		user.Created, err = time.Parse(time.RFC3339, created)
		if err != nil {
			return nil, err
		}
	}
	
	return &user, nil
}

// GetUserByID 根据ID获取用户信息
func GetUserByID(id int) (*models.User, error) {
	db := GetDB()
	
	var user models.User
	var created string
	err := db.QueryRow("SELECT id, name, username, password, created, active FROM users WHERE id = ?", id).Scan(
		&user.ID, &user.Name, &user.Username, &user.Password, &created, &user.Active)
	
	if err != nil {
		return nil, err
	}
	
	// 解析时间
	user.Created, err = time.Parse("2006-01-02 15:04:05", created)
	if err != nil {
		// 尝试其他时间格式
		user.Created, err = time.Parse(time.RFC3339, created)
		if err != nil {
			return nil, err
		}
	}
	
	return &user, nil
}

// UpdateUser 更新用户信息
func UpdateUser(user *models.User) error {
	db := GetDB()
	
	_, err := db.Exec("UPDATE users SET name = ?, username = ?, password = ?, active = ? WHERE id = ?",
		user.Name, user.Username, user.Password, user.Active, user.ID)
	
	return err
}

// GetAllUsers 获取所有用户
func GetAllUsers() ([]*models.User, error) {
	db := GetDB()
	
	rows, err := db.Query("SELECT id, name, username, password, created, active FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []*models.User
	for rows.Next() {
		var user models.User
		var created string
		err := rows.Scan(&user.ID, &user.Name, &user.Username, &user.Password, &created, &user.Active)
		if err != nil {
			return nil, err
		}
		
		// 解析时间
		user.Created, err = time.Parse("2006-01-02 15:04:05", created)
		if err != nil {
			// 尝试其他时间格式
			user.Created, err = time.Parse(time.RFC3339, created)
			if err != nil {
				return nil, err
			}
		}
		
		users = append(users, &user)
	}
	
	return users, nil
}

// AddUser 添加新用户
func AddUser(user *models.User) error {
	db := GetDB()
	
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// 检查用户是否已存在
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", user.Username).Scan(&count)
	if err != nil {
		return err
	}
	
	if count > 0 {
		// 提交事务并返回（用户已存在，不重复添加）
		return tx.Commit()
	}
	
	// 插入新用户
	_, err = tx.Exec("INSERT INTO users (name, username, password, active, created) VALUES (?, ?, ?, ?, ?)",
		user.Name, user.Username, user.Password, user.Active, user.Created.Format("2006-01-02 15:04:05"))
	if err != nil {
		return err
	}
	
	// 提交事务
	return tx.Commit()
}

// RecordConnection 记录SSH连接信息并返回数据库ID
func RecordConnection(conn *models.Connection) (int, error) {
	db := GetDB()
	
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	
	result, err := tx.Exec(`INSERT INTO connections (user_id, username, ip, connected_at, session_id) 
		VALUES (?, ?, ?, ?, ?)`,
		conn.UserID, conn.Username, conn.IP, conn.ConnectedAt.Format("2006-01-02 15:04:05"), conn.SessionID)
	if err != nil {
		return 0, err
	}
	
	// 获取插入的ID
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	
	// 提交事务
	err = tx.Commit()
	if err != nil {
		return 0, err
	}
	
	return int(id), nil
}

// GetConnectionBySessionID 根据会话ID获取连接信息
func GetConnectionBySessionID(sessionID string) (*models.Connection, error) {
	db := GetDB()
	
	row := db.QueryRow("SELECT id, user_id, username, ip, connected_at, disconnected_at, session_id FROM connections WHERE session_id = ?", sessionID)
	
	var conn models.Connection
	var connectedAtStr string
	var disconnectedAtStr *string
	err := row.Scan(&conn.ID, &conn.UserID, &conn.Username, &conn.IP, &connectedAtStr, &disconnectedAtStr, &conn.SessionID)
	if err != nil {
		return nil, err
	}
	
	// 解析时间
	conn.ConnectedAt, err = time.Parse("2006-01-02 15:04:05", connectedAtStr)
	if err != nil {
		conn.ConnectedAt, err = time.Parse(time.RFC3339, connectedAtStr)
		if err != nil {
			return nil, err
		}
	}
	
	if disconnectedAtStr != nil {
		disconnectedAt, err := time.Parse("2006-01-02 15:04:05", *disconnectedAtStr)
		if err != nil {
			disconnectedAt, err = time.Parse(time.RFC3339, *disconnectedAtStr)
			if err != nil {
				return nil, err
			}
		}
		conn.DisconnectedAt = &disconnectedAt
	}
	
	return &conn, nil
}

// RecordTargetConnection 记录目标连接信息
func RecordTargetConnection(targetConn *models.TargetConnection) (int, error) {
	db := GetDB()
	
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	
	result, err := tx.Exec(`INSERT INTO target_connections (connection_id, target, connected_at, bytes_up, bytes_down) 
		VALUES (?, ?, ?, ?, ?)`,
		targetConn.ConnectionID, targetConn.Target, targetConn.ConnectedAt.Format("2006-01-02 15:04:05"), targetConn.BytesUp, targetConn.BytesDown)
	if err != nil {
		return 0, err
	}
	
	// 获取插入的ID
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	
	// 提交事务
	err = tx.Commit()
	if err != nil {
		return 0, err
	}
	
	return int(id), nil
}

// UpdateTargetConnectionTraffic 更新目标连接的流量统计
func UpdateTargetConnectionTraffic(targetConnID int, bytesUp, bytesDown int64) error {
	db := GetDB()
	
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	_, err = tx.Exec("UPDATE target_connections SET bytes_up = ?, bytes_down = ? WHERE id = ?",
		bytesUp, bytesDown, targetConnID)
	if err != nil {
		return err
	}
	
	// 提交事务
	return tx.Commit()
}

// UpdateTargetConnectionDisconnectTime 更新目标连接的断开时间
func UpdateTargetConnectionDisconnectTime(targetConnID int, disconnectedAt time.Time) error {
	db := GetDB()
	
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	_, err = tx.Exec("UPDATE target_connections SET disconnected_at = ? WHERE id = ?",
		disconnectedAt.Format("2006-01-02 15:04:05"), targetConnID)
	if err != nil {
		return err
	}
	
	// 提交事务
	return tx.Commit()
}

// UpdateConnectionDisconnectTime 更新SSH连接的断开时间
func UpdateConnectionDisconnectTime(sessionID string, disconnectedAt time.Time) error {
	db := GetDB()
	
	// 开始事务
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	_, err = tx.Exec("UPDATE connections SET disconnected_at = ? WHERE session_id = ?",
		disconnectedAt.Format("2006-01-02 15:04:05"), sessionID)
	if err != nil {
		return err
	}
	
	// 提交事务
	return tx.Commit()
}

// GetConnectionsByUserID 根据用户ID获取连接记录
// 参数: userID - 用户ID
// 返回: 
//   []*models.Connection - 连接记录列表
//   error - 查询过程中的错误
func GetConnectionsByUserID(userID int) ([]*models.Connection, error) {
	db := GetDB()
	
	rows, err := db.Query("SELECT id, user_id, username, ip, connected_at, disconnected_at, session_id FROM connections WHERE user_id = ? ORDER BY connected_at DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var connections []*models.Connection
	for rows.Next() {
		var conn models.Connection
		var connectedAtStr string
		var disconnectedAtStr *string
		err := rows.Scan(&conn.ID, &conn.UserID, &conn.Username, &conn.IP, &connectedAtStr, &disconnectedAtStr, &conn.SessionID)
		if err != nil {
			return nil, err
		}
		
		// 解析时间
		conn.ConnectedAt, err = time.Parse("2006-01-02 15:04:05", connectedAtStr)
		if err != nil {
			// 尝试其他时间格式
			conn.ConnectedAt, err = time.Parse(time.RFC3339, connectedAtStr)
			if err != nil {
				return nil, err
			}
		}
		
		// 解析断开时间（可能为NULL）
		if disconnectedAtStr != nil {
			disconnectedAt, err := time.Parse("2006-01-02 15:04:05", *disconnectedAtStr)
			if err != nil {
				// 尝试其他时间格式
				disconnectedAt, err = time.Parse(time.RFC3339, *disconnectedAtStr)
				if err != nil {
					return nil, err
				}
			}
			conn.DisconnectedAt = &disconnectedAt
		}
		
		connections = append(connections, &conn)
	}
	
	return connections, nil
}

// GetAllConnections 获取所有连接记录
// 返回: 
//   []*models.Connection - 连接记录列表
//   error - 查询过程中的错误
func GetAllConnections() ([]*models.Connection, error) {
	db := GetDB()
	
	rows, err := db.Query("SELECT id, user_id, username, ip, connected_at, disconnected_at, session_id FROM connections ORDER BY connected_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var connections []*models.Connection
	for rows.Next() {
		var conn models.Connection
		var connectedAtStr string
		var disconnectedAtStr *string
		err := rows.Scan(&conn.ID, &conn.UserID, &conn.Username, &conn.IP, &connectedAtStr, &disconnectedAtStr, &conn.SessionID)
		if err != nil {
			return nil, err
		}
		
		// 解析时间
		conn.ConnectedAt, err = time.Parse("2006-01-02 15:04:05", connectedAtStr)
		if err != nil {
			// 尝试其他时间格式
			conn.ConnectedAt, err = time.Parse(time.RFC3339, connectedAtStr)
			if err != nil {
				return nil, err
			}
		}
		
		// 解析断开时间（可能为NULL）
		if disconnectedAtStr != nil {
			disconnectedAt, err := time.Parse("2006-01-02 15:04:05", *disconnectedAtStr)
			if err != nil {
				// 尝试其他时间格式
				disconnectedAt, err = time.Parse(time.RFC3339, *disconnectedAtStr)
				if err != nil {
					return nil, err
				}
			}
			conn.DisconnectedAt = &disconnectedAt
		}
		
		connections = append(connections, &conn)
	}
	
	return connections, nil
}

// GetTargetConnectionsByUserID 根据用户ID获取目标连接记录
// 参数: userID - 用户ID
// 返回: 
//   []*models.TargetConnection - 目标连接记录列表
//   error - 查询过程中的错误
func GetTargetConnectionsByUserID(userID int) ([]*models.TargetConnection, error) {
	db := GetDB()
	
	rows, err := db.Query(`
		SELECT tc.id, tc.connection_id, tc.target, tc.connected_at, tc.disconnected_at, tc.bytes_up, tc.bytes_down
		FROM target_connections tc
		JOIN connections c ON tc.connection_id = c.id
		WHERE c.user_id = ?
		ORDER BY tc.connected_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var targetConnections []*models.TargetConnection
	for rows.Next() {
		var targetConn models.TargetConnection
		var connectedAtStr string
		var disconnectedAtStr *string
		err := rows.Scan(&targetConn.ID, &targetConn.ConnectionID, &targetConn.Target, &connectedAtStr, &disconnectedAtStr, &targetConn.BytesUp, &targetConn.BytesDown)
		if err != nil {
			return nil, err
		}
		
		// 解析时间
		targetConn.ConnectedAt, err = time.Parse("2006-01-02 15:04:05", connectedAtStr)
		if err != nil {
			// 尝试其他时间格式
			targetConn.ConnectedAt, err = time.Parse(time.RFC3339, connectedAtStr)
			if err != nil {
				return nil, err
			}
		}
		
		// 解析断开时间（可能为NULL）
		if disconnectedAtStr != nil {
			disconnectedAt, err := time.Parse("2006-01-02 15:04:05", *disconnectedAtStr)
			if err != nil {
				// 尝试其他时间格式
				disconnectedAt, err = time.Parse(time.RFC3339, *disconnectedAtStr)
				if err != nil {
					return nil, err
				}
			}
			targetConn.DisconnectedAt = &disconnectedAt
		}
		
		targetConnections = append(targetConnections, &targetConn)
	}
	
	return targetConnections, nil
}

// GetAllTargetConnections 获取所有目标连接记录
// 返回: 
//   []*models.TargetConnection - 目标连接记录列表
//   error - 查询过程中的错误
func GetAllTargetConnections() ([]*models.TargetConnection, error) {
	db := GetDB()
	
	rows, err := db.Query(`
		SELECT id, connection_id, target, connected_at, disconnected_at, bytes_up, bytes_down
		FROM target_connections
		ORDER BY connected_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var targetConnections []*models.TargetConnection
	for rows.Next() {
		var targetConn models.TargetConnection
		var connectedAtStr string
		var disconnectedAtStr *string
		err := rows.Scan(&targetConn.ID, &targetConn.ConnectionID, &targetConn.Target, &connectedAtStr, &disconnectedAtStr, &targetConn.BytesUp, &targetConn.BytesDown)
		if err != nil {
			return nil, err
		}
		
		// 解析时间
		targetConn.ConnectedAt, err = time.Parse("2006-01-02 15:04:05", connectedAtStr)
		if err != nil {
			// 尝试其他时间格式
			targetConn.ConnectedAt, err = time.Parse(time.RFC3339, connectedAtStr)
			if err != nil {
				return nil, err
			}
		}
		
		// 解析断开时间（可能为NULL）
		if disconnectedAtStr != nil {
			disconnectedAt, err := time.Parse("2006-01-02 15:04:05", *disconnectedAtStr)
			if err != nil {
				// 尝试其他时间格式
				disconnectedAt, err = time.Parse(time.RFC3339, *disconnectedAtStr)
				if err != nil {
					return nil, err
				}
			}
			targetConn.DisconnectedAt = &disconnectedAt
		}
		
		targetConnections = append(targetConnections, &targetConn)
	}
	
	return targetConnections, nil
}

// GetStatistics 获取统计信息
// 返回: 
//   map[string]interface{} - 统计信息
//   error - 查询过程中的错误
func GetStatistics() (map[string]interface{}, error) {
	db := GetDB()
	
	stats := make(map[string]interface{})
	
	// 获取总连接数
	var totalConnections int
	err := db.QueryRow("SELECT COUNT(*) FROM connections").Scan(&totalConnections)
	if err != nil {
		return nil, err
	}
	stats["total_connections"] = totalConnections
	
	// 获取活跃用户数
	var activeUsers int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE active = 1").Scan(&activeUsers)
	if err != nil {
		return nil, err
	}
	stats["active_users"] = activeUsers
	
	// 获取总上行流量
	var totalTrafficUp int64
	err = db.QueryRow("SELECT COALESCE(SUM(bytes_up), 0) FROM target_connections").Scan(&totalTrafficUp)
	if err != nil {
		return nil, err
	}
	stats["total_traffic_up"] = totalTrafficUp
	
	// 获取总下行流量
	var totalTrafficDown int64
	err = db.QueryRow("SELECT COALESCE(SUM(bytes_down), 0) FROM target_connections").Scan(&totalTrafficDown)
	if err != nil {
		return nil, err
	}
	stats["total_traffic_down"] = totalTrafficDown
	
	return stats, nil
}