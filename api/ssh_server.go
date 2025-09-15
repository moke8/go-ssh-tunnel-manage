package api

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
	"ssh-manage/config"
	"ssh-manage/models"
	"ssh-manage/services"
	"ssh-manage/utils"

	"golang.org/x/crypto/ssh"
)

// 用于跟踪连接的结构体
type TrackedConnection struct {
	Connection *models.Connection
	UpdatedAt  time.Time
	mu         sync.Mutex
}

// 用于跟踪目标连接的结构体
type TrackedTargetConnection struct {
	TargetConnection *models.TargetConnection
	UpdatedAt        time.Time
	mu               sync.Mutex
}

// 存储所有活动连接的映射
var activeConnections = make(map[string]*TrackedConnection)
var activeTargetConnections = make(map[int]*TrackedTargetConnection)
var connectionsMutex sync.RWMutex

func StartSSHServer(cfg *config.Config) error {
	// 创建SSH服务器配置
	sshConfig := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			// 验证用户凭据
			user, err := services.AuthenticateUser(c.User(), string(password))
			if err != nil {
				log.Printf("Authentication failed for user %s: %v", c.User(), err)
				return nil, err
			}
			
			// 检查用户是否存在（认证是否成功）
			if user == nil {
				log.Printf("Authentication failed for user %s: invalid credentials", c.User())
				return nil, fmt.Errorf("invalid credentials")
			}
			
			log.Printf("Authentication successful for user %s from %s", c.User(), c.RemoteAddr())
			
			// 记录连接信息
			conn := &models.Connection{
				UserID:      user.ID,
				Username:    user.Username,
				IP:          c.RemoteAddr().String(),
				ConnectedAt: time.Now(),
				SessionID:   string(c.SessionID()),
			}
			
			// 记录连接到数据库并获取数据库ID
			connID, err := utils.RecordConnection(conn)
			if err != nil {
				log.Printf("Failed to record connection: %v", err)
				return nil, err // 认证失败，拒绝连接
			}
			
			// 更新连接对象的ID
			conn.ID = connID
			
			// 将连接添加到活动连接映射中
			connectionsMutex.Lock()
			activeConnections[conn.SessionID] = &TrackedConnection{
				Connection: conn,
				UpdatedAt:  time.Now(),
			}
			connectionsMutex.Unlock()
			
			return &ssh.Permissions{
				Extensions: map[string]string{
					"user": c.User(),
				},
			}, nil
		},
	}

	// 生成或加载固定的主机密钥
	privateKey, err := loadOrGenerateHostKey()
	if err != nil {
		return err
	}

	private, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return err
	}

	sshConfig.AddHostKey(private)

	// 启动定期更新数据库中流量统计的goroutine
	go updateTrafficStatsPeriodically()

	// 监听端口
	listener, err := net.Listen("tcp", ":"+cfg.SSHPort)
	if err != nil {
		return err
	}
	defer listener.Close()

	log.Printf("SSH Server listening on port %s", cfg.SSHPort)

	// 接受连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// 处理连接
		go handleConnection(conn, sshConfig)
	}
}

// 定期更新流量统计数据
func updateTrafficStatsPeriodically() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		updateTrafficStats()
	}
}

// 更新流量统计数据到数据库
func updateTrafficStats() {
	connectionsMutex.RLock()
	defer connectionsMutex.RUnlock()

	// 更新目标连接的流量统计
	for _, trackedTargetConn := range activeTargetConnections {
		trackedTargetConn.mu.Lock()
		targetConn := trackedTargetConn.TargetConnection
		trackedTargetConn.mu.Unlock()
		
		// 更新数据库中的流量统计
		err := utils.UpdateTargetConnectionTraffic(targetConn.ID, targetConn.BytesUp, targetConn.BytesDown)
		if err != nil {
			log.Printf("Failed to update traffic stats for target connection %d: %v", targetConn.ID, err)
		}
	}
}

