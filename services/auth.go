package services

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"ssh-manage/models"
	"ssh-manage/utils"

	_ "github.com/mattn/go-sqlite3"
)

// AuthenticateUser 验证用户身份
// 参数:
//   username - 用户名
//   password - 密码
// 返回:
//   *models.User - 用户信息
//   error - 错误信息
func AuthenticateUser(username, password string) (*models.User, error) {
	user, err := utils.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	
	if user.Password != password {
		return nil, nil // 密码错误
	}
	
	if !user.Active {
		return nil, nil // 用户未激活
	}
	
	return user, nil
}

// GetUserByID 根据ID获取用户信息
// 参数: id - 用户ID
// 返回: *models.User - 用户信息
func GetUserByID(id int) *models.User {
	user, err := utils.GetUserByID(id)
	if err != nil {
		log.Printf("Failed to get user by ID %d: %v", id, err)
		return nil
	}
	return user
}

// UpdateUser 更新用户信息
// 参数: user - 用户信息
// 返回: error - 错误信息
func UpdateUser(user *models.User) error {
	return utils.UpdateUser(user)
}

// GetAllUsers 获取所有用户
// 返回: []*models.User - 用户列表
func GetAllUsers() []*models.User {
	users, err := utils.GetAllUsers()
	if err != nil {
		log.Printf("Failed to get all users: %v", err)
		return []*models.User{}
	}
	return users
}

// AddUser 添加用户
// 参数: user - 用户信息
// 返回: error - 错误信息
func AddUser(user *models.User) error {
	return utils.AddUser(user)
}

// GetConnectionsByUserID 根据用户ID获取连接记录
// 参数: userID - 用户ID
// 返回: []*models.Connection - 连接记录列表
func GetConnectionsByUserID(userID int) []*models.Connection {
	connections, err := utils.GetConnectionsByUserID(userID)
	if err != nil {
		log.Printf("Failed to get connections by user ID %d: %v", userID, err)
		return []*models.Connection{}
	}
	return connections
}

// GetAllConnections 获取所有连接记录
// 返回: []*models.Connection - 连接记录列表
func GetAllConnections() []*models.Connection {
	connections, err := utils.GetAllConnections()
	if err != nil {
		log.Printf("Failed to get all connections: %v", err)
		return []*models.Connection{}
	}
	return connections
}

// GetTargetConnectionsByUserID 根据用户ID获取目标连接记录
// 参数: userID - 用户ID
// 返回: []*models.TargetConnection - 目标连接记录列表
func GetTargetConnectionsByUserID(userID int) []*models.TargetConnection {
	connections, err := utils.GetTargetConnectionsByUserID(userID)
	if err != nil {
		log.Printf("Failed to get target connections by user ID %d: %v", userID, err)
		return []*models.TargetConnection{}
	}
	return connections
}

// GetAllTargetConnections 获取所有目标连接记录
// 返回: []*models.TargetConnection - 目标连接记录列表
func GetAllTargetConnections() []*models.TargetConnection {
	connections, err := utils.GetAllTargetConnections()
	if err != nil {
		log.Printf("Failed to get all target connections: %v", err)
		return []*models.TargetConnection{}
	}
	return connections
}

// GetStatistics 获取统计信息
// 返回: map[string]interface{} - 统计信息
func GetStatistics() map[string]interface{} {
	stats, err := utils.GetStatistics()
	if err != nil {
		log.Printf("Failed to get statistics: %v", err)
		return map[string]interface{}{}
	}
	return stats
}

// GenerateRSAKey 生成RSA密钥对
// 返回: 
//   []byte - 私钥PEM格式数据
//   []byte - 公钥PEM格式数据
//   error - 错误信息
func GenerateRSAKey() ([]byte, []byte, error) {
	// 生成私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// 将私钥编码为PEM格式
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// 获取公钥并编码为PEM格式
	publicKey := &privateKey.PublicKey
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, nil, err
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	return pem.EncodeToMemory(privateKeyPEM), pem.EncodeToMemory(publicKeyPEM), nil
}