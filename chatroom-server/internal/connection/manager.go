package connection

import (
	"errors"
	"sync"

	"golang.org/x/net/websocket"
)

// Manager 连接管理器
type Manager struct {
	mu      sync.RWMutex
	clients map[string]*Client // connID -> Client
	users   map[string]*Client // username -> Client
}

// NewManager 创建连接管理器
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*Client),
		users:   make(map[string]*Client),
	}
}

// AddClient 添加连接
func (m *Manager) AddClient(ws *websocket.Conn) *Client {
	m.mu.Lock()
	defer m.mu.Unlock()

	client := NewClient(ws)
	m.clients[client.ID] = client

	// 启动发送协程
	go client.WritePump()

	return client
}

// RemoveClient 移除连接
func (m *Manager) RemoveClient(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok := m.clients[id]
	if !ok {
		return
	}

	// 清理用户映射
	if client.Username != "" {
		delete(m.users, client.Username)
	}

	// 关闭连接
	close(client.SendChan)
	client.Conn.Close()

	delete(m.clients, id)
}

// BindUser 绑定用户名到连接
func (m *Manager) BindUser(connID, username string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, ok := m.clients[connID]
	if !ok {
		return errors.New("connection not found")
	}

	client.Username = username
	m.users[username] = client

	return nil
}

// GetOnlineUsers 获取所有在线用户名
func (m *Manager) GetOnlineUsers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]string, 0, len(m.users))
	for username := range m.users {
		users = append(users, username)
	}
	return users
}

// Broadcast 广播消息给所有在线用户
func (m *Manager) Broadcast(data []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, client := range m.users {
		select {
		case client.SendChan <- data:
		default:
			// 通道满，跳过
		}
	}
}

// SendToUser 发送给特定用户
func (m *Manager) SendToUser(username string, data []byte) error {
	m.mu.RLock()
	client, ok := m.users[username]
	m.mu.RUnlock()

	if !ok {
		return errors.New("user not online")
	}

	select {
	case client.SendChan <- data:
		return nil
	default:
		return errors.New("send buffer full")
	}
}

// IsUserOnline 检查用户是否在线
func (m *Manager) IsUserOnline(username string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.users[username]
	return exists
}

// CloseAll 关闭所有连接（用于优雅关闭）
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, client := range m.clients {
		close(client.SendChan)
		client.Conn.Close()
	}

	m.clients = make(map[string]*Client)
	m.users = make(map[string]*Client)
}

// GetClientCount 获取连接数
func (m *Manager) GetClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.clients)
}