func handleConnection(conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to handshake: %v", err)
		return
	}
	defer func() {
		// 连接关闭时更新断开时间
		sessionID := string(sshConn.SessionID())
		connectionsMutex.Lock()
		if trackedConn, exists := activeConnections[sessionID]; exists {
			disconnectedAt := time.Now()
			trackedConn.Connection.DisconnectedAt = &disconnectedAt
			// 更新断开连接时间
			utils.UpdateConnectionDisconnectTime(sessionID, disconnectedAt)
			// 从活动连接中移除
			delete(activeConnections, sessionID)
		}
		connectionsMutex.Unlock()
		sshConn.Close()
	}()

	log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())
	
	// 全局请求处理
	go handleGlobalRequests(reqs)
	
	// 通道处理
	for newChannel := range chans {
		go handleChannel(newChannel, string(sshConn.SessionID()), sshConn)
	}
}

// handleGlobalRequests 处理全局请求
func handleGlobalRequests(reqs <-chan *ssh.Request) {
	for req := range reqs {
		switch req.Type {
		case "tcpip-forward":
			handleTCPIPForward(req)
		case "cancel-tcpip-forward":
			handleCancelTCPIPForward(req)
		default:
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

// handleChannel 处理通道请求
func handleChannel(newChannel ssh.NewChannel, sessionID string, sshConn *ssh.ServerConn) {
	switch newChannel.ChannelType() {
	case "session":
		handleSessionChannel(newChannel, sessionID)
	case "direct-tcpip":
		handleDirectTCPIPChannel(newChannel, sessionID, sshConn)
	default:
		newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
		return
	}
}

// handleSessionChannel 处理会话通道
func handleSessionChannel(newChannel ssh.NewChannel, sessionID string) {
	channel, requests, err := newChannel.Accept()
	if err != nil {
		log.Printf("Could not accept session channel: %v", err)
		return
	}
	defer channel.Close()
	
	// 处理通道请求
	go func(in <-chan *ssh.Request) {
		for req := range in {
			ok := false
			switch req.Type {
			case "exec":
				ok = true
			case "shell":
				ok = true
			case "pty-req":
				ok = true
			case "env":
				ok = true
			}
			if req.WantReply {
				req.Reply(ok, nil)
			}
		}
	}(requests)
	
	// 简单的回显服务
	buf := make([]byte, 1024)
	for {
		n, err := channel.Read(buf)
		if err != nil {
			break
		}
		
		// 回显数据
		wn, err := channel.Write(buf[:n])
		if err != nil {
			break
		}
		
		_ = wn
	}
}

// handleDirectTCPIPChannel 处理直接TCP/IP通道
func handleDirectTCPIPChannel(newChannel ssh.NewChannel, sessionID string, sshConn *ssh.ServerConn) {
	// 解析目标地址信息
	extraData := newChannel.ExtraData()
	
	addr, port, err := parseDirectTCPIPData(extraData)
	if err != nil {
		log.Printf("Failed to parse direct-tcpip data: %v", err)
		newChannel.Reject(ssh.ConnectionFailed, "failed to parse target address")
		return
	}
	
	targetAddr := net.JoinHostPort(addr, port)
	
	// 检查目标地址是否被防火墙允许
	if !utils.IsAddressAllowed(targetAddr) {
		log.Printf("Connection to %s rejected by firewall rules", targetAddr)
		newChannel.Reject(ssh.Prohibited, "connection to target address is prohibited by firewall rules")
		return
	}
	
	// 接受通道
	channel, requests, err := newChannel.Accept()
	if err != nil {
		log.Printf("Could not accept direct-tcpip channel: %v", err)
		return
	}
	defer channel.Close()
	
	// 直接拒绝所有通道请求
	go ssh.DiscardRequests(requests)
	
	// 获取SSH连接ID
	var sshConnectionID int
	connectionsMutex.Lock()
	sshConnInfo, exists := activeConnections[sessionID]
	if !exists {
		connectionsMutex.Unlock()
		log.Printf("SSH connection not found in memory for session %s, trying to fetch from database", sessionID)
		
		// 尝试从数据库获取连接信息
		conn, dbErr := utils.GetConnectionBySessionID(sessionID)
		if dbErr != nil {
			log.Printf("SSH connection not found in database for session %s: %v", sessionID, dbErr)
			return
		}
		sshConnectionID = conn.ID
	} else {
		// 获取SSH连接的数据库ID
		sshConnectionID = sshConnInfo.Connection.ID
		connectionsMutex.Unlock()
	}
	
	// 创建目标连接记录
	targetConn := &models.TargetConnection{
		ConnectionID: sshConnectionID, // 使用已确认存在的SSH连接ID
		Target:       targetAddr,
		ConnectedAt:  time.Now(),
		BytesUp:      0,
		BytesDown:    0,
	}
	
	// 记录目标连接
	targetConnID, err := utils.RecordTargetConnection(targetConn)
	if err != nil {
		log.Printf("Failed to record target connection: %v", err)
		// 即使记录失败，也继续处理连接
		connectionsMutex.Lock()
		activeTargetConnections[targetConn.ID] = &TrackedTargetConnection{
			TargetConnection: targetConn,
			UpdatedAt:        time.Now(),
		}
		connectionsMutex.Unlock()
	} else {
		// 使用数据库返回的ID
		targetConn.ID = targetConnID
		
		// 将目标连接添加到活动连接映射中
		connectionsMutex.Lock()
		activeTargetConnections[targetConnID] = &TrackedTargetConnection{
			TargetConnection: targetConn,
			UpdatedAt:        time.Now(),
		}
		connectionsMutex.Unlock()
	}
	
	// 连接到目标地址
	targetConnNet, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("Failed to connect to target %s: %v", targetAddr, err)
		// 更新目标连接的断开时间
		connectionsMutex.Lock()
		if trackedTargetConn, exists := activeTargetConnections[targetConn.ID]; exists {
			disconnectedAt := time.Now()
			trackedTargetConn.TargetConnection.DisconnectedAt = &disconnectedAt
			// 更新断开连接时间
			utils.UpdateTargetConnectionDisconnectTime(trackedTargetConn.TargetConnection.ID, disconnectedAt)
			// 从活动连接中移除
			delete(activeTargetConnections, targetConn.ID)
		}
		connectionsMutex.Unlock()
		return
	}
	defer func() {
		// 确保只关闭一次
		var once sync.Once
		once.Do(func() {
			if targetConnNet != nil {
				targetConnNet.Close()
			}
		})
		
		// 更新目标连接的断开时间
		connectionsMutex.Lock()
		if trackedTargetConn, exists := activeTargetConnections[targetConn.ID]; exists {
			disconnectedAt := time.Now()
			trackedTargetConn.TargetConnection.DisconnectedAt = &disconnectedAt
			// 更新断开连接时间
			utils.UpdateTargetConnectionDisconnectTime(trackedTargetConn.TargetConnection.ID, disconnectedAt)
			// 从活动连接中移除
			delete(activeTargetConnections, targetConn.ID)
		}
		connectionsMutex.Unlock()
	}()
	
	log.Printf("Established direct-tcpip connection to %s", targetAddr)
	
	// 双向复制数据并统计流量
	var wg sync.WaitGroup
	wg.Add(2)
	
	// 从SSH通道复制到目标连接
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, err := channel.Read(buf)
			if n > 0 {
				// 更新上行流量统计（从客户端到目标）
				updateTargetTraffic(targetConn.ID, int64(n), 0)
				
				wn, writeErr := targetConnNet.Write(buf[:n])
				if writeErr != nil {
					break
				}
				
				// 确保写入的字节数与读取的字节数一致
				if wn != n {
					log.Printf("Expected to write %d bytes, wrote %d bytes", n, wn)
					break
				}
			}
			if err != nil {
				break
			}
		}
	}()
	
	// 从目标连接复制到SSH通道
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, err := targetConnNet.Read(buf)
			if n > 0 {
				// 更新下行流量统计（从目标到客户端）
				updateTargetTraffic(targetConn.ID, 0, int64(n))
				
				wn, writeErr := channel.Write(buf[:n])
				if writeErr != nil {
					break
				}
				
				// 确保写入的字节数与读取的字节数一致
				if wn != n {
					log.Printf("Expected to write %d bytes, wrote %d bytes", n, wn)
					break
				}
			}
			if err != nil {
				break
			}
		}
	}()
	
	wg.Wait()
	log.Printf("Closed direct-tcpip connection to %s", targetAddr)
}

