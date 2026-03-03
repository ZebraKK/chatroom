package user

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/xiaowyu/chatroom-server/internal/storage"
)

// Manager 用户管理器
type Manager struct {
	mu    sync.RWMutex
	users map[string]*User // username -> User
}

// NewManager 创建用户管理器
func NewManager() *Manager {
	return &Manager{
		users: make(map[string]*User),
	}
}

// Register 注册用户（处理用户名冲突）
func (m *Manager) Register(username, signingKey, encryptionKey, algorithm string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 查找可用用户名
	finalUsername := username
	counter := 1
	for {
		if _, exists := m.users[finalUsername]; !exists {
			break
		}
		finalUsername = fmt.Sprintf("%s_%d", username, counter)
		counter++
	}

	// 创建用户
	user := &User{
		Username:      finalUsername,
		SigningKey:    signingKey,
		EncryptionKey: encryptionKey,
		Algorithm:     algorithm,
		RegisteredAt:  time.Now().Unix(),
	}

	m.users[finalUsername] = user

	return finalUsername, nil
}

// GetUser 获取用户
func (m *Manager) GetUser(username string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.users[username]
	if !ok {
		return nil, errors.New("user not found")
	}

	return user, nil
}

// GetPublicKeys 获取用户的公钥（新增方法，用于签名验证）
func (m *Manager) GetPublicKeys(username string) (signingKey ed25519.PublicKey, encryptKey [32]byte, err error) {
	user, err := m.GetUser(username)
	if err != nil {
		return nil, [32]byte{}, err
	}

	signingKeyBytes, err := user.GetSigningKeyBytes()
	if err != nil {
		return nil, [32]byte{}, err
	}

	encryptKeyBytes, err := user.GetEncryptionKeyBytes()
	if err != nil {
		return nil, [32]byte{}, err
	}

	return signingKeyBytes, encryptKeyBytes, nil
}

// GetMultiplePublicKeys 批量获取用户公钥
func (m *Manager) GetMultiplePublicKeys(usernames []string) map[string]*User {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*User)
	for _, username := range usernames {
		if user, ok := m.users[username]; ok {
			result[username] = user
		}
	}

	return result
}

// GetAllUsers 获取所有用户
func (m *Manager) GetAllUsers() []*User {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]*User, 0, len(m.users))
	for _, user := range m.users {
		users = append(users, user)
	}

	return users
}

// Load 从存储加载用户数据
func (m *Manager) Load(s storage.Storage) error {
	storageUsers, err := s.LoadUsers()
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, su := range storageUsers {
		user := &User{
			Username:      su.Username,
			SigningKey:    su.SigningKey,
			EncryptionKey: su.EncryptionKey,
			Algorithm:     su.Algorithm,
			RegisteredAt:  su.RegisteredAt,
		}
		m.users[user.Username] = user
	}

	return nil
}

// Save 保存用户数据到存储
func (m *Manager) Save(s storage.Storage) error {
	m.mu.RLock()
	users := m.GetAllUsers()
	m.mu.RUnlock()

	// 转换为存储格式
	storageUsers := make([]*storage.User, len(users))
	for i, u := range users {
		storageUsers[i] = &storage.User{
			Username:      u.Username,
			SigningKey:    u.SigningKey,
			EncryptionKey: u.EncryptionKey,
			Algorithm:     u.Algorithm,
			RegisteredAt:  u.RegisteredAt,
		}
	}

	return s.SaveUsers(storageUsers)
}

// UserExists 检查用户是否存在
func (m *Manager) UserExists(username string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.users[username]
	return exists
}