// updateTargetTraffic 更新指定目标连接的流量统计
func updateTargetTraffic(targetConnID int, bytesUp, bytesDown int64) {
	connectionsMutex.RLock()
	trackedTargetConn, exists := activeTargetConnections[targetConnID]
	connectionsMutex.RUnlock()
	
	if exists {
		trackedTargetConn.mu.Lock()
		trackedTargetConn.TargetConnection.BytesUp += bytesUp
		trackedTargetConn.TargetConnection.BytesDown += bytesDown
		trackedTargetConn.UpdatedAt = time.Now()
		trackedTargetConn.mu.Unlock()
	}
}

// handleTCPIPForward 处理TCP/IP转发请求
func handleTCPIPForward(req *ssh.Request) {
	// 解析请求数据
	payload := req.Payload
	
	// 读取地址和端口
	r := bytes.NewReader(payload)
	
	// 读取地址长度
	addrLenBytes := make([]byte, 4)
	_, err := r.Read(addrLenBytes)
	if err != nil {
		if req.WantReply {
			req.Reply(false, nil)
		}
		return
	}
	
	addrLen := binary.BigEndian.Uint32(addrLenBytes)
	
	// 读取地址
	addrBytes := make([]byte, addrLen)
	_, err = r.Read(addrBytes)
	if err != nil {
		if req.WantReply {
			req.Reply(false, nil)
		}
		return
	}
	
	// 读取端口
	portBytes := make([]byte, 4)
	_, err = r.Read(portBytes)
	if err != nil {
		if req.WantReply {
			req.Reply(false, nil)
		}
		return
	}
	
	port := binary.BigEndian.Uint32(portBytes)
	
	log.Printf("TCP/IP forward request for %s:%d", string(addrBytes), port)
	
	if req.WantReply {
		// 返回绑定的端口（这里简化处理，返回请求的端口）
		replyPort := make([]byte, 4)
		binary.BigEndian.PutUint32(replyPort, port)
		req.Reply(true, replyPort)
	}
	
	log.Printf("Accepted tcpip-forward request for %s:%d", string(addrBytes), port)
}

// handleCancelTCPIPForward 处理取消TCP/IP转发请求
func handleCancelTCPIPForward(req *ssh.Request) {
	if req.WantReply {
		req.Reply(true, nil)
	}
	log.Printf("Accepted cancel-tcpip-forward request")
}

// parseDirectTCPIPData 解析direct-tcpip通道的额外数据
func parseDirectTCPIPData(data []byte) (addr, port string, err error) {
	if len(data) < 8 {
		return "", "", io.ErrUnexpectedEOF
	}
	
	// 解析目标地址
	addrLen := binary.BigEndian.Uint32(data[:4])
	if uint32(len(data)) < 4+addrLen+4 {
		return "", "", io.ErrUnexpectedEOF
	}
	addr = string(data[4 : 4+addrLen])
	
	// 解析目标端口
	portNum := binary.BigEndian.Uint32(data[4+addrLen : 4+addrLen+4])
	port = strconv.FormatUint(uint64(portNum), 10)
	
	return addr, port, nil
}

// loadOrGenerateHostKey 加载或生成主机密钥
func loadOrGenerateHostKey() ([]byte, error) {
	// 确保目录存在
	keyDir := "data"
	os.MkdirAll(keyDir, 0755)
	
	keyPath := filepath.Join(keyDir, "host_key")
	
	// 检查主机密钥文件是否存在
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		// 生成新的RSA密钥对
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}

		// 将私钥编码为PEM格式
		privateKeyPEM := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		}

		// 保存私钥到文件
		privateKeyFile, err := os.Create(keyPath)
		if err != nil {
			return nil, err
		}
		defer privateKeyFile.Close()

		if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
			return nil, err
		}
		
		log.Printf("Generated and saved new host key to %s", keyPath)
		
		return pem.EncodeToMemory(privateKeyPEM), nil
	} else {
		// 从文件加载私钥
		privateKeyBytes, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, err
		}
		
		log.Printf("Loaded existing host key from %s", keyPath)
		
		return privateKeyBytes, nil
	}
}